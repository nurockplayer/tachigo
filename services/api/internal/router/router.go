package router

import (
	"log"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

type InternalRouterConfig struct {
	DB     *gorm.DB
	Config *config.Config
}

func New(
	authSvc *services.AuthService,
	userSvc *services.UserService,
	addrSvc *services.AddressService,
	extSvc *services.ExtensionService,
	emailAuthSvc *services.EmailAuthService,
	watchSvc *services.WatchService,
	channelConfigSvc *services.ChannelConfigService,
	pointsSvc *services.PointsService,
	airdropSvc *services.AirdropService,
	streamerSvc *services.StreamerService,
	agencySvc *services.AgencyService,
	claimSvc *services.ClaimService,
	spendSvc *services.SpendService,
	raffleSvc *services.RaffleService,
	agencyHandler *handlers.AgencyHandler,
	allowedOrigins []string,
	internalRouterConfig ...InternalRouterConfig,
) *gin.Engine {
	var cfg *config.Config
	if len(internalRouterConfig) > 0 {
		cfg = internalRouterConfig[0].Config
	}

	if cfg != nil && cfg.Server.GinMode != "" {
		gin.SetMode(cfg.Server.GinMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.CORS(allowedOrigins))

	if cfg != nil {
		if err := r.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
			log.Printf("warning: SetTrustedProxies: %v", err)
		}
	}

	authH := handlers.NewAuthHandler(authSvc, cfg).WithEmailAuth(emailAuthSvc)
	userH := handlers.NewUserHandler(userSvc)
	addrH := handlers.NewAddressHandler(addrSvc)
	extH := handlers.NewExtensionHandler(extSvc)
	emailH := handlers.NewEmailAuthHandler(emailAuthSvc)
	watchH := handlers.NewWatchHandler(watchSvc, pointsSvc)
	channelConfigH := handlers.NewChannelConfigHandler(channelConfigSvc, streamerSvc)
	pointsH := handlers.NewPointsHandler(pointsSvc)
	streamerH := handlers.NewStreamerHandler(streamerSvc)
	claimH := handlers.NewClaimHandler(claimSvc)
	spendH := handlers.NewSpendHandler(spendSvc)
	airdropH := handlers.NewAirdropHandler(airdropSvc, agencySvc, streamerSvc)
	raffleH := handlers.NewRaffleHandler(raffleSvc)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	enableSwagger := cfg == nil || cfg.Server.EnableSwagger
	if enableSwagger {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	v1 := r.Group("/api/v1")

	// ── Public auth endpoints ─────────────────────────────────────────────
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authH.Register)
		auth.POST("/login", authH.Login)
		auth.POST("/refresh", authH.Refresh)
		auth.POST("/logout", authH.Logout)

		// Twitch OAuth
		auth.GET("/twitch", authH.TwitchLogin)
		auth.GET("/twitch/callback", authH.TwitchCallback)

		// Google OAuth
		auth.GET("/google", authH.GoogleLogin)
		auth.GET("/google/callback", authH.GoogleCallback)

		// Web3 / SIWE
		auth.POST("/web3/nonce", authH.Web3Nonce)
		auth.POST("/web3/verify", authH.Web3Verify)

		// Email verification (confirm is public so users can click link without login)
		auth.POST("/verify-email/confirm", emailH.ConfirmVerification)

		// Password reset (public)
		auth.POST("/forgot-password", emailH.ForgotPassword)
		auth.POST("/reset-password", emailH.ResetPassword)
	}

	// ── Claim endpoints ───────────────────────────────────────────────────
	// GET is public; POST requires the winner's JWT.
	v1.GET("/claim/:token", raffleH.GetClaim)
	claimAuth := v1.Group("/claim")
	claimAuth.Use(middleware.JWTAuth(authSvc))
	{
		claimAuth.POST("/:token", raffleH.SubmitClaim)
	}

	// ── Twitch Extension endpoints ────────────────────────────────────────
	ext := v1.Group("/extension")
	{
		ext.POST("/auth/login", extH.Login)
		ext.POST("/t-point/complete", extH.TPointComplete)
		ext.POST("/bits/complete", extH.BitsComplete) // deprecated alias

		// Raffle result — public read (no auth required)
		ext.GET("/raffles/:id/result", raffleH.GetResult)

		// Watch-time points (requires tachigo JWT — viewer must log in first)
		watch := ext.Group("/watch")
		watch.Use(middleware.JWTAuth(authSvc))
		{
			watch.POST("/start", watchH.StartSession)
			watch.POST("/heartbeat", watchH.Heartbeat)
			watch.POST("/click", watchH.Click)
			watch.POST("/end", watchH.EndSession)
			watch.GET("/balance", watchH.GetBalance)
		}
	}

	// ── Authenticated routes ──────────────────────────────────────────────
	protected := v1.Group("/")
	protected.Use(middleware.JWTAuth(authSvc))
	{
		// Points balance
		protected.GET("users/me/points", pointsH.GetBalance)
		protected.GET("users/me/points/history", pointsH.GetHistory)

		// T-Point → $TACHI claim
		protected.POST("users/me/points/claim", claimH.Claim)
		protected.GET("users/me/tachi/balance", claimH.GetTachiBalance)

		// $TACHI spend (burn)
		protected.POST("spend/redeem", spendH.Redeem)

		// User profile
		protected.GET("users/me", userH.Me)
		protected.PUT("users/me", userH.UpdateMe)
		protected.POST("users/me/wallet", userH.LinkWallet)
		protected.GET("users/me/providers", userH.ListProviders)
		protected.DELETE("auth/providers/:provider", authH.UnlinkProvider)

		// Email verification send/resend (authenticated)
		protected.POST("auth/verify-email/send", emailH.SendVerification)

		// Shipping addresses
		addresses := protected.Group("users/me/addresses")
		{
			addresses.GET("", addrH.List)
			addresses.POST("", addrH.Create)
			addresses.PUT("/:id", addrH.Update)
			addresses.DELETE("/:id", addrH.Delete)
			addresses.PUT("/:id/default", addrH.SetDefault)
		}
	}

	dashboard := v1.Group("/dashboard")
	dashboard.Use(middleware.JWTAuth(authSvc))
	dashboard.Use(middleware.RequireRole(models.RoleAdmin, models.RoleStreamer, models.RoleAgency))
	{
		dashboard.POST("/streamers", middleware.RequireRole(models.RoleAdmin), streamerH.Create)
		dashboard.GET("/streamers", middleware.RequireRole(models.RoleAgency, models.RoleAdmin), streamerH.List)
		dashboard.GET("/streamers/:streamer_id/stats",
			middleware.RequireRole(models.RoleStreamer, models.RoleAgency, models.RoleAdmin),
			streamerH.GetStats)
		dashboard.POST("/streamers/register",
			middleware.RequireRole(models.RoleStreamer),
			streamerH.Register)
		dashboard.GET("/streamers/channels",
			middleware.RequireRole(models.RoleStreamer),
			streamerH.ListChannels)
		dashboard.GET("/channels/:channel_id/stats",
			middleware.RequireRole(models.RoleAdmin, models.RoleStreamer),
			streamerH.GetChannelStats)
		dashboard.GET("/channels/:channel_id/config",
			middleware.RequireRole(models.RoleAdmin, models.RoleStreamer, models.RoleAgency),
			channelConfigH.GetChannelConfig)
		dashboard.PUT("/channels/:channel_id/config",
			middleware.RequireRole(models.RoleAdmin, models.RoleStreamer),
			channelConfigH.UpdateChannelConfig)

		// Raffle management (streamer only)
		dashboard.POST("/raffles",
			middleware.RequireRole(models.RoleStreamer),
			raffleH.Create)
		dashboard.GET("/raffles",
			middleware.RequireRole(models.RoleStreamer),
			raffleH.List)
		dashboard.GET("/raffles/:id",
			middleware.RequireRole(models.RoleStreamer),
			raffleH.Get)
		dashboard.POST("/raffles/:id/entries/import-csv",
			middleware.RequireRole(models.RoleStreamer),
			raffleH.ImportCSV)
		dashboard.POST("/raffles/:id/draws",
			middleware.RequireRole(models.RoleStreamer),
			raffleH.DrawNext)
		dashboard.GET("/raffles/:id/draws",
			middleware.RequireRole(models.RoleStreamer),
			raffleH.ListDraws)
		dashboard.POST("/raffles/:id/complete",
			middleware.RequireRole(models.RoleStreamer),
			raffleH.Complete)
		dashboard.PATCH("/raffles/:id/discord-webhook",
			middleware.RequireRole(models.RoleStreamer),
			raffleH.SetDiscordWebhook)
		dashboard.POST("/raffles/:id/snapshot",
			middleware.RequireRole(models.RoleStreamer),
			raffleH.Snapshot)
	}

	dashboardAirdrop := v1.Group("/dashboard/channels/:channel_id")
	dashboardAirdrop.Use(middleware.JWTAuth(authSvc))
	dashboardAirdrop.Use(middleware.RequireRole(models.RoleAdmin, models.RoleStreamer, models.RoleAgency))
	{
		dashboardAirdrop.POST("/airdrop", airdropH.Airdrop)
	}

	if len(internalRouterConfig) > 0 &&
		internalRouterConfig[0].DB != nil &&
		internalRouterConfig[0].Config != nil &&
		internalRouterConfig[0].Config.Internal.TachiyaSharedSecret != "" {
		internalPointsH := handlers.NewInternalPointsHandler(internalRouterConfig[0].DB)
		internal := v1.Group("/internal/tachiya")
		internal.Use(middleware.TachiyaInternalAuth(internalRouterConfig[0].Config))
		{
			internal.GET("/users/points/balance", internalPointsH.GetUserPointsBalance)
		}
	}

	// ── Agency management ─────────────────────────────────────────────────
	// POST /agencies — admin only
	agencies := v1.Group("/agencies")
	agencies.Use(middleware.JWTAuth(authSvc))
	{
		agencies.POST("", middleware.RequireRole(models.RoleAdmin), agencyHandler.Create)
		// GET /agencies/:id — agency or admin
		agencies.GET("/:id",
			middleware.RequireRole(models.RoleAgency, models.RoleAdmin),
			agencyHandler.Get,
		)
		// PUT /agencies/:id/settings — agency or admin
		agencies.PUT("/:id/settings",
			middleware.RequireRole(models.RoleAgency, models.RoleAdmin),
			agencyHandler.UpdateSettings,
		)
		// GET /agencies/:id/streamers — agency or admin
		agencies.GET("/:id/streamers",
			middleware.RequireRole(models.RoleAgency, models.RoleAdmin),
			agencyHandler.ListStreamers,
		)
		// POST /agencies/:id/resend-setup — admin only
		agencies.POST("/:id/resend-setup",
			middleware.RequireRole(models.RoleAdmin),
			agencyHandler.ResendSetup,
		)
	}

	// ── Events ────────────────────────────────────────────────────────────
	events := v1.Group("/events")
	events.Use(middleware.JWTAuth(authSvc))
	events.Use(middleware.RequireRole(models.RoleStreamer, models.RoleAgency, models.RoleAdmin))
	{
		events.POST("/create", func(c *gin.Context) { c.JSON(501, gin.H{"error": "not implemented"}) })
		events.POST("/:id/settle", func(c *gin.Context) { c.JSON(501, gin.H{"error": "not implemented"}) })
	}

	// ── Admin (admin only) ──────────────────────────────────────────
	admin := v1.Group("/admin")
	admin.Use(middleware.JWTAuth(authSvc))
	admin.Use(middleware.RequireRole(models.RoleAdmin))
	{
		admin.GET("/users", func(c *gin.Context) { c.JSON(501, gin.H{"error": "not implemented"}) })
	}

	return r
}
