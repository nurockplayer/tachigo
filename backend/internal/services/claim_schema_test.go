package services

import "testing"

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
}
