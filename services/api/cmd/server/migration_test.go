package main

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMigration011AddsAgencyUserForeignKeyConstraint(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}

	path := filepath.Join(filepath.Dir(file), "..", "..", "migrations", "011_streamers_agency.sql")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	sql := string(body)
	if !strings.Contains(sql, "fk_streamers_agency_user_id") {
		t.Fatalf("migration must create named fk_streamers_agency_user_id constraint")
	}
	fkPattern := regexp.MustCompile(`FOREIGN KEY\s*\(\s*agency_user_id\s*\)\s*REFERENCES\s+users\s*\(\s*id\s*\)`)
	if !fkPattern.MatchString(sql) {
		t.Fatalf("migration must add foreign key on streamers.agency_user_id")
	}
}

func TestMigration020ReconcilesAtlasBaselineOwnership(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}

	path := filepath.Join(filepath.Dir(file), "..", "..", "migrations", "020_atlas_reconcile_current_schema.sql")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	sql := string(body)
	for _, want := range []string{
		"CREATE TABLE IF NOT EXISTS watch_time_stats",
		"CREATE TABLE IF NOT EXISTS broadcast_time_stats",
		"CREATE TABLE IF NOT EXISTS broadcast_time_logs",
		"CREATE TABLE IF NOT EXISTS raffles",
		"CREATE TABLE IF NOT EXISTS raffle_entries",
		"CREATE TABLE IF NOT EXISTS raffle_draws",
		"CREATE TABLE IF NOT EXISTS raffle_claims",
		"fk_streamers_agency_user_id",
		"legacy agency_streamers backfill conflict",
		"UPDATE streamers s",
		"FROM agency_streamers",
		"idx_claims_tx_hash_not_null",
		"idx_claims_id_user_id",
		"idx_points_ledgers_id_user_id",
		"idx_points_transactions_id_ledger_id",
		"invalid coupon_redemptions rows detected",
		"fk_claim_items_claim_user",
		"fk_claim_items_ledger_user",
		"fk_claim_items_tx_ledger",
		"chk_coupon_redemptions_amount_gt_0",
		"idx_coupon_redemptions_compensation",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("migration 020 missing %q", want)
		}
	}
}

func TestMigrationDirectoryAtlasChecksumIsCurrent(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}

	migrationsDir := filepath.Join(filepath.Dir(file), "..", "..", "migrations")
	dir, err := migrate.NewLocalDir(migrationsDir)
	if err != nil {
		t.Fatalf("open migration dir: %v", err)
	}
	sum, err := dir.Checksum()
	if err != nil {
		t.Fatalf("checksum migration dir: %v", err)
	}
	got, err := sum.MarshalText()
	if err != nil {
		t.Fatalf("marshal checksum: %v", err)
	}
	want, err := os.ReadFile(filepath.Join(migrationsDir, migrate.HashFileName))
	if err != nil {
		t.Fatalf("read atlas.sum: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("atlas.sum is stale; regenerate after migration edits")
	}
}

func TestBackfillStreamerAgencyUserID_SingleAgencyChannel(t *testing.T) {
	db := newAgencyMigrationTestDB(t)

	if err := db.Exec(`INSERT INTO users (id) VALUES ('agency-1'), ('streamer-1')`).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Exec(`INSERT INTO streamers (id, user_id, agency_user_id, channel_id) VALUES ('streamer-row-1', 'streamer-1', NULL, 'channel-1')`).Error; err != nil {
		t.Fatalf("seed streamers: %v", err)
	}
	if err := db.Exec(`INSERT INTO agency_streamers (id, agency_id, channel_id) VALUES ('link-1', 'agency-1', 'channel-1')`).Error; err != nil {
		t.Fatalf("seed agency_streamers: %v", err)
	}

	if err := failOnAgencyBackfillConflicts(db); err != nil {
		t.Fatalf("unexpected conflict: %v", err)
	}
	if err := backfillStreamerAgencyUserID(db); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	var agencyUserID string
	if err := db.Raw(`SELECT agency_user_id FROM streamers WHERE id = 'streamer-row-1'`).Scan(&agencyUserID).Error; err != nil {
		t.Fatalf("load streamer: %v", err)
	}
	if agencyUserID != "agency-1" {
		t.Fatalf("want agency_user_id=agency-1, got %q", agencyUserID)
	}
}

func TestFailOnAgencyBackfillConflicts_MultiAgencyChannel(t *testing.T) {
	db := newAgencyMigrationTestDB(t)

	if err := db.Exec(`INSERT INTO users (id) VALUES ('agency-1'), ('agency-2'), ('streamer-1')`).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Exec(`INSERT INTO streamers (id, user_id, agency_user_id, channel_id) VALUES ('streamer-row-1', 'streamer-1', NULL, 'channel-1')`).Error; err != nil {
		t.Fatalf("seed streamer needing backfill: %v", err)
	}
	if err := db.Exec(`INSERT INTO agency_streamers (id, agency_id, channel_id) VALUES ('link-1', 'agency-1', 'channel-1'), ('link-2', 'agency-2', 'channel-1')`).Error; err != nil {
		t.Fatalf("seed agency_streamers: %v", err)
	}

	err := failOnAgencyBackfillConflicts(db)
	if err == nil {
		t.Fatal("want conflict error, got nil")
	}
	if err.Error() != "agency backfill conflict: 1 channel(s) map to multiple agencies in agency_streamers; resolve before deploying" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFailOnAgencyBackfillConflicts_IgnoresLegacyChannelOutsideBackfillScope(t *testing.T) {
	db := newAgencyMigrationTestDB(t)

	if err := db.Exec(`INSERT INTO users (id) VALUES ('agency-1'), ('agency-2'), ('streamer-1')`).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Exec(`INSERT INTO streamers (id, user_id, agency_user_id, channel_id) VALUES ('streamer-row-1', 'streamer-1', NULL, 'channel-ok')`).Error; err != nil {
		t.Fatalf("seed streamer needing backfill: %v", err)
	}
	if err := db.Exec(`INSERT INTO agency_streamers (id, agency_id, channel_id) VALUES ('link-1', 'agency-1', 'legacy-conflict'), ('link-2', 'agency-2', 'legacy-conflict')`).Error; err != nil {
		t.Fatalf("seed unrelated legacy conflict: %v", err)
	}

	if err := failOnAgencyBackfillConflicts(db); err != nil {
		t.Fatalf("unexpected conflict for unrelated legacy channel: %v", err)
	}
}

func TestFailOnAgencyBackfillConflicts_IgnoresAlreadyBackfilledStreamer(t *testing.T) {
	db := newAgencyMigrationTestDB(t)

	if err := db.Exec(`INSERT INTO users (id) VALUES ('agency-1'), ('agency-2'), ('streamer-1')`).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Exec(`INSERT INTO streamers (id, user_id, agency_user_id, channel_id) VALUES ('streamer-row-1', 'streamer-1', 'agency-1', 'channel-already-filled')`).Error; err != nil {
		t.Fatalf("seed already-backfilled streamer: %v", err)
	}
	if err := db.Exec(`INSERT INTO agency_streamers (id, agency_id, channel_id) VALUES ('link-1', 'agency-1', 'channel-already-filled'), ('link-2', 'agency-2', 'channel-already-filled')`).Error; err != nil {
		t.Fatalf("seed legacy conflict for already-filled streamer: %v", err)
	}

	if err := failOnAgencyBackfillConflicts(db); err != nil {
		t.Fatalf("unexpected conflict for already-backfilled streamer: %v", err)
	}
}

func TestPostgresBackfillUsesDeterministicChannelSource(t *testing.T) {
	sql := postgresBackfillStreamerAgencyUserIDSQL

	if !strings.Contains(sql, "DISTINCT ON (channel_id)") {
		t.Fatalf("postgres backfill SQL must use DISTINCT ON (channel_id):\n%s", sql)
	}
	if !strings.Contains(sql, "ORDER BY channel_id") {
		t.Fatalf("postgres backfill SQL must order its DISTINCT ON source:\n%s", sql)
	}
}

func newAgencyMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	stmts := []string{
		`CREATE TABLE users (id TEXT PRIMARY KEY)`,
		`CREATE TABLE streamers (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			agency_user_id TEXT,
			channel_id TEXT NOT NULL
		)`,
		`CREATE TABLE agency_streamers (
			id TEXT PRIMARY KEY,
			agency_id TEXT NOT NULL,
			channel_id TEXT NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("migrate db: %v", err)
		}
	}
	return db
}
