package router_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/tachigo/tachigo/internal/config"
	"github.com/tachigo/tachigo/internal/handlers"
	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/router"
	"github.com/tachigo/tachigo/internal/services"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type mockMailer struct{}

func (m *mockMailer) Send(to, subject, body string) error {
	return nil
}

type routerTestEnv struct {
	db      *gorm.DB
	authSvc *services.AuthService
	router  *gin.Engine
}

func newRouterTestEnv(t *testing.T) *routerTestEnv {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if err := migrateTestDB(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test-access-secret-at-least-32-chars!",
			RefreshSecret: "test-refresh-secret",
			AccessTTL:     15 * time.Minute,
			RefreshTTL:    30 * 24 * time.Hour,
		},
		App: config.AppConfig{
			FrontendURL: "http://localhost:3000",
		},
	}
	authSvc := services.NewAuthService(db, cfg)
	userSvc := services.NewUserService(db)
	addrSvc := services.NewAddressService(db)
	emailAuthSvc := services.NewEmailAuthService(db, cfg, &mockMailer{})
	extSvc := services.NewExtensionService(db, cfg, authSvc)
	watchSvc := services.NewWatchService(db)
	channelConfigSvc := services.NewChannelConfigService(db)
	pointsSvc := services.NewPointsService(db, watchSvc)
	airdropSvc := services.NewAirdropService(db, pointsSvc, channelConfigSvc)
	streamerSvc := services.NewStreamerService(db, pointsSvc)
	agencySvc := services.NewAgencyService(db)
	claimSvc := services.NewClaimService(db, config.ContractConfig{}, nil)
	spendSvc := services.NewSpendService(db, config.ContractConfig{}, nil)
	raffleSvc := services.NewRaffleService(db, "")
	agencyHandler := handlers.NewAgencyHandler(agencySvc, emailAuthSvc)

	engine := router.New(
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
		agencyHandler,
		[]string{"http://localhost:3000"},
	)

	return &routerTestEnv{
		db:      db,
		authSvc: authSvc,
		router:  engine,
	}
}

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
	}

	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}

	return nil
}

func (e *routerTestEnv) tokenForRole(t *testing.T, role models.UserRole, prefix string) (string, string) {
	t.Helper()

	email := prefix + "@example.com"
	username := prefix
	password := "password123"

	user, _, err := e.authSvc.Register(services.RegisterInput{
		Username: username,
		Email:    email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("register(%s): %v", role, err)
	}

	if err := e.db.Model(user).Update("role", role).Error; err != nil {
		t.Fatalf("set role(%s): %v", role, err)
	}

	_, tokens, err := e.authSvc.Login(services.LoginInput{
		Email:    email,
		Password: password,
	})
	if err != nil {
		t.Fatalf("login(%s): %v", role, err)
	}

	return email, tokens.AccessToken
}

func seedAgencyOwnedStreamerChannel(t *testing.T, env *routerTestEnv, agencyEmail, channelID string) {
	t.Helper()

	var agency models.User
	if err := env.db.Where("email = ?", agencyEmail).First(&agency).Error; err != nil {
		t.Fatalf("load agency: %v", err)
	}

	streamer, _, err := env.authSvc.Register(services.RegisterInput{
		Username: "streamer_" + channelID,
		Email:    "streamer_" + channelID + "@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("register streamer: %v", err)
	}
	if err := env.db.Model(streamer).Update("role", models.RoleStreamer).Error; err != nil {
		t.Fatalf("set streamer role: %v", err)
	}

	if err := env.db.Create(&models.Streamer{
		UserID:       streamer.ID,
		AgencyUserID: &agency.ID,
		ChannelID:    channelID,
		DisplayName:  "Agency owned channel",
	}).Error; err != nil {
		t.Fatalf("seed streamer channel: %v", err)
	}
}

func TestDashboardRouter_AgencyOwnedChannelConfigAccessible(t *testing.T) {
	env := newRouterTestEnv(t)
	agencyEmail, token := env.tokenForRole(t, models.RoleAgency, "agency_router")
	seedAgencyOwnedStreamerChannel(t, env, agencyEmail, "agency_owned_channel")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/channels/agency_owned_channel/config", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDashboardRouter_AgencyNonOwnedChannelConfigForbidden(t *testing.T) {
	env := newRouterTestEnv(t)
	_, token := env.tokenForRole(t, models.RoleAgency, "agency_router")
	otherAgencyEmail, _ := env.tokenForRole(t, models.RoleAgency, "other_agency_router")
	seedAgencyOwnedStreamerChannel(t, env, otherAgencyEmail, "other_agency_owned_channel")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/channels/other_agency_owned_channel/config", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	env.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestInternalRouter_SkipsRouteWhenSharedSecretMissing(t *testing.T) {
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if err := migrateTestDB(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test-access-secret-at-least-32-chars!",
			RefreshSecret: "test-refresh-secret",
			AccessTTL:     15 * time.Minute,
			RefreshTTL:    30 * 24 * time.Hour,
		},
		App: config.AppConfig{
			FrontendURL: "http://localhost:3000",
		},
	}

	authSvc := services.NewAuthService(db, cfg)
	userSvc := services.NewUserService(db)
	addrSvc := services.NewAddressService(db)
	emailAuthSvc := services.NewEmailAuthService(db, cfg, &mockMailer{})
	extSvc := services.NewExtensionService(db, cfg, authSvc)
	watchSvc := services.NewWatchService(db)
	channelConfigSvc := services.NewChannelConfigService(db)
	pointsSvc := services.NewPointsService(db, watchSvc)
	airdropSvc := services.NewAirdropService(db, pointsSvc, channelConfigSvc)
	streamerSvc := services.NewStreamerService(db, pointsSvc)
	agencySvc := services.NewAgencyService(db)
	claimSvc := services.NewClaimService(db, config.ContractConfig{}, nil)
	spendSvc := services.NewSpendService(db, config.ContractConfig{}, nil)
	raffleSvc := services.NewRaffleService(db, "")
	agencyHandler := handlers.NewAgencyHandler(agencySvc, emailAuthSvc)

	engine := router.New(
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
		agencyHandler,
		[]string{"http://localhost:3000"},
		router.InternalRouterConfig{DB: db, Config: cfg},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/internal/tachiya/users/points/balance?email=viewer@example.com", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404 when internal secret missing, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestInternalRouter_WithSecretSet_MiddlewareRejectsAndRouteRegistered(t *testing.T) {
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", dbName)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if err := migrateTestDB(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	const secret = "test-internal-secret"
	cfg := &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test-access-secret-at-least-32-chars!",
			RefreshSecret: "test-refresh-secret",
			AccessTTL:     15 * time.Minute,
			RefreshTTL:    30 * 24 * time.Hour,
		},
		App:      config.AppConfig{FrontendURL: "http://localhost:3000"},
		Internal: config.InternalConfig{TachiyaSharedSecret: secret},
	}

	authSvc := services.NewAuthService(db, cfg)
	userSvc := services.NewUserService(db)
	addrSvc := services.NewAddressService(db)
	emailAuthSvc := services.NewEmailAuthService(db, cfg, &mockMailer{})
	extSvc := services.NewExtensionService(db, cfg, authSvc)
	watchSvc := services.NewWatchService(db)
	channelConfigSvc := services.NewChannelConfigService(db)
	pointsSvc := services.NewPointsService(db, watchSvc)
	airdropSvc := services.NewAirdropService(db, pointsSvc, channelConfigSvc)
	streamerSvc := services.NewStreamerService(db, pointsSvc)
	agencySvc := services.NewAgencyService(db)
	claimSvc := services.NewClaimService(db, config.ContractConfig{}, nil)
	spendSvc := services.NewSpendService(db, config.ContractConfig{}, nil)
	raffleSvc := services.NewRaffleService(db, "")
	agencyHandler := handlers.NewAgencyHandler(agencySvc, emailAuthSvc)

	engine := router.New(
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
		agencyHandler,
		[]string{"http://localhost:3000"},
		router.InternalRouterConfig{DB: db, Config: cfg},
	)

	const path = "/api/v1/internal/tachiya/users/points/balance?email=nobody@example.com"

	// Without the secret header: middleware should reject with 401.
	reqNoHeader := httptest.NewRequest(http.MethodGet, path, nil)
	recNoHeader := httptest.NewRecorder()
	engine.ServeHTTP(recNoHeader, reqNoHeader)
	if recNoHeader.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 without secret header, got %d: %s", recNoHeader.Code, recNoHeader.Body.String())
	}

	// With the correct secret header: route is registered; unknown user → 404 from handler.
	reqWithHeader := httptest.NewRequest(http.MethodGet, path, nil)
	reqWithHeader.Header.Set("X-Tachiya-Internal-Secret", secret)
	recWithHeader := httptest.NewRecorder()
	engine.ServeHTTP(recWithHeader, reqWithHeader)
	if recWithHeader.Code != http.StatusNotFound {
		t.Fatalf("want 404 (user not found) with valid secret header, got %d: %s", recWithHeader.Code, recWithHeader.Body.String())
	}
}
