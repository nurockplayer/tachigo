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
	"github.com/tachigo/tachigo/internal/models"
)

type mockMintCaller struct {
	txHash   string
	txHashes []string
	err      error
	calls    []mintCall
}

type mintCall struct {
	toAddr string
	amount int64
}

func (m *mockMintCaller) MintOnChain(_ context.Context, toAddr string, amount int64) (string, error) {
	m.calls = append(m.calls, mintCall{toAddr: toAddr, amount: amount})
	if m.err != nil {
		return "", m.err
	}
	callIdx := len(m.calls) - 1
	if callIdx < len(m.txHashes) {
		return m.txHashes[callIdx], nil
	}
	return m.txHash, nil
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

func (m *inspectingMintCaller) MintOnChain(_ context.Context, _ string, _ int64) (string, error) {
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
	mintCaller := &mockMintCaller{txHash: "0xabc"}
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
	if len(mintCaller.calls) != 1 {
		t.Fatalf("expected 1 mint call, got %d", len(mintCaller.calls))
	}
	if mintCaller.calls[0].amount != 150 {
		t.Fatalf("expected mint amount=150, got %d", mintCaller.calls[0].amount)
	}
}

func TestClaim_PartialAmount(t *testing.T) {
	db := newTestDB(t)
	mintCaller := &mockMintCaller{txHash: "0xdef"}
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
	mintCaller := &mockMintCaller{txHashes: []string{"0x987a", "0x987b"}}
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
	mintCaller := &mockMintCaller{txHash: "0x123"}
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
	if len(mintCaller.calls) != 1 {
		t.Fatalf("expected 1 mint call, got %d", len(mintCaller.calls))
	}
	if mintCaller.calls[0].toAddr != "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045" {
		t.Fatalf("unexpected mint address: %s", mintCaller.calls[0].toAddr)
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
	mintCaller := &mockMintCaller{txHash: "0xclaimtx"}
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

func TestClaim_MintFailureLeavesDBUnchanged(t *testing.T) {
	db := newTestDB(t)
	mintCaller := &mockMintCaller{err: errors.New("mint reverted")}
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
