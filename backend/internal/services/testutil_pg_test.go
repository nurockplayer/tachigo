//go:build integration

package services

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newPGTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set DATABASE_URL to run integration tests")
	}

	adminDB, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
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

	testDB, err := gorm.Open(postgres.Open(withSearchPath(databaseURL, schemaName)), &gorm.Config{
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
		`CREATE TABLE IF NOT EXISTS points_transactions (
			id UUID PRIMARY KEY,
			ledger_id UUID NOT NULL,
			watch_session_id UUID,
			source VARCHAR(50) NOT NULL,
			delta BIGINT NOT NULL,
			balance_after BIGINT NOT NULL,
			sku TEXT,
			note TEXT,
			created_at TIMESTAMPTZ
		)`,
	}

	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}

	return nil
}
