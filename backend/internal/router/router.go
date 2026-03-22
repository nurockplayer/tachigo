package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

func New(
	authSvc *services.AuthService,
	userSvc *services.UserService,
	addrSvc *services.AddressService,
	extSvc *services.ExtensionService,
	emailAuthSvc *services.EmailAuthService,
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
	}

	// ── Authenticated routes ──────────────────────────────────────────────
	protected := v1.Group("/")
	protected.Use(middleware.JWTAuth(authSvc))
	{
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

	return r
}
