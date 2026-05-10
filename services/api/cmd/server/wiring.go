package main

import (
	"context"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/router"
	"github.com/tachigo/tachigo/internal/services"
	"gorm.io/gorm"
)

func wire(db *gorm.DB, cfg *config.Config, ctx context.Context) *gin.Engine {
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
	var ethClient *ethclient.Client
	if cfg.Contract.TachiContractAddress != "" && cfg.Contract.SepoliaSignerKey != "" {
		var err error
		dialCtx, dialCancel := context.WithTimeout(ctx, 10*time.Second)
		ethClient, err = ethclient.DialContext(dialCtx, cfg.Contract.RPCEndpoint)
		dialCancel()
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
	oauthTokenSecret := cfg.OAuth.TokenEncryptionKey
	if oauthTokenSecret == "" {
		oauthTokenSecret = cfg.JWT.RefreshSecret
	}
	raffleSvc := services.NewRaffleService(db, cfg.OAuth.Twitch.ClientID, cfg.App.FrontendURL, mailer, oauthTokenSecret)
	services.NewRaffleScheduler(raffleSvc).Start(ctx)
	agencyH := handlers.NewAgencyHandler(agencySvc, emailAuthSvc)

	return router.New(
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
		cfg.Server.AllowedOrigins,
		router.InternalRouterConfig{DB: db, Config: cfg},
	)
}
