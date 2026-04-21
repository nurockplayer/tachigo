package handlers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

func migrateTestDB(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE,
			email TEXT UNIQUE,
			password_hash TEXT,
			avatar_url TEXT,
			role TEXT NOT NULL DEFAULT 'viewer',
			is_active INTEGER NOT NULL DEFAULT 1,
			email_verified INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS auth_providers (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			provider TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			access_token TEXT,
			refresh_token TEXT,
			token_expires_at DATETIME,
			metadata TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_providers_provider_provider_id_active
			ON auth_providers (provider, provider_id)
			WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_providers_web3_user_active
			ON auth_providers (user_id, provider)
			WHERE provider = 'web3' AND deleted_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS shipping_addresses (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			recipient_name TEXT NOT NULL,
			phone TEXT,
			address_line1 TEXT NOT NULL,
			address_line2 TEXT,
			city TEXT NOT NULL,
			district TEXT,
			postal_code TEXT,
			country TEXT NOT NULL DEFAULT 'TW',
			is_default INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			token_hash TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			created_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS web3_nonces (
			id TEXT PRIMARY KEY,
			nonce TEXT NOT NULL UNIQUE,
			address TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS email_verifications (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			token_hash TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			created_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS password_resets (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at DATETIME NOT NULL,
			created_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS channel_configs (
			channel_id TEXT PRIMARY KEY,
			seconds_per_point INTEGER NOT NULL DEFAULT 60,
			multiplier INTEGER NOT NULL DEFAULT 1,
			daily_airdrop_limit INTEGER NOT NULL DEFAULT 5000,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS agency_streamers (
			id TEXT PRIMARY KEY,
			agency_id TEXT NOT NULL,
			channel_id TEXT NOT NULL,
			created_at DATETIME,
			UNIQUE (agency_id, channel_id)
		)`,
		`CREATE TABLE IF NOT EXISTS streamers (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			agency_user_id TEXT REFERENCES users(id),
			channel_id TEXT NOT NULL,
			display_name TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_streamers_user_channel
			ON streamers (user_id, channel_id)`,
		`CREATE INDEX IF NOT EXISTS idx_streamers_agency_user_id
			ON streamers (agency_user_id)`,
		`CREATE TABLE IF NOT EXISTS watch_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			channel_id TEXT NOT NULL,
			accumulated_seconds INTEGER NOT NULL DEFAULT 0,
			rewarded_seconds INTEGER NOT NULL DEFAULT 0,
			last_heartbeat_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			click_cooldown_until DATETIME NOT NULL DEFAULT '1970-01-01 00:00:00',
			is_active INTEGER NOT NULL DEFAULT 1,
			ended_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_watch_sessions_active_user_channel
			ON watch_sessions (user_id, channel_id)
			WHERE is_active = 1`,
		`CREATE TABLE IF NOT EXISTS points_ledgers (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			channel_id TEXT NOT NULL,
			cumulative_total INTEGER NOT NULL DEFAULT 0,
			spendable_balance INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_points_ledgers_user_channel
			ON points_ledgers (user_id, channel_id)`,
		`CREATE TABLE IF NOT EXISTS points_transactions (
			id TEXT PRIMARY KEY,
			ledger_id TEXT NOT NULL REFERENCES points_ledgers(id),
			watch_session_id TEXT,
			source TEXT NOT NULL,
			delta INTEGER NOT NULL,
			balance_after INTEGER NOT NULL,
			sku TEXT,
			note TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS claims (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			wallet_addr TEXT NOT NULL,
			amount INTEGER NOT NULL CHECK (amount > 0),
			status TEXT NOT NULL CHECK (status IN ('pending', 'broadcast', 'confirmed', 'failed')),
			tx_hash TEXT,
			error_message TEXT,
			broadcast_at DATETIME,
			confirmed_at DATETIME,
			failed_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_claims_user_created_at
			ON claims (user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_claims_status_created_at
			ON claims (status, created_at DESC)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_claims_tx_hash_not_null
			ON claims (tx_hash)
			WHERE tx_hash IS NOT NULL`,
		`CREATE TABLE IF NOT EXISTS claim_items (
			id TEXT PRIMARY KEY,
			claim_id TEXT NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
			ledger_id TEXT NOT NULL REFERENCES points_ledgers(id),
			points_transaction_id TEXT NOT NULL REFERENCES points_transactions(id),
			amount INTEGER NOT NULL CHECK (amount > 0),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (points_transaction_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_claim_items_claim_id
			ON claim_items (claim_id)`,
		`CREATE INDEX IF NOT EXISTS idx_claim_items_ledger_id
			ON claim_items (ledger_id)`,
		`CREATE TABLE IF NOT EXISTS watch_time_stats (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			channel_id TEXT NOT NULL,
			total_watch_seconds INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_watch_time_user_channel
			ON watch_time_stats (user_id, channel_id)`,
		`CREATE TABLE IF NOT EXISTS broadcast_time_stats (
			id TEXT PRIMARY KEY,
			streamer_id TEXT NOT NULL,
			channel_id TEXT NOT NULL,
			total_broadcast_seconds INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_broadcast_time_streamer_channel
			ON broadcast_time_stats (streamer_id, channel_id)`,
		`CREATE TABLE IF NOT EXISTS broadcast_time_logs (
			id TEXT PRIMARY KEY,
			streamer_id TEXT NOT NULL,
			channel_id TEXT NOT NULL,
			seconds INTEGER NOT NULL,
			recorded_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tachi_balances (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			balance INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (user_id),
			CHECK (balance >= 0)
		)`,
		`CREATE TABLE IF NOT EXISTS raffles (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			title TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'draft',
			source TEXT NOT NULL DEFAULT 'csv',
			scheduled_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_raffles_user_id ON raffles (user_id)`,
		`CREATE TABLE IF NOT EXISTS raffle_entries (
			id TEXT PRIMARY KEY,
			raffle_id TEXT NOT NULL REFERENCES raffles(id),
			user_id TEXT REFERENCES users(id),
			twitch_login TEXT NOT NULL,
			display_name TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (raffle_id, twitch_login),
			UNIQUE (id, raffle_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_raffle_entries_raffle_id ON raffle_entries (raffle_id)`,
		`CREATE TABLE IF NOT EXISTS raffle_draws (
			id TEXT PRIMARY KEY,
			raffle_id TEXT NOT NULL REFERENCES raffles(id),
			entry_id TEXT NOT NULL,
			claim_token TEXT NOT NULL UNIQUE,
			claim_expires_at DATETIME NOT NULL,
			drawn_at DATETIME NOT NULL,
			UNIQUE (raffle_id, entry_id),
			FOREIGN KEY (entry_id, raffle_id) REFERENCES raffle_entries(id, raffle_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_raffle_draws_raffle_id ON raffle_draws (raffle_id)`,
		`CREATE TABLE IF NOT EXISTS raffle_claims (
			id TEXT PRIMARY KEY,
			draw_id TEXT NOT NULL UNIQUE REFERENCES raffle_draws(id),
			recipient_name TEXT NOT NULL,
			phone TEXT NOT NULL DEFAULT '',
			address_line1 TEXT NOT NULL,
			address_line2 TEXT NOT NULL DEFAULT '',
			city TEXT NOT NULL,
			postal_code TEXT NOT NULL DEFAULT '',
			country TEXT NOT NULL DEFAULT 'TW',
			submitted_at DATETIME NOT NULL
		)`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			return err
		}
	}
	return nil
}

func init() {
	gin.SetMode(gin.TestMode)
}

// mockMailer captures sent emails for handler tests.
type mockMailer struct {
	sent []struct{ to, subject, body string }
}

func (m *mockMailer) Send(to, subject, body string) error {
	m.sent = append(m.sent, struct{ to, subject, body string }{to, subject, body})
	return nil
}

type testEnv struct {
	db           *gorm.DB
	authSvc      *services.AuthService
	userSvc      *services.UserService
	emailAuthSvc *services.EmailAuthService
	router       *gin.Engine
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	return newTestEnvWithConfig(t, "development", "")
}

func newTestEnvWithServerEnv(t *testing.T, serverEnv string) *testEnv {
	t.Helper()
	return newTestEnvWithConfig(t, serverEnv, "")
}

func newTestEnvWithConfig(t *testing.T, serverEnv, frontendURL string) *testEnv {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if err := migrateTestDB(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Env: serverEnv,
		},
		App: config.AppConfig{
			FrontendURL: frontendURL,
		},
		JWT: config.JWTConfig{
			AccessSecret:  "test-access-secret-at-least-32-chars!",
			RefreshSecret: "test-refresh-secret",
			AccessTTL:     15 * time.Minute,
			RefreshTTL:    30 * 24 * time.Hour,
		},
	}

	authSvc := services.NewAuthService(db, cfg)
	userSvc := services.NewUserService(db)
	addrSvc := services.NewAddressService(db)
	emailAuthSvc := services.NewEmailAuthService(db, cfg, &mockMailer{})

	authH := handlers.NewAuthHandler(authSvc, cfg).WithEmailAuth(emailAuthSvc)
	userH := handlers.NewUserHandler(userSvc)
	addrH := handlers.NewAddressHandler(addrSvc)
	emailH := handlers.NewEmailAuthHandler(emailAuthSvc)

	r := gin.New()
	r.Use(gin.Recovery())

	v1 := r.Group("/api/v1")

	auth := v1.Group("/auth")
	auth.POST("/register", authH.Register)
	auth.POST("/login", authH.Login)
	auth.POST("/refresh", authH.Refresh)
	auth.POST("/logout", authH.Logout)
	auth.POST("/web3/nonce", authH.Web3Nonce)
	auth.POST("/web3/verify", authH.Web3Verify)
	auth.POST("/verify-email/confirm", emailH.ConfirmVerification)
	auth.POST("/forgot-password", emailH.ForgotPassword)
	auth.POST("/reset-password", emailH.ResetPassword)

	protected := v1.Group("/")
	protected.Use(middleware.JWTAuth(authSvc))
	protected.GET("users/me", userH.Me)
	protected.PUT("users/me", userH.UpdateMe)
	protected.GET("users/me/providers", userH.ListProviders)
	protected.POST("users/me/wallet", userH.LinkWallet)
	protected.DELETE("auth/providers/:provider", authH.UnlinkProvider)
	protected.POST("auth/verify-email/send", emailH.SendVerification)

	addrs := protected.Group("users/me/addresses")
	addrs.GET("", addrH.List)
	addrs.POST("", addrH.Create)
	addrs.PUT("/:id", addrH.Update)
	addrs.DELETE("/:id", addrH.Delete)
	addrs.PUT("/:id/default", addrH.SetDefault)

	return &testEnv{db: db, authSvc: authSvc, userSvc: userSvc, emailAuthSvc: emailAuthSvc, router: r}
}

// registerUser is a helper that registers a user and returns access + refresh tokens.
func (e *testEnv) registerUser(t *testing.T, username, email, password string) (accessToken, refreshToken string) {
	t.Helper()
	_, tokens, err := e.authSvc.Register(services.RegisterInput{
		Username: username,
		Email:    email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("registerUser: %v", err)
	}
	return tokens.AccessToken, tokens.RefreshToken
}

// parseBody is a helper to decode JSON response bodies in tests.
func parseBody(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("parseBody: %v", err)
	}
	return out
}
