package services

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/tachigo/tachigo/internal/config"
	contractpkg "github.com/tachigo/tachigo/internal/contract"
	"github.com/tachigo/tachigo/internal/models"
)

type mockMintCaller struct {
	broadcastHash   string
	broadcastHashes []string
	broadcastErr    error
	waitErr         error
	broadcastCalls  []mintCall
	waitCalls       []string
}

type mintCall struct {
	toAddr string
	amount int64
}

func (m *mockMintCaller) MintBroadcastOnChain(_ context.Context, toAddr string, amount int64) (string, error) {
	m.broadcastCalls = append(m.broadcastCalls, mintCall{toAddr: toAddr, amount: amount})
	if m.broadcastErr != nil {
		return "", m.broadcastErr
	}
	callIdx := len(m.broadcastCalls) - 1
	if callIdx < len(m.broadcastHashes) {
		return m.broadcastHashes[callIdx], nil
	}
	return m.broadcastHash, nil
}

func (m *mockMintCaller) WaitMintReceiptOnChain(_ context.Context, txHash string) error {
	m.waitCalls = append(m.waitCalls, txHash)
	return m.waitErr
}

type inspectingMintCaller struct {
	db              *gorm.DB
	userID          uuid.UUID
	channelID       string
	wantSpendable   int64
	observed        int64
	observedMatches bool
}

type mockMintContract struct {
	broadcastHash  string
	broadcastErr   error
	waitErr        error
	broadcastCalls []mintContractCall
	waitCalls      []string
}

type mintContractCall struct {
	toAddr string
	amount string
}

func (m *mockMintContract) MintBroadcast(_ context.Context, toAddr common.Address, amount *big.Int, _ *ecdsa.PrivateKey) (string, error) {
	m.broadcastCalls = append(m.broadcastCalls, mintContractCall{
		toAddr: toAddr.Hex(),
		amount: amount.String(),
	})
	return m.broadcastHash, m.broadcastErr
}

func (m *mockMintContract) WaitMintReceipt(_ context.Context, txHash string) error {
	m.waitCalls = append(m.waitCalls, txHash)
	return m.waitErr
}

func (m *inspectingMintCaller) MintBroadcastOnChain(_ context.Context, _ string, _ int64) (string, error) {
	if err := m.db.Raw(
		"SELECT spendable_balance FROM points_ledgers WHERE user_id = ? AND channel_id = ?",
		m.userID,
		m.channelID,
	).Scan(&m.observed).Error; err != nil {
		return "", err
	}
	m.observedMatches = m.observed == m.wantSpendable
	return "0xreserved", nil
}

func (m *inspectingMintCaller) WaitMintReceiptOnChain(_ context.Context, _ string) error {
	return nil
}

func testSignerKeyHex(t *testing.T) string {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate test signer key: %v", err)
	}
	return hex.EncodeToString(crypto.FromECDSA(key))
}

func newFileClaimTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "claim.db")), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		TranslateError: true,
	})
	if err != nil {
		t.Fatalf("open file test db: %v", err)
	}
	if err := db.Exec(`PRAGMA foreign_keys = ON`).Error; err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if err := migrateTestDB(db); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return db
}

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

func seedWeb3Provider(t *testing.T, db *gorm.DB, userID uuid.UUID, addr string) {
	t.Helper()
	if err := db.Create(&models.AuthProvider{
		UserID:     userID,
		Provider:   models.ProviderWeb3,
		ProviderID: addr,
	}).Error; err != nil {
		t.Fatalf("seedWeb3Provider: %v", err)
	}
}

func TestGetTachiBalance_Zero(t *testing.T) {
	db := newTestDB(t)
	svc := &ClaimService{db: db}
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
	mintCaller := &mockMintCaller{broadcastHash: "0xabc"}
	svc := &ClaimService{db: db, mintCaller: mintCaller}
	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedLedger(t, db, userID, "ch1", 100)
	seedLedger(t, db, userID, "ch2", 50)

	newBal, err := svc.Claim(context.Background(), userID, 0) // claim all
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
	if len(mintCaller.broadcastCalls) != 1 {
		t.Fatalf("expected 1 mint call, got %d", len(mintCaller.broadcastCalls))
	}
	if mintCaller.broadcastCalls[0].amount != 150 {
		t.Fatalf("expected mint amount=150, got %d", mintCaller.broadcastCalls[0].amount)
	}
}

func TestClaim_PartialAmount(t *testing.T) {
	db := newTestDB(t)
	mintCaller := &mockMintCaller{broadcastHash: "0xdef"}
	svc := &ClaimService{db: db, mintCaller: mintCaller}
	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedLedger(t, db, userID, "ch1", 100)

	newBal, err := svc.Claim(context.Background(), userID, 40)
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

func TestClaim_ReservesSpendableBeforeMint(t *testing.T) {
	db := newFileClaimTestDB(t)
	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedLedger(t, db, userID, "ch1", 80)
	mintCaller := &inspectingMintCaller{
		db:            db,
		userID:        userID,
		channelID:     "ch1",
		wantSpendable: 30,
	}
	svc := &ClaimService{db: db, mintCaller: mintCaller}

	if _, err := svc.Claim(context.Background(), userID, 50); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mintCaller.observedMatches {
		t.Fatalf("expected spendable to be reserved before mint, observed %d", mintCaller.observed)
	}
}

func TestClaim_InsufficientBalance(t *testing.T) {
	db := newTestDB(t)
	svc := &ClaimService{db: db}
	userID := userIDForClaim(t, db)
	seedLedger(t, db, userID, "ch1", 30)

	_, err := svc.Claim(context.Background(), userID, 100)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if err != ErrClaimInsufficientBalance {
		t.Fatalf("expected ErrClaimInsufficientBalance, got %v", err)
	}
}

func TestClaim_NoLedgers(t *testing.T) {
	db := newTestDB(t)
	svc := &ClaimService{db: db}
	userID := userIDForClaim(t, db)

	// amount=0 with no ledgers → claimAmount=0 → ErrClaimAmountInvalid
	_, err := svc.Claim(context.Background(), userID, 0)
	if !errors.Is(err, ErrClaimAmountInvalid) {
		t.Fatalf("expected ErrClaimAmountInvalid, got %v", err)
	}
}

func TestClaim_AccumulatesOnSecondClaim(t *testing.T) {
	db := newTestDB(t)
	mintCaller := &mockMintCaller{broadcastHashes: []string{"0x987a", "0x987b"}}
	svc := &ClaimService{db: db, mintCaller: mintCaller}
	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedLedger(t, db, userID, "ch1", 200)

	bal1, err1 := svc.Claim(context.Background(), userID, 100)
	if err1 != nil {
		t.Fatalf("first claim unexpected error: %v", err1)
	}
	if bal1 != 100 {
		t.Fatalf("expected first tachi_balance=100, got %d", bal1)
	}
	newBal, err := svc.Claim(context.Background(), userID, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newBal != 150 {
		t.Fatalf("expected tachi_balance=150, got %d", newBal)
	}
}

func TestClaim_MintSuccessUpdatesDB(t *testing.T) {
	db := newTestDB(t)
	mintCaller := &mockMintCaller{broadcastHash: "0x123"}
	svc := &ClaimService{db: db, mintCaller: mintCaller}
	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedLedger(t, db, userID, "ch1", 80)

	newBal, err := svc.Claim(context.Background(), userID, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newBal != 50 {
		t.Fatalf("expected tachi_balance=50, got %d", newBal)
	}
	if len(mintCaller.broadcastCalls) != 1 {
		t.Fatalf("expected 1 mint call, got %d", len(mintCaller.broadcastCalls))
	}
	if mintCaller.broadcastCalls[0].toAddr != "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045" {
		t.Fatalf("unexpected mint address: %s", mintCaller.broadcastCalls[0].toAddr)
	}

	var remaining int64
	if err := db.Raw("SELECT spendable_balance FROM points_ledgers WHERE user_id = ? AND channel_id = 'ch1'", userID).Scan(&remaining).Error; err != nil {
		t.Fatalf("query remaining: %v", err)
	}
	if remaining != 30 {
		t.Fatalf("expected remaining spendable=30, got %d", remaining)
	}
}

func TestClaim_PersistsClaimAndClaimItems(t *testing.T) {
	db := newTestDB(t)
	mintCaller := &mockMintCaller{broadcastHash: "0xclaimtx"}
	svc := &ClaimService{db: db, mintCaller: mintCaller}
	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedLedger(t, db, userID, "ch1", 70)
	seedLedger(t, db, userID, "ch2", 40)

	if _, err := svc.Claim(context.Background(), userID, 100); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var claimCount int64
	if err := db.Model(&models.Claim{}).Where("user_id = ?", userID).Count(&claimCount).Error; err != nil {
		t.Fatalf("count claims: %v", err)
	}
	if claimCount != 1 {
		t.Fatalf("expected 1 claim row, got %d", claimCount)
	}

	var itemCount int64
	if err := db.Model(&models.ClaimItem{}).Joins("JOIN claims ON claims.id = claim_items.claim_id").Where("claims.user_id = ?", userID).Count(&itemCount).Error; err != nil {
		t.Fatalf("count claim_items: %v", err)
	}
	if itemCount != 2 {
		t.Fatalf("expected 2 claim_items rows, got %d", itemCount)
	}

	var wrongUserCount int64
	if err := db.Model(&models.ClaimItem{}).Where("claim_user_id <> ?", userID).Count(&wrongUserCount).Error; err != nil {
		t.Fatalf("count wrong claim_user_id rows: %v", err)
	}
	if wrongUserCount != 0 {
		t.Fatalf("expected all claim_items.claim_user_id = user_id, got %d mismatches", wrongUserCount)
	}
}

func TestClaim_BroadcastFailureLeavesDBUnchanged(t *testing.T) {
	db := newTestDB(t)
	mintCaller := &mockMintCaller{broadcastErr: errors.New("send mint tx: rpc unavailable")}
	svc := &ClaimService{db: db, mintCaller: mintCaller}
	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedLedger(t, db, userID, "ch1", 80)

	_, err := svc.Claim(context.Background(), userID, 50)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	var remaining int64
	if err := db.Raw("SELECT spendable_balance FROM points_ledgers WHERE user_id = ? AND channel_id = 'ch1'", userID).Scan(&remaining).Error; err != nil {
		t.Fatalf("query remaining: %v", err)
	}
	if remaining != 80 {
		t.Fatalf("expected remaining spendable=80, got %d", remaining)
	}

	var balanceCount int64
	if err := db.Raw("SELECT COUNT(*) FROM tachi_balances WHERE user_id = ?", userID).Scan(&balanceCount).Error; err != nil {
		t.Fatalf("query balances: %v", err)
	}
	if balanceCount != 0 {
		t.Fatalf("expected no tachi balance rows, got %d", balanceCount)
	}

	var claimCount int64
	if err := db.Model(&models.Claim{}).Where("user_id = ?", userID).Count(&claimCount).Error; err != nil {
		t.Fatalf("count claims: %v", err)
	}
	if claimCount != 0 {
		t.Fatalf("expected no claim rows after pre-broadcast failure, got %d", claimCount)
	}
}

func TestClaim_WaitFailureKeepsBroadcastClaimAndDoesNotRollback(t *testing.T) {
	db := newTestDB(t)
	mintCaller := &mockMintCaller{
		broadcastHash: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		waitErr:       errors.New("wait mint receipt: context deadline exceeded"),
	}
	svc := &ClaimService{db: db, mintCaller: mintCaller}
	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedLedger(t, db, userID, "ch1", 80)

	_, err := svc.Claim(context.Background(), userID, 50)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	var remaining int64
	if err := db.Raw("SELECT spendable_balance FROM points_ledgers WHERE user_id = ? AND channel_id = 'ch1'", userID).Scan(&remaining).Error; err != nil {
		t.Fatalf("query remaining: %v", err)
	}
	if remaining != 30 {
		t.Fatalf("expected reserved spendable=30, got %d", remaining)
	}

	var claim models.Claim
	if err := db.Where("user_id = ?", userID).First(&claim).Error; err != nil {
		t.Fatalf("load claim: %v", err)
	}
	if claim.Status != models.ClaimStatusBroadcast {
		t.Fatalf("expected claim status broadcast, got %s", claim.Status)
	}
	if claim.TxHash == nil || *claim.TxHash != mintCaller.broadcastHash {
		t.Fatalf("expected txHash %s, got %v", mintCaller.broadcastHash, claim.TxHash)
	}
	if claim.BroadcastAt == nil {
		t.Fatal("expected broadcast_at to be set")
	}

	var balanceCount int64
	if err := db.Raw("SELECT COUNT(*) FROM tachi_balances WHERE user_id = ?", userID).Scan(&balanceCount).Error; err != nil {
		t.Fatalf("query balances: %v", err)
	}
	if balanceCount != 0 {
		t.Fatalf("expected no tachi balance rows, got %d", balanceCount)
	}
}

func TestClaim_BroadcastPersistFailureReturnsTxHashForReconciliation(t *testing.T) {
	db := newTestDB(t)
	persistErr := errors.New("persist broadcast failed")
	errHash := "0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	if err := db.Callback().Update().Before("gorm:update").Register("fail_claim_broadcast_update", func(tx *gorm.DB) {
		if tx.Statement.Table != "claims" {
			return
		}
		updates, ok := tx.Statement.Dest.(map[string]interface{})
		if !ok || updates["status"] != models.ClaimStatusBroadcast {
			return
		}
		tx.AddError(persistErr)
	}); err != nil {
		t.Fatalf("register update callback: %v", err)
	}

	mintCaller := &mockMintCaller{broadcastHash: errHash}
	svc := &ClaimService{db: db, mintCaller: mintCaller}
	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedLedger(t, db, userID, "ch1", 80)

	_, err := svc.Claim(context.Background(), userID, 50)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	var recordErr *ClaimBroadcastRecordError
	if !errors.As(err, &recordErr) {
		t.Fatalf("expected ClaimBroadcastRecordError, got %T: %v", err, err)
	}
	if recordErr.TxHash != errHash {
		t.Fatalf("expected txHash %s, got %s", errHash, recordErr.TxHash)
	}
	if recordErr.ClaimID == uuid.Nil {
		t.Fatal("expected claim ID to be populated")
	}
	if recordErr.UserID != userID {
		t.Fatalf("expected userID %s, got %s", userID, recordErr.UserID)
	}
	if !errors.Is(err, persistErr) {
		t.Fatalf("expected error to wrap persistErr, got %v", err)
	}
	if len(mintCaller.waitCalls) != 0 {
		t.Fatalf("receipt wait should not run when broadcast state cannot be persisted, got %d calls", len(mintCaller.waitCalls))
	}
}

func TestClaim_ReceiptFailedMarksFailedAndCompensates(t *testing.T) {
	db := newTestDB(t)
	mintCaller := &mockMintCaller{
		broadcastHash: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		waitErr:       contractpkg.ErrMintReceiptStatusFailed,
	}
	svc := &ClaimService{db: db, mintCaller: mintCaller}
	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedLedger(t, db, userID, "ch1", 80)

	_, err := svc.Claim(context.Background(), userID, 50)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	var remaining int64
	if err := db.Raw("SELECT spendable_balance FROM points_ledgers WHERE user_id = ? AND channel_id = 'ch1'", userID).Scan(&remaining).Error; err != nil {
		t.Fatalf("query remaining: %v", err)
	}
	if remaining != 80 {
		t.Fatalf("expected compensated spendable=80, got %d", remaining)
	}

	var claim models.Claim
	if err := db.Where("user_id = ?", userID).First(&claim).Error; err != nil {
		t.Fatalf("load claim: %v", err)
	}
	if claim.Status != models.ClaimStatusFailed {
		t.Fatalf("expected claim status failed, got %s", claim.Status)
	}
	if claim.FailedAt == nil {
		t.Fatal("expected failed_at to be set")
	}

	var txCount int64
	if err := db.Model(&models.PointsTransaction{}).Where("ledger_id IN (SELECT id FROM points_ledgers WHERE user_id = ?)", userID).Count(&txCount).Error; err != nil {
		t.Fatalf("count points transactions: %v", err)
	}
	if txCount != 2 {
		t.Fatalf("expected claim debit + compensation credit transactions, got %d", txCount)
	}
}

func TestClaim_FinalizeFailureMarksFinalizeFailedStatus(t *testing.T) {
	db := newTestDB(t)
	persistErr := errors.New("db finalize failed")

	if err := db.Callback().Update().Before("gorm:update").Register("fail_finalize_update", func(tx *gorm.DB) {
		if tx.Statement.Table != "claims" {
			return
		}
		updates, ok := tx.Statement.Dest.(map[string]interface{})
		if !ok || updates["status"] != models.ClaimStatusConfirmed {
			return
		}
		tx.AddError(persistErr)
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}

	mintCaller := &mockMintCaller{broadcastHash: "0xfinalizeFailHash"}
	svc := &ClaimService{db: db, mintCaller: mintCaller}
	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedLedger(t, db, userID, "ch1", 100)

	_, err := svc.Claim(context.Background(), userID, 60)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !errors.Is(err, persistErr) {
		t.Fatalf("expected error to wrap persistErr, got %v", err)
	}

	var claim models.Claim
	if err := db.Where("user_id = ?", userID).First(&claim).Error; err != nil {
		t.Fatalf("load claim: %v", err)
	}
	if claim.Status != models.ClaimStatusFinalizeFailed {
		t.Fatalf("expected status finalize_failed, got %s", claim.Status)
	}
	if claim.FinalizeFailedAt == nil {
		t.Fatal("expected finalize_failed_at to be set")
	}
	if claim.ErrorMessage == nil || *claim.ErrorMessage == "" {
		t.Fatal("expected error_message to be populated")
	}
	if claim.TxHash == nil || *claim.TxHash != "0xfinalizeFailHash" {
		t.Fatalf("expected tx_hash to be preserved, got %v", claim.TxHash)
	}

	var spendable int64
	if err := db.Raw("SELECT spendable_balance FROM points_ledgers WHERE user_id = ? AND channel_id = 'ch1'", userID).Scan(&spendable).Error; err != nil {
		t.Fatalf("query spendable: %v", err)
	}
	if spendable != 40 {
		t.Fatalf("expected spendable=40 (deducted, not compensated), got %d", spendable)
	}
}

func TestClaim_FinalizeIdempotent(t *testing.T) {
	db := newTestDB(t)
	mintCaller := &mockMintCaller{broadcastHash: "0xidempotentHash"}
	svc := &ClaimService{db: db, mintCaller: mintCaller}
	userID := userIDForClaim(t, db)
	seedWeb3Provider(t, db, userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	seedLedger(t, db, userID, "ch1", 100)

	bal1, err := svc.Claim(context.Background(), userID, 60)
	if err != nil {
		t.Fatalf("first claim unexpected error: %v", err)
	}
	if bal1 != 60 {
		t.Fatalf("expected tachi_balance=60, got %d", bal1)
	}

	var claim models.Claim
	if err := db.Where("user_id = ?", userID).First(&claim).Error; err != nil {
		t.Fatalf("load claim: %v", err)
	}

	reservation := claimReservation{
		claimID: claim.ID,
		userID:  userID,
		amount:  60,
	}

	var bal2 int64
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		bal2, err = svc.finalizeClaim(tx, reservation, "0xidempotentHash")
		return err
	}); err != nil {
		t.Fatalf("second finalizeClaim unexpected error: %v", err)
	}

	if bal2 != 60 {
		t.Fatalf("expected tachi_balance=60 after retry, got %d", bal2)
	}

	var finalBal int64
	if err := db.Raw("SELECT balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&finalBal).Error; err != nil {
		t.Fatalf("query final balance: %v", err)
	}
	if finalBal != 60 {
		t.Fatalf("expected final tachi_balance=60, got %d", finalBal)
	}
}

func TestClaim_FinalizeConfirmedWithoutBalanceReturnsError(t *testing.T) {
	db := newTestDB(t)
	svc := &ClaimService{db: db}
	userID := userIDForClaim(t, db)
	txHash := "0xconfirmedMissingBalanceHash"

	if err := db.Exec(`
		INSERT INTO claims (id, user_id, wallet_addr, amount, status, tx_hash, confirmed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, uuid.NewString(), userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 60, models.ClaimStatusConfirmed, txHash).Error; err != nil {
		t.Fatalf("insert claim: %v", err)
	}

	var claim models.Claim
	if err := db.Where("user_id = ?", userID).First(&claim).Error; err != nil {
		t.Fatalf("load claim: %v", err)
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		_, err := svc.finalizeClaim(tx, claimReservation{
			claimID: claim.ID,
			userID:  userID,
			amount:  60,
		}, txHash)
		return err
	})
	if err == nil {
		t.Fatal("expected confirmed claim without tachi balance to return error")
	}
}

func TestClaim_FinalizeFailedRetryConfirmsAndCreditsOnce(t *testing.T) {
	db := newTestDB(t)
	svc := &ClaimService{db: db}
	userID := userIDForClaim(t, db)
	txHash := "0xretryHash"

	if err := db.Exec(`
		INSERT INTO claims (id, user_id, wallet_addr, amount, status, tx_hash, error_message, broadcast_at, finalize_failed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, uuid.NewString(), userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 60, models.ClaimStatusFinalizeFailed, txHash, "db finalize failed").Error; err != nil {
		t.Fatalf("insert claim: %v", err)
	}

	var claim models.Claim
	if err := db.Where("user_id = ?", userID).First(&claim).Error; err != nil {
		t.Fatalf("load claim: %v", err)
	}

	reservation := claimReservation{
		claimID: claim.ID,
		userID:  userID,
		amount:  60,
	}

	var bal1 int64
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		bal1, err = svc.finalizeClaim(tx, reservation, txHash)
		return err
	}); err != nil {
		t.Fatalf("retry finalizeClaim unexpected error: %v", err)
	}
	if bal1 != 60 {
		t.Fatalf("expected tachi_balance=60 after retry, got %d", bal1)
	}

	var bal2 int64
	if err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		bal2, err = svc.finalizeClaim(tx, reservation, txHash)
		return err
	}); err != nil {
		t.Fatalf("second retry finalizeClaim unexpected error: %v", err)
	}
	if bal2 != 60 {
		t.Fatalf("expected tachi_balance=60 after second retry, got %d", bal2)
	}

	if err := db.Where("id = ?", claim.ID).First(&claim).Error; err != nil {
		t.Fatalf("reload claim: %v", err)
	}
	if claim.Status != models.ClaimStatusConfirmed {
		t.Fatalf("expected claim status confirmed, got %s", claim.Status)
	}
	if claim.ConfirmedAt == nil {
		t.Fatal("expected confirmed_at to be set")
	}
	if claim.FinalizeFailedAt == nil {
		t.Fatal("expected finalize_failed_at history to remain")
	}

	var finalBal int64
	if err := db.Raw("SELECT balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&finalBal).Error; err != nil {
		t.Fatalf("query final balance: %v", err)
	}
	if finalBal != 60 {
		t.Fatalf("expected final tachi_balance=60, got %d", finalBal)
	}
}

func TestClaim_FinalizeRejectsNonFinalizableStatus(t *testing.T) {
	cases := []struct {
		name   string
		status models.ClaimStatus
		txHash string
	}{
		{name: "failed", status: models.ClaimStatusFailed, txHash: "0xfailedHash"},
		{name: "pending", status: models.ClaimStatusPending, txHash: "0xpendingHash"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			svc := &ClaimService{db: db}
			userID := userIDForClaim(t, db)

			if err := db.Exec(`
				INSERT INTO claims (id, user_id, wallet_addr, amount, status, tx_hash, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			`, uuid.NewString(), userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 60, tc.status, tc.txHash).Error; err != nil {
				t.Fatalf("insert claim: %v", err)
			}

			var claim models.Claim
			if err := db.Where("user_id = ?", userID).First(&claim).Error; err != nil {
				t.Fatalf("load claim: %v", err)
			}

			err := db.Transaction(func(tx *gorm.DB) error {
				_, err := svc.finalizeClaim(tx, claimReservation{claimID: claim.ID, userID: userID, amount: 60}, tc.txHash)
				return err
			})
			if err == nil {
				t.Fatal("expected error but got nil")
			}

			var balanceCount int64
			if err := db.Raw("SELECT COUNT(*) FROM tachi_balances WHERE user_id = ?", userID).Scan(&balanceCount).Error; err != nil {
				t.Fatalf("count tachi_balances: %v", err)
			}
			if balanceCount != 0 {
				t.Fatalf("expected no tachi balance rows, got %d", balanceCount)
			}
		})
	}
}

func TestClaim_MarkFinalizeFailedDoesNotOverwriteConfirmed(t *testing.T) {
	db := newTestDB(t)
	svc := &ClaimService{db: db}
	userID := userIDForClaim(t, db)
	txHash := "0xconfirmedHash"

	if err := db.Exec(`
		INSERT INTO claims (id, user_id, wallet_addr, amount, status, tx_hash, confirmed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, uuid.NewString(), userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 60, models.ClaimStatusConfirmed, txHash).Error; err != nil {
		t.Fatalf("insert claim: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO tachi_balances (id, user_id, balance, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	`, uuid.NewString(), userID, 60).Error; err != nil {
		t.Fatalf("insert tachi_balance: %v", err)
	}

	var claim models.Claim
	if err := db.Where("user_id = ?", userID).First(&claim).Error; err != nil {
		t.Fatalf("load claim: %v", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return svc.markFinalizeFailedClaim(tx, claimReservation{
			claimID: claim.ID,
			userID:  userID,
			amount:  60,
		}, txHash, errors.New("late finalize marker"))
	}); err != nil {
		t.Fatalf("markFinalizeFailedClaim unexpected error: %v", err)
	}

	if err := db.Where("id = ?", claim.ID).First(&claim).Error; err != nil {
		t.Fatalf("reload claim: %v", err)
	}
	if claim.Status != models.ClaimStatusConfirmed {
		t.Fatalf("expected claim status confirmed, got %s", claim.Status)
	}
	if claim.FinalizeFailedAt != nil {
		t.Fatal("expected finalize_failed_at to remain nil")
	}

	var balance int64
	if err := db.Raw("SELECT balance FROM tachi_balances WHERE user_id = ?", userID).Scan(&balance).Error; err != nil {
		t.Fatalf("query tachi balance: %v", err)
	}
	if balance != 60 {
		t.Fatalf("expected tachi_balance=60, got %d", balance)
	}
}

func TestClaim_FinalizeTxHashMismatchDoesNotCredit(t *testing.T) {
	db := newTestDB(t)
	svc := &ClaimService{db: db}
	userID := userIDForClaim(t, db)

	if err := db.Exec(`
		INSERT INTO claims (id, user_id, wallet_addr, amount, status, tx_hash, broadcast_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, uuid.NewString(), userID, "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 60, models.ClaimStatusBroadcast, "0xhashA").Error; err != nil {
		t.Fatalf("insert claim: %v", err)
	}

	var claim models.Claim
	if err := db.Where("user_id = ?", userID).First(&claim).Error; err != nil {
		t.Fatalf("load claim: %v", err)
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		_, err := svc.finalizeClaim(tx, claimReservation{
			claimID: claim.ID,
			userID:  userID,
			amount:  60,
		}, "0xhashB")
		return err
	})
	if err == nil {
		t.Fatal("expected finalizeClaim error but got nil")
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return svc.markFinalizeFailedClaim(tx, claimReservation{
			claimID: claim.ID,
			userID:  userID,
			amount:  60,
		}, "0xhashB", errors.New("mismatch"))
	}); err == nil {
		t.Fatal("expected markFinalizeFailedClaim error but got nil")
	}

	var balanceCount int64
	if err := db.Raw("SELECT COUNT(*) FROM tachi_balances WHERE user_id = ?", userID).Scan(&balanceCount).Error; err != nil {
		t.Fatalf("count tachi_balances: %v", err)
	}
	if balanceCount != 0 {
		t.Fatalf("expected no tachi balance rows, got %d", balanceCount)
	}
}

func TestNewClaimService_InvalidContractAddressDoesNotInitializeToken(t *testing.T) {
	db := newTestDB(t)

	svc := NewClaimService(db, config.ContractConfig{
		TachiContractAddress: "0xnot-valid",
		SepoliaSignerKey:     "abcd",
	}, &ethclient.Client{})

	if svc.tachiToken != nil {
		t.Fatal("expected invalid contract address to leave tachiToken nil")
	}
}

func TestMintOnChain_BroadcastFailureReturnsEmptyHash(t *testing.T) {
	token := &mockMintContract{
		broadcastErr: errors.New("send mint tx: rpc unavailable"),
	}
	svc := &ClaimService{
		contractCfg: config.ContractConfig{SepoliaSignerKey: testSignerKeyHex(t)},
		tachiToken:  token,
	}

	txHash, err := svc.MintOnChain(context.Background(), "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 10)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if txHash != "" {
		t.Fatalf("expected empty txHash on broadcast failure, got %s", txHash)
	}
	if len(token.waitCalls) != 0 {
		t.Fatalf("wait should not be called when broadcast fails, got %d calls", len(token.waitCalls))
	}
}

func TestMintOnChain_WaitFailureReturnsBroadcastHash(t *testing.T) {
	token := &mockMintContract{
		broadcastHash: "0x1111111111111111111111111111111111111111111111111111111111111111",
		waitErr:       errors.New("wait mint receipt: context deadline exceeded"),
	}
	svc := &ClaimService{
		contractCfg: config.ContractConfig{SepoliaSignerKey: testSignerKeyHex(t)},
		tachiToken:  token,
	}

	txHash, err := svc.MintOnChain(context.Background(), "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 10)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if txHash != token.broadcastHash {
		t.Fatalf("expected txHash %s, got %s", token.broadcastHash, txHash)
	}
	if len(token.waitCalls) != 1 {
		t.Fatalf("expected 1 wait call, got %d", len(token.waitCalls))
	}
	if token.waitCalls[0] != token.broadcastHash {
		t.Fatalf("expected wait hash %s, got %s", token.broadcastHash, token.waitCalls[0])
	}
}

func TestMintOnChain_Success(t *testing.T) {
	token := &mockMintContract{
		broadcastHash: "0x2222222222222222222222222222222222222222222222222222222222222222",
	}
	svc := &ClaimService{
		contractCfg: config.ContractConfig{SepoliaSignerKey: testSignerKeyHex(t)},
		tachiToken:  token,
	}

	txHash, err := svc.MintOnChain(context.Background(), "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txHash != token.broadcastHash {
		t.Fatalf("expected txHash %s, got %s", token.broadcastHash, txHash)
	}
	if len(token.broadcastCalls) != 1 {
		t.Fatalf("expected 1 broadcast call, got %d", len(token.broadcastCalls))
	}
	if len(token.waitCalls) != 1 {
		t.Fatalf("expected 1 wait call, got %d", len(token.waitCalls))
	}
}
