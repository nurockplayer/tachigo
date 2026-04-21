// @title           tachigo API
// @version         1.0
// @description     Backend API for tachigo — Twitch extension + Web3 rewards platform
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Enter: Bearer {access_token}
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/joho/godotenv"

	_ "github.com/tachigo/tachigo/docs"
	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/database"
	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/router"
	"github.com/tachigo/tachigo/internal/services"
)

const defaultSepoliaRPCURL = "https://ethereum-sepolia-rpc.publicnode.com"

func main() {
	// Load .env (ignore error in production where env is set externally)
	_ = godotenv.Load()

	cfg := config.Load()

	db := database.Connect(cfg.Database.DSN)

	// Create custom ENUM types before AutoMigrate (GORM cannot create them automatically).
	// NOTE: keep in sync with models.UserRole constants in internal/models/user.go.
	// 'agency' was added in refs #99; if adding new roles, update this list.
	if err := initializeUserRoleEnum(func(query string) error {
		return db.Exec(query).Error
	}); err != nil {
		log.Fatalf("failed to create user_role enum: %v", err)
	}

	// Auto-migrate all models
	if err := db.AutoMigrate(
		&models.User{},
		&models.AuthProvider{},
		&models.ShippingAddress{},
		&models.RefreshToken{},
		&models.Web3Nonce{},
		&models.EmailVerification{},
		&models.PasswordReset{},
		// Points & watch-time
		&models.Streamer{},
		&models.ChannelConfig{},
		&models.PointsLedger{},
		&models.PointsTransaction{},
		&models.WatchSession{},
		&models.WatchTimeStat{},
		&models.BroadcastTimeStat{},
		&models.BroadcastTimeLog{},
		// Tachi token balance — refs #103
		&models.TachiBalance{},
		// Agency management — refs #99
		&models.AgencyStreamer{},
		// Raffle system — refs #227
		&models.Raffle{},
		&models.RaffleEntry{},
		&models.RaffleDraw{},
		&models.RaffleClaim{},
	); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// FK constraint on tachi_balances.user_id — GORM AutoMigrate does not create FK
	// constraints without an explicit association field, so we add it manually (idempotent).
	if err := db.Exec(`
		DO $$ BEGIN
			ALTER TABLE tachi_balances ADD CONSTRAINT fk_tachi_balances_user_id
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
		EXCEPTION WHEN duplicate_object THEN NULL;
		END $$;
	`).Error; err != nil {
		log.Fatalf("failed to create tachi_balances FK: %v", err)
	}

	// Partial unique index: only one active session per (user_id, channel_id).
	// GORM AutoMigrate does not support partial indexes via struct tags, so we
	// create it manually with CREATE INDEX IF NOT EXISTS (idempotent).
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_watch_sessions_active_user_channel
		ON watch_sessions (user_id, channel_id)
		WHERE is_active = true
	`).Error; err != nil {
		log.Fatalf("failed to create partial index: %v", err)
	}
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_points_ledgers_user_channel
		ON points_ledgers (user_id, channel_id)
	`).Error; err != nil {
		log.Fatalf("failed to create points ledger index: %v", err)
	}
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_streamers_user_channel
		ON streamers (user_id, channel_id)
	`).Error; err != nil {
		log.Fatalf("failed to create streamer index: %v", err)
	}
	if err := applyStreamerAgencyMigration(db); err != nil {
		log.Fatalf("failed to run migration 008: %v", err)
	}
	// Wire services
	authSvc := services.NewAuthService(db, cfg)
	userSvc := services.NewUserService(db)
	addrSvc := services.NewAddressService(db)
	extSvc := services.NewExtensionService(db, cfg, authSvc)
	mailer := services.NewMailer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
	emailAuthSvc := services.NewEmailAuthService(db, cfg, mailer)
	watchSvc := services.NewWatchService(db)
	channelConfigSvc := services.NewChannelConfigService(db)
	pointsSvc := services.NewPointsService(db, watchSvc)
	streamerSvc := services.NewStreamerService(db, pointsSvc)
	agencySvc := services.NewAgencyService(db)
	airdropSvc := services.NewAirdropService(db, pointsSvc, channelConfigSvc)
	// TODO: move Sepolia RPC URL into config.Contract.RPCEndpoint once config schema is extended.
	var ethClient *ethclient.Client
	if cfg.Contract.TachiContractAddress != "" && cfg.Contract.SepoliaSignerKey != "" {
		var err error
		dialCtx, dialCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dialCancel()
		ethClient, err = ethclient.DialContext(dialCtx, defaultSepoliaRPCURL)
		if err != nil {
			log.Printf("warning: failed to connect Sepolia RPC: %v", err)
			ethClient = nil
		}
	}
	claimSvc := services.NewClaimService(db, cfg.Contract, ethClient)
	spendSvc := services.NewSpendService(db, cfg.Contract, ethClient)
	raffleSvc := services.NewRaffleService(db, cfg.OAuth.Twitch.ClientID)
	agencyH := handlers.NewAgencyHandler(agencySvc, emailAuthSvc)

	// CORS origins from env, default to localhost for dev
	originsEnv := os.Getenv("ALLOWED_ORIGINS")
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:5173"}
	if originsEnv != "" {
		allowedOrigins = strings.Split(originsEnv, ",")
	}

	r := router.New(
		authSvc,
		userSvc,
		addrSvc,
		extSvc,
		emailAuthSvc,
		watchSvc,
		channelConfigSvc,
		pointsSvc,
		airdropSvc,
		streamerSvc,
		agencySvc,
		claimSvc,
		spendSvc,
		raffleSvc,
		agencyH,
		allowedOrigins,
		router.InternalRouterConfig{DB: db, Config: cfg},
	)

	addr := ":" + cfg.Server.Port
	log.Printf("server starting on %s (env=%s)", addr, cfg.Server.Env)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func initializeUserRoleEnum(exec func(query string) error) error {
	if err := exec(`CREATE TYPE user_role AS ENUM ('viewer', 'streamer', 'agency', 'admin')`); err != nil {
		if !isDuplicateObject(err) {
			return err
		}
	}

	if err := exec(`ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'agency'`); err != nil {
		return err
	}

	return nil
}

func isDuplicateObject(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "42710"
	}
	return false
}
