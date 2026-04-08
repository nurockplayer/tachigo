package main

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMigration008AddsAgencyUserForeignKeyConstraint(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file")
	}

	path := filepath.Join(filepath.Dir(file), "..", "..", "migrations", "008_streamers_agency.sql")
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

	if err := db.Exec(`INSERT INTO users (id) VALUES ('agency-1'), ('agency-2')`).Error; err != nil {
		t.Fatalf("seed users: %v", err)
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
