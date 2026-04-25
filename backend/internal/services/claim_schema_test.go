package services

import (
	"errors"
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
	assertContains(t, claimsDDL, "CHECK (status IN ('pending', 'broadcast', 'confirmed', 'failed', 'finalize_failed'))")
	assertContains(t, claimsDDL, "finalize_failed_at DATETIME")
	assertContains(t, claimsDDL, "created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP")
	assertContains(t, claimsDDL, "updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP")

	var claimItemsDDL string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'claim_items'").Scan(&claimItemsDDL).Error; err != nil {
		t.Fatalf("query claim_items ddl: %v", err)
	}
	assertContains(t, claimItemsDDL, "claim_user_id")
	assertContains(t, claimItemsDDL, "UNIQUE (points_transaction_id)")

	assertIndexExists(t, db, "idx_claims_tx_hash_not_null")
}

func TestMigrateTestDB_RejectsClaimItemWhenTransactionLedgerMismatch(t *testing.T) {
	db := newTestDB(t)

	mustExec(t, db, `
		INSERT INTO users (id) VALUES
			('u1')
	`)
	mustExec(t, db, `
		INSERT INTO points_ledgers (id, user_id, channel_id, spendable_balance, cumulative_total, created_at, updated_at) VALUES
			('l1', 'u1', 'c1', 100, 100, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			('l2', 'u1', 'c2', 100, 100, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)
	mustExec(t, db, `
		INSERT INTO points_transactions (id, ledger_id, source, delta, balance_after, created_at) VALUES
			('tx2', 'l2', 'claim', -10, 90, CURRENT_TIMESTAMP)
	`)
	mustExec(t, db, `
		INSERT INTO claims (id, user_id, wallet_addr, amount, status, created_at, updated_at) VALUES
			('c1', 'u1', '0x1111111111111111111111111111111111111111', 10, 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)

	err := db.Exec(`
		INSERT INTO claim_items (id, claim_id, claim_user_id, ledger_id, points_transaction_id, amount, created_at)
		VALUES ('ci1', 'c1', 'u1', 'l1', 'tx2', 10, CURRENT_TIMESTAMP)
	`).Error
	assertForeignKeyRejected(t, err)
}

func TestMigrateTestDB_AllowsClaimItemWhenCompositeKeysMatch(t *testing.T) {
	db := newTestDB(t)

	mustExec(t, db, `
		INSERT INTO users (id) VALUES
			('u1')
	`)
	mustExec(t, db, `
		INSERT INTO points_ledgers (id, user_id, channel_id, spendable_balance, cumulative_total, created_at, updated_at) VALUES
			('l1', 'u1', 'c1', 100, 100, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)
	mustExec(t, db, `
		INSERT INTO points_transactions (id, ledger_id, source, delta, balance_after, created_at) VALUES
			('tx1', 'l1', 'claim', -10, 90, CURRENT_TIMESTAMP)
	`)
	mustExec(t, db, `
		INSERT INTO claims (id, user_id, wallet_addr, amount, status, created_at, updated_at) VALUES
			('c1', 'u1', '0x1111111111111111111111111111111111111111', 10, 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)

	if err := db.Exec(`
		INSERT INTO claim_items (id, claim_id, claim_user_id, ledger_id, points_transaction_id, amount, created_at)
		VALUES ('ci-ok', 'c1', 'u1', 'l1', 'tx1', 10, CURRENT_TIMESTAMP)
	`).Error; err != nil {
		t.Fatalf("expected valid claim_item insert to succeed, got: %v", err)
	}
}

func TestMigrateTestDB_RejectsClaimItemWhenClaimUserLedgerUserMismatch(t *testing.T) {
	db := newTestDB(t)

	mustExec(t, db, `
		INSERT INTO users (id) VALUES
			('u1'),
			('u2')
	`)
	mustExec(t, db, `
		INSERT INTO points_ledgers (id, user_id, channel_id, spendable_balance, cumulative_total, created_at, updated_at) VALUES
			('l2', 'u2', 'c2', 100, 100, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)
	mustExec(t, db, `
		INSERT INTO points_transactions (id, ledger_id, source, delta, balance_after, created_at) VALUES
			('tx2', 'l2', 'claim', -10, 90, CURRENT_TIMESTAMP)
	`)
	mustExec(t, db, `
		INSERT INTO claims (id, user_id, wallet_addr, amount, status, created_at, updated_at) VALUES
			('c1', 'u1', '0x1111111111111111111111111111111111111111', 10, 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)

	err := db.Exec(`
		INSERT INTO claim_items (id, claim_id, claim_user_id, ledger_id, points_transaction_id, amount, created_at)
		VALUES ('ci2', 'c1', 'u1', 'l2', 'tx2', 10, CURRENT_TIMESTAMP)
	`).Error
	assertForeignKeyRejected(t, err)
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

func mustExec(t *testing.T, db *gorm.DB, sql string) {
	t.Helper()
	if err := db.Exec(sql).Error; err != nil {
		t.Fatalf("exec sql failed: %v\nSQL=%s", err, sql)
	}
}

func assertForeignKeyRejected(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected foreign key rejection, got nil error")
	}
	if errors.Is(err, gorm.ErrForeignKeyViolated) {
		return
	}
	if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
		return
	}
	t.Fatalf("expected foreign key rejection, got: %v", err)
}
