package services

import (
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestMigrateTestDB_CreatesClaimTables(t *testing.T) {
	db := newTestDB(t)

	var claimsCount int64
	if err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'claims'").Scan(&claimsCount).Error; err != nil {
		t.Fatalf("query claims table: %v", err)
	}
	if claimsCount != 1 {
		t.Fatalf("expected claims table to exist, got count=%d", claimsCount)
	}

	var claimItemsCount int64
	if err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'claim_items'").Scan(&claimItemsCount).Error; err != nil {
		t.Fatalf("query claim_items table: %v", err)
	}
	if claimItemsCount != 1 {
		t.Fatalf("expected claim_items table to exist, got count=%d", claimItemsCount)
	}

	var claimsDDL string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'claims'").Scan(&claimsDDL).Error; err != nil {
		t.Fatalf("query claims ddl: %v", err)
	}
	assertContains(t, claimsDDL, "CHECK (amount > 0)")
	assertContains(t, claimsDDL, "CHECK (status IN ('pending', 'broadcast', 'confirmed', 'failed'))")
	assertContains(t, claimsDDL, "created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP")
	assertContains(t, claimsDDL, "updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP")

	var claimItemsDDL string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'claim_items'").Scan(&claimItemsDDL).Error; err != nil {
		t.Fatalf("query claim_items ddl: %v", err)
	}
	assertContains(t, claimItemsDDL, "UNIQUE (points_transaction_id)")

	assertIndexExists(t, db, "idx_claims_tx_hash_not_null")
}

func assertIndexExists(t *testing.T, db *gorm.DB, name string) {
	t.Helper()

	var n int64
	if err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?", name).Scan(&n).Error; err != nil {
		t.Fatalf("query index %s: %v", name, err)
	}
	if n != 1 {
		t.Fatalf("expected index %s to exist, got count=%d", name, n)
	}
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected DDL to contain %q\nDDL=%s", want, got)
	}
}
