package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── mock BurnCaller ──────────────────────────────────────────────────────────

type mockBurnCaller struct {
	txHash string
	err    error
	calls  []burnCall
}

type burnCall struct {
	fromAddr string
	amount   int64
}

func (m *mockBurnCaller) BurnOnChain(_ context.Context, fromAddr string, amount int64) (string, error) {
	m.calls = append(m.calls, burnCall{fromAddr: fromAddr, amount: amount})
	if m.err != nil {
		return "", m.err
	}
	return m.txHash, nil
}

// ── seed helpers ─────────────────────────────────────────────────────────────

func seedTachiBalance(t *testing.T, db *gorm.DB, userID uuid.UUID, balance int64) {
	t.Helper()
	if err := db.Exec(`
		INSERT INTO tachi_balances (id, user_id, balance, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`, uuid.New().String(), userID.String(), balance).Error; err != nil {
		t.Fatalf("seedTachiBalance: %v", err)
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestRedeem_Success(t *testing.T) {
	db := newTestDB(t)
	burnCaller := &mockBurnCaller{txHash: "0xburn123"}
	svc := &SpendService{db: db, burnCaller: burnCaller}

	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedTachiBalance(t, db, userID, 500)

	newBal, err := svc.Redeem(context.Background(), userID, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newBal != 400 {
		t.Fatalf("expected newBalance=400, got %d", newBal)
	}
	if len(burnCaller.calls) != 1 {
		t.Fatalf("expected 1 burn call, got %d", len(burnCaller.calls))
	}
	if burnCaller.calls[0].amount != 100 {
		t.Fatalf("expected burn amount=100, got %d", burnCaller.calls[0].amount)
	}
	if burnCaller.calls[0].fromAddr != "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045" {
		t.Fatalf("unexpected burn fromAddr: %s", burnCaller.calls[0].fromAddr)
	}

	var dbBal int64
	db.Raw("SELECT balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&dbBal)
	if dbBal != 400 {
		t.Fatalf("expected db balance=400, got %d", dbBal)
	}
}

func TestRedeem_InsufficientBalance(t *testing.T) {
	db := newTestDB(t)
	svc := &SpendService{db: db}

	userID := userIDForClaim(t, db)
	seedTachiBalance(t, db, userID, 50)

	_, err := svc.Redeem(context.Background(), userID, 100)
	if !errors.Is(err, ErrSpendInsufficientBalance) {
		t.Fatalf("expected ErrSpendInsufficientBalance, got %v", err)
	}

	var dbBal int64
	db.Raw("SELECT balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&dbBal)
	if dbBal != 50 {
		t.Fatalf("expected balance unchanged at 50, got %d", dbBal)
	}
}

func TestRedeem_WalletNotLinked(t *testing.T) {
	db := newTestDB(t)
	svc := &SpendService{db: db}

	userID := userIDForClaim(t, db)
	seedTachiBalance(t, db, userID, 200)
	// no web3 provider seeded

	_, err := svc.Redeem(context.Background(), userID, 100)
	if !errors.Is(err, ErrSpendWalletNotLinked) {
		t.Fatalf("expected ErrSpendWalletNotLinked, got %v", err)
	}

	var dbBal int64
	db.Raw("SELECT balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&dbBal)
	if dbBal != 200 {
		t.Fatalf("expected balance unchanged at 200, got %d", dbBal)
	}
}

func TestRedeem_BurnFailureRollback(t *testing.T) {
	db := newTestDB(t)
	burnCaller := &mockBurnCaller{err: errors.New("burn reverted")}
	svc := &SpendService{db: db, burnCaller: burnCaller}

	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedTachiBalance(t, db, userID, 300)

	_, err := svc.Redeem(context.Background(), userID, 100)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	var dbBal int64
	db.Raw("SELECT balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&dbBal)
	if dbBal != 300 {
		t.Fatalf("expected balance rolled back to 300, got %d", dbBal)
	}
}
