package services

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tachigo/tachigo/internal/models"
)

// seedLedger inserts a points_ledger row and returns its id.
func seedLedger(t *testing.T, db *gorm.DB, userID uuid.UUID, channelID string, spendable int64) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := db.Exec(`
		INSERT INTO points_ledgers (id, user_id, channel_id, spendable_balance, cumulative_total, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, id, userID, channelID, spendable, spendable).Error; err != nil {
		t.Fatalf("seedLedger: %v", err)
	}
	return id
}

func userIDForClaim(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	return seedStreamerUserRow(t, db, models.RoleViewer)
}

func TestGetTachiBalance_Zero(t *testing.T) {
	db := newTestDB(t)
	svc := NewClaimService(db)
	userID := userIDForClaim(t, db)

	bal, err := svc.GetTachiBalance(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bal != 0 {
		t.Fatalf("expected 0, got %d", bal)
	}
}

func TestClaim_All(t *testing.T) {
	db := newTestDB(t)
	svc := NewClaimService(db)
	userID := userIDForClaim(t, db)
	seedLedger(t, db, userID, "ch1", 100)
	seedLedger(t, db, userID, "ch2", 50)

	newBal, err := svc.Claim(userID, 0) // claim all
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newBal != 150 {
		t.Fatalf("expected tachi_balance=150, got %d", newBal)
	}

	// spendable_balance should be 0 in all ledgers
	var total int64
	db.Raw("SELECT COALESCE(SUM(spendable_balance),0) FROM points_ledgers WHERE user_id = ?", userID).Scan(&total)
	if total != 0 {
		t.Fatalf("expected spendable_balance=0, got %d", total)
	}
}

func TestClaim_PartialAmount(t *testing.T) {
	db := newTestDB(t)
	svc := NewClaimService(db)
	userID := userIDForClaim(t, db)
	seedLedger(t, db, userID, "ch1", 100)

	newBal, err := svc.Claim(userID, 40)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newBal != 40 {
		t.Fatalf("expected tachi_balance=40, got %d", newBal)
	}

	var remaining int64
	db.Raw("SELECT spendable_balance FROM points_ledgers WHERE user_id = ? AND channel_id = 'ch1'", userID).Scan(&remaining)
	if remaining != 60 {
		t.Fatalf("expected remaining spendable=60, got %d", remaining)
	}
}

func TestClaim_InsufficientBalance(t *testing.T) {
	db := newTestDB(t)
	svc := NewClaimService(db)
	userID := userIDForClaim(t, db)
	seedLedger(t, db, userID, "ch1", 30)

	_, err := svc.Claim(userID, 100)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if err != ErrClaimInsufficientBalance {
		t.Fatalf("expected ErrClaimInsufficientBalance, got %v", err)
	}
}

func TestClaim_NoLedgers(t *testing.T) {
	db := newTestDB(t)
	svc := NewClaimService(db)
	userID := userIDForClaim(t, db)

	// amount=0 with no ledgers → claimAmount=0 → ErrClaimAmountInvalid
	_, err := svc.Claim(userID, 0)
	if !errors.Is(err, ErrClaimAmountInvalid) {
		t.Fatalf("expected ErrClaimAmountInvalid, got %v", err)
	}
}

func TestClaim_AccumulatesOnSecondClaim(t *testing.T) {
	db := newTestDB(t)
	svc := NewClaimService(db)
	userID := userIDForClaim(t, db)
	seedLedger(t, db, userID, "ch1", 200)

	bal1, err1 := svc.Claim(userID, 100)
	if err1 != nil {
		t.Fatalf("first claim unexpected error: %v", err1)
	}
	if bal1 != 100 {
		t.Fatalf("expected first tachi_balance=100, got %d", bal1)
	}
	newBal, err := svc.Claim(userID, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newBal != 150 {
		t.Fatalf("expected tachi_balance=150, got %d", newBal)
	}
}
