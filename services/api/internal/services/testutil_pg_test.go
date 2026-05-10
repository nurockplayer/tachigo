//go:build integration

package services

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// pgContainerURL holds the connection string for the shared PostgreSQL container.
// Set by TestMain before any test runs.
var pgContainerURL string

func TestMain(m *testing.M) {
	// If DATABASE_URL is set, use it directly (e.g. CI with external postgres).
	if u := os.Getenv("DATABASE_URL"); u != "" {
		pgContainerURL = u
		os.Exit(m.Run())
	}

	ctx := context.Background()
	pgc, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("tachigo"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		if pgc != nil {
			_ = pgc.Terminate(ctx)
		}
		log.Fatalf("start postgres container: %v", err)
	}

	connStr, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		if terr := pgc.Terminate(ctx); terr != nil {
			log.Printf("terminate postgres container: %v", terr)
		}
		log.Fatalf("get connection string: %v", err)
	}
	pgContainerURL = connStr

	code := m.Run()

	if err := pgc.Terminate(ctx); err != nil {
		log.Printf("terminate postgres container: %v", err)
	}
	os.Exit(code)
}

func newPGTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	if pgContainerURL == "" {
		t.Skip("no postgres available")
	}

	adminDB, err := gorm.Open(postgres.Open(pgContainerURL), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Fatalf("open postgres admin db: %v", err)
	}

	schemaName := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := adminDB.Exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"`, schemaName)).Error; err != nil {
		t.Fatalf("create schema %s: %v", schemaName, err)
	}

	testDB, err := gorm.Open(postgres.Open(withSearchPath(pgContainerURL, schemaName)), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Fatalf("open postgres test db: %v", err)
	}

	if err := migratePGTestDB(testDB); err != nil {
		t.Fatalf("migrate postgres test db: %v", err)
	}

	t.Cleanup(func() {
		if err := adminDB.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS "%s" CASCADE`, schemaName)).Error; err != nil {
			t.Fatalf("drop schema %s: %v", schemaName, err)
		}
	})

	return testDB
}

func withSearchPath(databaseURL, schemaName string) string {
	if strings.HasPrefix(databaseURL, "postgres://") || strings.HasPrefix(databaseURL, "postgresql://") {
		parsed, err := url.Parse(databaseURL)
		if err != nil {
			return databaseURL
		}
		query := parsed.Query()
		query.Set("search_path", schemaName)
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}
	return strings.TrimSpace(databaseURL) + " search_path=" + schemaName
}

func migratePGTestDB(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			username TEXT UNIQUE,
			email TEXT UNIQUE,
			password_hash TEXT,
			avatar_url TEXT,
			role TEXT NOT NULL DEFAULT 'viewer' CHECK (role IN ('viewer', 'streamer', 'agency', 'admin')),
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			email_verified BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ,
			deleted_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS auth_providers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			provider VARCHAR(20) NOT NULL,
			provider_id VARCHAR(255) NOT NULL,
			access_token TEXT,
			refresh_token TEXT,
			token_expires_at TIMESTAMPTZ,
			metadata JSONB,
			created_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ,
			deleted_at TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_auth_providers_user_id
			ON auth_providers (user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_auth_providers_deleted_at
			ON auth_providers (deleted_at)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_providers_provider_provider_id_active
			ON auth_providers (provider, provider_id)
			WHERE deleted_at IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_providers_web3_user_active
			ON auth_providers (user_id, provider)
			WHERE provider = 'web3' AND deleted_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS watch_sessions (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			channel_id TEXT NOT NULL,
			accumulated_seconds BIGINT NOT NULL DEFAULT 0,
			rewarded_seconds BIGINT NOT NULL DEFAULT 0,
			last_heartbeat_at TIMESTAMPTZ NOT NULL,
			click_cooldown_until TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00',
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			ended_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_watch_sessions_active_user_channel
			ON watch_sessions (user_id, channel_id)
			WHERE is_active = TRUE`,
		`CREATE TABLE IF NOT EXISTS channel_configs (
			channel_id TEXT PRIMARY KEY,
			seconds_per_point BIGINT NOT NULL DEFAULT 60,
			multiplier BIGINT NOT NULL DEFAULT 1,
			daily_airdrop_limit BIGINT NOT NULL DEFAULT 5000,
			created_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS points_ledgers (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			channel_id TEXT NOT NULL,
			cumulative_total BIGINT NOT NULL DEFAULT 0,
			spendable_balance BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_points_ledgers_user_channel
			ON points_ledgers (user_id, channel_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_points_ledgers_id_user_id
			ON points_ledgers (id, user_id)`,
		`CREATE TABLE IF NOT EXISTS points_transactions (
			id UUID PRIMARY KEY,
			ledger_id UUID NOT NULL,
			watch_session_id UUID,
			source VARCHAR(50) NOT NULL,
			delta BIGINT NOT NULL,
			balance_after BIGINT NOT NULL,
			sku TEXT,
			note TEXT,
			external_transaction_id VARCHAR(255) UNIQUE,
			created_at TIMESTAMPTZ
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_points_transactions_id_ledger_id
			ON points_transactions (id, ledger_id)`,
		`CREATE TABLE IF NOT EXISTS claims (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			wallet_addr VARCHAR(42) NOT NULL,
			amount BIGINT NOT NULL CHECK (amount > 0),
			status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'broadcast', 'confirmed', 'failed', 'finalize_failed')),
			tx_hash VARCHAR(66),
			error_message TEXT,
			broadcast_at TIMESTAMPTZ,
			confirmed_at TIMESTAMPTZ,
			failed_at TIMESTAMPTZ,
			finalize_failed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_claims_id_user_id
			ON claims (id, user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_claims_user_created_at
			ON claims (user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_claims_status_created_at
			ON claims (status, created_at DESC)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_claims_tx_hash_not_null
			ON claims (tx_hash)
			WHERE tx_hash IS NOT NULL`,
		`CREATE TABLE IF NOT EXISTS claim_items (
			id UUID PRIMARY KEY,
			claim_id UUID NOT NULL,
			claim_user_id UUID NOT NULL,
			ledger_id UUID NOT NULL,
			points_transaction_id UUID NOT NULL,
			amount BIGINT NOT NULL CHECK (amount > 0),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			FOREIGN KEY (claim_id, claim_user_id) REFERENCES claims(id, user_id) ON DELETE CASCADE,
			FOREIGN KEY (ledger_id, claim_user_id) REFERENCES points_ledgers(id, user_id),
			FOREIGN KEY (points_transaction_id, ledger_id) REFERENCES points_transactions(id, ledger_id),
			UNIQUE (points_transaction_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_claim_items_claim_id
			ON claim_items (claim_id)`,
		`CREATE INDEX IF NOT EXISTS idx_claim_items_ledger_id
			ON claim_items (ledger_id)`,
		`CREATE TABLE IF NOT EXISTS tachi_balances (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			balance NUMERIC(20,6) NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (user_id),
			CHECK (balance >= 0)
		)`,
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ
		)`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id
			ON refresh_tokens (user_id)`,
	}

	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}

	return nil
}
