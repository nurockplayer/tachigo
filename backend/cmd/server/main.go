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
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"

	_ "github.com/tachigo/tachigo/docs"
	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/database"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/router"
	"github.com/tachigo/tachigo/internal/services"
)

func main() {
	// Load .env (ignore error in production where env is set externally)
	_ = godotenv.Load()

	cfg := config.Load()

	db := database.Connect(cfg.Database.DSN)

	// Create custom ENUM types before AutoMigrate (GORM cannot create them automatically).
	if err := db.Exec(`
		DO $$ BEGIN
			CREATE TYPE user_role AS ENUM ('viewer', 'streamer', 'admin');
		EXCEPTION WHEN duplicate_object THEN NULL;
		END $$;
	`).Error; err != nil {
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
	); err != nil {
		log.Fatalf("migration failed: %v", err)
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

	// CORS origins from env, default to localhost for dev
	originsEnv := os.Getenv("ALLOWED_ORIGINS")
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:5173"}
	if originsEnv != "" {
		allowedOrigins = strings.Split(originsEnv, ",")
	}

	r := router.New(authSvc, userSvc, addrSvc, extSvc, emailAuthSvc, watchSvc, channelConfigSvc, pointsSvc, streamerSvc, allowedOrigins)

	addr := ":" + cfg.Server.Port
	log.Printf("server starting on %s (env=%s)", addr, cfg.Server.Env)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
