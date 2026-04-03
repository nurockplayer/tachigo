package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

func New(
	authSvc *services.AuthService,
	userSvc *services.UserService,
	addrSvc *services.AddressService,
	extSvc *services.ExtensionService,
	emailAuthSvc *services.EmailAuthService,
	watchSvc *services.WatchService,
	channelConfigSvc *services.ChannelConfigService,
	pointsSvc *services.PointsService,
	allowedOrigins []string,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.CORS(allowedOrigins))

	authH := handlers.NewAuthHandler(authSvc).WithEmailAuth(emailAuthSvc)
	userH := handlers.NewUserHandler(userSvc)
	addrH := handlers.NewAddressHandler(addrSvc)
	extH := handlers.NewExtensionHandler(extSvc)
	emailH := handlers.NewEmailAuthHandler(emailAuthSvc)
	watchH := handlers.NewWatchHandler(watchSvc, pointsSvc)
	channelConfigH := handlers.NewChannelConfigHandler(channelConfigSvc)
	pointsH := handlers.NewPointsHandler(pointsSvc)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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

	// ── Twitch Extension endpoints ────────────────────────────────────────
	ext := v1.Group("/extension")
	{
		ext.POST("/auth/login", extH.Login)
		ext.POST("/bits/complete", extH.BitsComplete)

		// Watch-time points (requires tachigo JWT — viewer must log in first)
		watch := ext.Group("/watch")
		watch.Use(middleware.JWTAuth(authSvc))
		{
			watch.POST("/start", watchH.StartSession)
			watch.POST("/heartbeat", watchH.Heartbeat)
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

		// User profile
		protected.GET("users/me", userH.Me)
		protected.PUT("users/me", userH.UpdateMe)
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
	dashboard.Use(middleware.RequireRole(models.RoleAdmin, models.RoleStreamer))
	{
		dashboard.PUT("/channels/:channel_id/config", channelConfigH.UpdateChannelConfig)
	}

	// ── Agency management ─────────────────────────────────────────────────
	// POST /agencies — admin only
	agencies := v1.Group("/agencies")
	agencies.Use(middleware.JWTAuth(authSvc))
	{
		agencies.POST("", middleware.RequireRole(models.RoleAdmin), func(c *gin.Context) {
			c.JSON(501, gin.H{"error": "not implemented"})
		})
		// PUT /agencies/:id/settings — agency or admin
		agencies.PUT("/:id/settings",
			middleware.RequireRole(models.RoleAgency, models.RoleAdmin),
			func(c *gin.Context) { c.JSON(501, gin.H{"error": "not implemented"}) },
		)
		// GET /agencies/:id/streamers — agency or admin
		agencies.GET("/:id/streamers",
			middleware.RequireRole(models.RoleAgency, models.RoleAdmin),
			func(c *gin.Context) { c.JSON(501, gin.H{"error": "not implemented"}) },
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
