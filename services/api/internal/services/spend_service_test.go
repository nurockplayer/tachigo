package services

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── mock BurnCaller ──────────────────────────────────────────────────────────

type mockBurnCaller struct {
	txHash string
	err    error
	calls  []burnCall
	after  func()
}

type burnCall struct {
	fromAddr string
	amount   int64
}

func (m *mockBurnCaller) BurnOnChain(_ context.Context, fromAddr string, amount int64) (string, error) {
	m.calls = append(m.calls, burnCall{fromAddr: fromAddr, amount: amount})
	if m.after != nil {
		m.after()
	}
	return m.txHash, m.err
}

// ── mock TachiyaClient ───────────────────────────────────────────────────────

type mockTachiyaClient struct {
	voucherCode string
	err         error
	calls       []tachiyaCall
	ctxErr      error
	beforeReply func()
}

type tachiyaCall struct {
	couponID string
	tcgCost  int64
}

func (m *mockTachiyaClient) RedeemCoupon(ctx context.Context, couponID string, tcgCost int64) (string, error) {
	m.calls = append(m.calls, tachiyaCall{couponID: couponID, tcgCost: tcgCost})
	m.ctxErr = ctx.Err()
	if m.beforeReply != nil {
		m.beforeReply()
	}
	return m.voucherCode, m.err
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
	tachiyaClient := &mockTachiyaClient{voucherCode: "VOUCHER-XYZ"}
	svc := &SpendService{db: db, burnCaller: burnCaller, tachiyaClient: tachiyaClient}

	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedTachiBalance(t, db, userID, 500)

	newBal, voucherCode, err := svc.Redeem(context.Background(), userID, "coupon-123", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newBal != 400 {
		t.Fatalf("expected newBalance=400, got %d", newBal)
	}
	if voucherCode != "VOUCHER-XYZ" {
		t.Fatalf("expected voucherCode=VOUCHER-XYZ, got %s", voucherCode)
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
	if len(tachiyaClient.calls) != 1 {
		t.Fatalf("expected 1 tachiya call, got %d", len(tachiyaClient.calls))
	}
	if tachiyaClient.calls[0].couponID != "coupon-123" {
		t.Fatalf("expected couponID=coupon-123, got %s", tachiyaClient.calls[0].couponID)
	}
	if tachiyaClient.calls[0].tcgCost != 100 {
		t.Fatalf("expected tcgCost=100, got %d", tachiyaClient.calls[0].tcgCost)
	}

	var dbBal int64
	db.Raw("SELECT balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&dbBal)
	if dbBal != 400 {
		t.Fatalf("expected db balance=400, got %d", dbBal)
	}

	var redemptionStatus string
	db.Raw("SELECT status FROM coupon_redemptions WHERE user_id = ?", userID).Scan(&redemptionStatus)
	if redemptionStatus != "redeemed" {
		t.Fatalf("expected coupon_redemption status=redeemed, got %q", redemptionStatus)
	}

	var voucher string
	db.Raw("SELECT voucher_code FROM coupon_redemptions WHERE user_id = ?", userID).Scan(&voucher)
	if voucher != "VOUCHER-XYZ" {
		t.Fatalf("expected voucher_code=VOUCHER-XYZ, got %q", voucher)
	}
}

func TestRedeem_TachiyaIssuedVoucherButPersistFailureReturnsErrorAndLeavesPending(t *testing.T) {
	db := newTestDB(t)
	updateErr := errors.New("forced redeemed voucher update failure")
	if err := db.Callback().Update().Before("gorm:update").Register("fail_coupon_redemption_redeemed_update", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "coupon_redemptions" {
			tx.AddError(updateErr)
		}
	}); err != nil {
		t.Fatalf("register update callback: %v", err)
	}

	burnCaller := &mockBurnCaller{txHash: "0xburn123"}
	tachiyaClient := &mockTachiyaClient{voucherCode: "VOUCHER-XYZ"}
	svc := &SpendService{db: db, burnCaller: burnCaller, tachiyaClient: tachiyaClient}

	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedTachiBalance(t, db, userID, 500)

	_, _, err := svc.Redeem(context.Background(), userID, "coupon-123", 100)
	if !errors.Is(err, updateErr) {
		t.Fatalf("expected redeemed voucher persistence error, got %v", err)
	}

	var status string
	var voucher sql.NullString
	var txHash string
	if err := db.Raw("SELECT status, voucher_code, tx_hash FROM coupon_redemptions WHERE user_id = ?", userID).Row().Scan(&status, &voucher, &txHash); err != nil {
		t.Fatalf("scan coupon_redemption: %v", err)
	}
	if status != "pending" {
		t.Fatalf("expected status to remain pending for reconciliation, got %q", status)
	}
	if txHash != "0xburn123" {
		t.Fatalf("expected pending redemption to retain burn tx_hash for reconciliation, got %q", txHash)
	}
	if voucher.Valid {
		t.Fatalf("expected voucher_code to remain unset after persist failure, got %q", voucher.String)
	}
}

func TestRedeem_TachiyaCallOutlivesRequestCancellationAfterBurn(t *testing.T) {
	db := newTestDB(t)
	reqCtx, cancelReq := context.WithCancel(context.Background())
	burnCaller := &mockBurnCaller{
		txHash: "0xburn123",
		after:  cancelReq,
	}
	tachiyaClient := &mockTachiyaClient{voucherCode: "VOUCHER-XYZ"}
	svc := &SpendService{db: db, burnCaller: burnCaller, tachiyaClient: tachiyaClient}

	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedTachiBalance(t, db, userID, 500)

	if _, _, err := svc.Redeem(reqCtx, userID, "coupon-123", 100); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tachiyaClient.ctxErr != nil {
		t.Fatalf("tachiya context should not inherit canceled request context, got %v", tachiyaClient.ctxErr)
	}
}

func TestRedeem_InsufficientBalance(t *testing.T) {
	db := newTestDB(t)
	svc := &SpendService{db: db}

	userID := userIDForClaim(t, db)
	seedTachiBalance(t, db, userID, 50)

	_, _, err := svc.Redeem(context.Background(), userID, "", 100)
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

	_, _, err := svc.Redeem(context.Background(), userID, "", 100)
	if !errors.Is(err, ErrSpendWalletNotLinked) {
		t.Fatalf("expected ErrSpendWalletNotLinked, got %v", err)
	}

	var dbBal int64
	db.Raw("SELECT balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&dbBal)
	if dbBal != 200 {
		t.Fatalf("expected balance unchanged at 200, got %d", dbBal)
	}
}

func TestRedeem_BurnBroadcastedButReceiptUnknown(t *testing.T) {
	db := newTestDB(t)
	// Simulates: tx was broadcast (txHash returned) but WaitMined failed
	burnCaller := &mockBurnCaller{txHash: "0xbroadcasted", err: errors.New("context deadline exceeded")}
	svc := &SpendService{db: db, burnCaller: burnCaller}

	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedTachiBalance(t, db, userID, 300)

	_, _, err := svc.Redeem(context.Background(), userID, "coupon-unknown", 100)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	// DB reservation must NOT be rolled back: chain may have burned the tokens
	var dbBal int64
	db.Raw("SELECT balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&dbBal)
	if dbBal != 200 {
		t.Fatalf("expected balance kept at 200 (no rollback), got %d", dbBal)
	}

	var status string
	var couponID string
	var amount int64
	var txHash string
	if err := db.Raw("SELECT status, coupon_id, amount, tx_hash FROM coupon_redemptions WHERE user_id = ?", userID).Row().Scan(&status, &couponID, &amount, &txHash); err != nil {
		t.Fatalf("scan coupon_redemption: %v", err)
	}
	if status != "pending" {
		t.Fatalf("expected status=pending for reconciliation, got %q", status)
	}
	if couponID != "coupon-unknown" {
		t.Fatalf("expected coupon_id=coupon-unknown, got %q", couponID)
	}
	if amount != 100 {
		t.Fatalf("expected amount=100, got %d", amount)
	}
	if txHash != "0xbroadcasted" {
		t.Fatalf("expected tx_hash=0xbroadcasted, got %q", txHash)
	}
}

func TestRedeem_BroadcastUnknownPendingRecordFailureLogsReconciliationMetadata(t *testing.T) {
	db := newTestDB(t)
	createErr := errors.New("forced coupon redemption create failure")
	if err := db.Callback().Create().Before("gorm:create").Register("fail_coupon_redemption_create", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "coupon_redemptions" {
			tx.AddError(createErr)
		}
	}); err != nil {
		t.Fatalf("register create callback: %v", err)
	}

	var logBuf bytes.Buffer
	previousLogOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(previousLogOutput)

	burnCaller := &mockBurnCaller{txHash: "0xbroadcasted", err: errors.New("context deadline exceeded")}
	svc := &SpendService{db: db, burnCaller: burnCaller}

	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedTachiBalance(t, db, userID, 300)

	_, _, err := svc.Redeem(context.Background(), userID, "coupon-unknown", 100)
	if !errors.Is(err, createErr) {
		t.Fatalf("expected create error, got %v", err)
	}

	logOutput := logBuf.String()
	wantFragments := []string{
		"broadcast-unknown burn redemption record failed",
		"user_id=" + userID.String(),
		"coupon_id=coupon-unknown",
		"amount=100",
		"tx_hash=0xbroadcasted",
		"receipt_err=context deadline exceeded",
		"db_err=forced coupon redemption create failure",
	}
	for _, fragment := range wantFragments {
		if !strings.Contains(logOutput, fragment) {
			t.Fatalf("log output missing %q:\n%s", fragment, logOutput)
		}
	}
}

func TestRedeem_BurnFailureRollback(t *testing.T) {
	db := newTestDB(t)
	burnCaller := &mockBurnCaller{err: errors.New("burn reverted")}
	svc := &SpendService{db: db, burnCaller: burnCaller}

	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedTachiBalance(t, db, userID, 300)

	_, _, err := svc.Redeem(context.Background(), userID, "", 100)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	var dbBal int64
	db.Raw("SELECT balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&dbBal)
	if dbBal != 300 {
		t.Fatalf("expected balance rolled back to 300, got %d", dbBal)
	}
}

func TestRedeem_TachiyaFailure_ReturnsErrorAndRecordsCompensation(t *testing.T) {
	db := newTestDB(t)
	burnCaller := &mockBurnCaller{txHash: "0xburn123"}
	tachiyaClient := &mockTachiyaClient{err: errors.New("tachiya unavailable")}
	svc := &SpendService{db: db, burnCaller: burnCaller, tachiyaClient: tachiyaClient}

	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedTachiBalance(t, db, userID, 300)

	_, _, err := svc.Redeem(context.Background(), userID, "coupon-123", 100)
	if !errors.Is(err, ErrTachiyaRedeemFailed) {
		t.Fatalf("expected ErrTachiyaRedeemFailed, got %v", err)
	}

	var dbBal int64
	db.Raw("SELECT balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&dbBal)
	if dbBal != 200 {
		t.Fatalf("expected balance=200 (burn not rolled back), got %d", dbBal)
	}

	var status string
	var errorMessage sql.NullString
	db.Raw("SELECT status, error_message FROM coupon_redemptions WHERE user_id = ?", userID).Row().Scan(&status, &errorMessage)
	if status != "compensation-needed" {
		t.Fatalf("expected status=compensation-needed, got %q", status)
	}
	if !errorMessage.Valid || errorMessage.String != "tachiya unavailable" {
		t.Fatalf("expected error_message=tachiya unavailable, got valid=%v value=%q", errorMessage.Valid, errorMessage.String)
	}
}

func TestCouponRedemptionSchema(t *testing.T) {
	db := newTestDB(t)
	userID := userIDForClaim(t, db)
	redemptionID := uuid.NewString()
	err := db.Exec(`
		INSERT INTO coupon_redemptions (id, user_id, coupon_id, amount, tx_hash, status, created_at, updated_at)
		VALUES (?, ?, 'coupon-123', 100, '0xabc', 'pending',
		        CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, redemptionID, userID.String()).Error
	if err != nil {
		t.Fatalf("coupon_redemptions table not ready: %v", err)
	}
}

func TestCouponRedemptionSchemaRequiresExistingUser(t *testing.T) {
	db := newTestDB(t)
	redemptionID := uuid.NewString()
	missingUserID := uuid.NewString()
	err := db.Exec(`
		INSERT INTO coupon_redemptions (id, user_id, coupon_id, amount, tx_hash, status, created_at, updated_at)
		VALUES (?, ?, 'coupon-123', 100, '0xabc', 'pending',
		        CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, redemptionID, missingUserID).Error
	if err == nil {
		t.Fatal("expected coupon_redemptions.user_id to require an existing user")
	}
}
