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
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"

	_ "github.com/tachigo/tachigo/docs"
	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/database"
	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/router"
	"github.com/tachigo/tachigo/internal/services"
	"gorm.io/gorm"
)

const defaultSepoliaRPCURL = "https://ethereum-sepolia-rpc.publicnode.com"

func main() {
	// Load .env (ignore error in production where env is set externally)
	_ = godotenv.Load()

	cfg := config.Load()
	if config.ShouldValidateProductionSecrets(cfg) {
		if err := config.ValidateProductionSecrets(cfg); err != nil {
			log.Fatalf("invalid secrets: %v", err)
		}
	}

	db := database.Connect(cfg.Database.DSN)

	if err := hashLegacyRaffleClaimTokens(db); err != nil {
		log.Fatalf("failed to hash existing claim tokens: %v", err)
	}

	// Wire services
	authSvc := services.NewAuthService(db, cfg)
	userSvc := services.NewUserService(db)
	addrSvc := services.NewAddressService(db)
	mailer := services.NewMailer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
	emailAuthSvc := services.NewEmailAuthService(db, cfg, mailer)
	watchSvc := services.NewWatchService(db)
	channelConfigSvc := services.NewChannelConfigService(db)
	pointsSvc := services.NewPointsService(db, watchSvc)
	extSvc := services.NewExtensionService(db, cfg, authSvc, pointsSvc)
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
	tachiyaClient := services.NewTachiyaHTTPClient(cfg.Internal.TachiyaBaseURL, cfg.Internal.TachiyaSharedSecret)
	spendSvc := services.NewSpendService(db, cfg.Contract, ethClient, tachiyaClient)
	if cfg.Server.Env == "production" && cfg.OAuth.Twitch.ClientID == "" {
		log.Fatal("TWITCH_CLIENT_ID is required in production for raffle snapshot sync")
	}
	raffleSvc := services.NewRaffleService(db, cfg.OAuth.Twitch.ClientID, cfg.App.FrontendURL, mailer)
	// Tie scheduler lifetime to server shutdown signals so the goroutine exits cleanly.
	serverCtx, serverStop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer serverStop()
	services.NewRaffleScheduler(raffleSvc).Start(serverCtx)
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

func hashLegacyRaffleClaimTokens(db *gorm.DB) error {
	// claim_token was previously a raw UUIDv7 (36 chars); it now stores the
	// SHA-256 hex digest (64 chars). This idempotent repair is data-only.
	return db.Exec(`
		UPDATE raffle_draws
		SET claim_token = encode(sha256(claim_token::bytea), 'hex')
		WHERE length(claim_token) = 36
	`).Error
}
