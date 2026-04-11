# SpendService 實作計畫

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 實作 `SpendService.Redeem()` 讓用戶花費 `$TACHI` 換折價券（呼叫合約 `burn()`，成功後扣 DB），並掛上 `POST /api/v1/spend/redeem` endpoint。

**Architecture:** 與 `ClaimService` 鏡像——reserve-then-burn：先在 DB txn 內鎖定餘額、解析 wallet、扣除 `tachi_balances`，再呼叫合約 `BurnOnChain`；若 burn 失敗則 rollback DB。`TachiToken.Burn()` 仿照 `TachiToken.Mint()` 新增。

**Tech Stack:** Go 1.21, Gin, GORM, go-ethereum, SQLite (tests), PostgreSQL (prod)

---

## 檔案清單

| 操作 | 路徑 |
|---|---|
| 修改 | `backend/internal/contract/tachi_token.go` |
| 新增 | `backend/internal/services/spend_service.go` |
| 新增 | `backend/internal/services/spend_service_test.go` |
| 新增 | `backend/internal/handlers/spend_handler.go` |
| 修改 | `backend/internal/router/router.go` |
| 修改 | `backend/cmd/server/main.go` |

---

## Task 0：建立 feature branch

- [ ] **Step 1：從 develop 建立並切換 branch**

```bash
git checkout develop && git pull
git checkout -b feat/spend-service
```

期望：`Switched to a new branch 'feat/spend-service'`

---

## Task 1：新增 `TachiToken.Burn()`

**Files:**
- Modify: `backend/internal/contract/tachi_token.go`

- [ ] **Step 1：在 `tachi_token.go` 末尾新增 `Burn()` 方法**

在檔案最後（第 128 行後）加入：

```go
func (t *TachiToken) Burn(ctx context.Context, fromAddr common.Address, amount *big.Int, signerKey *ecdsa.PrivateKey) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.client == nil {
		return "", fmt.Errorf("eth client is nil")
	}
	if signerKey == nil {
		return "", fmt.Errorf("signer key is nil")
	}
	if amount == nil || amount.Sign() <= 0 {
		return "", fmt.Errorf("amount must be greater than zero")
	}

	fromSignerAddr := crypto.PubkeyToAddress(signerKey.PublicKey)
	data, err := t.abi.Pack("burn", fromAddr, amount)
	if err != nil {
		return "", fmt.Errorf("pack burn calldata: %w", err)
	}

	chainID, err := t.client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("get chain ID: %w", err)
	}

	nonce, err := t.client.PendingNonceAt(ctx, fromSignerAddr)
	if err != nil {
		return "", fmt.Errorf("get pending nonce: %w", err)
	}

	tipCap, err := t.client.SuggestGasTipCap(ctx)
	if err != nil {
		return "", fmt.Errorf("suggest gas tip cap: %w", err)
	}

	header, err := t.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("get latest header: %w", err)
	}
	if header.BaseFee == nil {
		return "", fmt.Errorf("latest header missing base fee")
	}

	feeCap := new(big.Int).Mul(header.BaseFee, big.NewInt(2))
	feeCap.Add(feeCap, tipCap)

	callMsg := ethereum.CallMsg{
		From:      fromSignerAddr,
		To:        &t.address,
		GasFeeCap: feeCap,
		GasTipCap: tipCap,
		Data:      data,
	}
	gasLimit, err := t.client.EstimateGas(ctx, callMsg)
	if err != nil {
		return "", fmt.Errorf("estimate gas: %w", err)
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: tipCap,
		GasFeeCap: feeCap,
		Gas:       gasLimit,
		To:        &t.address,
		Data:      data,
	})

	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), signerKey)
	if err != nil {
		return "", fmt.Errorf("sign burn tx: %w", err)
	}

	if err := t.client.SendTransaction(ctx, signedTx); err != nil {
		return "", fmt.Errorf("send burn tx: %w", err)
	}
	receipt, err := bind.WaitMined(ctx, t.client, signedTx)
	if err != nil {
		return "", fmt.Errorf("wait burn receipt: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return "", fmt.Errorf("burn tx failed: %s", signedTx.Hash().Hex())
	}

	return signedTx.Hash().Hex(), nil
}
```

- [ ] **Step 2：確認編譯通過**

```bash
cd backend && go build ./internal/contract/...
```

期望：無錯誤輸出

- [ ] **Step 3：Commit**

```bash
git add backend/internal/contract/tachi_token.go
git commit -m "feat: add TachiToken.Burn() method

refs #186

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 2：寫 `spend_service_test.go`（TDD 先寫測試）

**Files:**
- Create: `backend/internal/services/spend_service_test.go`

- [ ] **Step 1：建立測試檔案**

建立 `backend/internal/services/spend_service_test.go`：

```go
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
```

- [ ] **Step 2：確認測試失敗（SpendService 尚未存在）**

```bash
cd backend && go test ./internal/services/ -run "TestRedeem" -v 2>&1 | head -20
```

期望：編譯錯誤 `undefined: SpendService` 或類似

- [ ] **Step 3：Commit 測試檔**

```bash
git add backend/internal/services/spend_service_test.go
git commit -m "test: add SpendService.Redeem unit tests (failing)

refs #186

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 3：實作 `spend_service.go`

**Files:**
- Create: `backend/internal/services/spend_service.go`

- [ ] **Step 1：建立 `spend_service.go`**

建立 `backend/internal/services/spend_service.go`：

```go
package services

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/tachigo/tachigo/internal/config"
	contractpkg "github.com/tachigo/tachigo/internal/contract"
	"github.com/tachigo/tachigo/internal/models"
)

var (
	ErrSpendAmountInvalid       = errors.New("spend amount must be greater than zero")
	ErrSpendInsufficientBalance = errors.New("insufficient tachi balance")
	ErrSpendWalletNotLinked     = errors.New("web3 wallet not linked")
	ErrSpendContractConfig      = errors.New("spend contract config is incomplete")
)

// BurnCaller abstracts the on-chain burn call; replaced with a mock in tests.
type BurnCaller interface {
	BurnOnChain(ctx context.Context, fromAddr string, amount int64) (txHash string, err error)
}

type SpendService struct {
	db          *gorm.DB
	contractCfg config.ContractConfig
	tachiToken  *contractpkg.TachiToken
	burnCaller  BurnCaller
}

type spendReservation struct {
	fromAddr   string
	amount     int64
	newBalance int64
}

func NewSpendService(db *gorm.DB, contractCfg config.ContractConfig, ethClient *ethclient.Client) *SpendService {
	svc := &SpendService{
		db:          db,
		contractCfg: contractCfg,
	}
	if ethClient != nil && contractCfg.TachiContractAddress != "" && contractCfg.SepoliaSignerKey != "" {
		if common.IsHexAddress(contractCfg.TachiContractAddress) {
			t, err := contractpkg.NewTachiToken(common.HexToAddress(contractCfg.TachiContractAddress), ethClient)
			if err == nil {
				svc.tachiToken = t
			}
		}
	}
	svc.burnCaller = svc
	return svc
}

// SetBurnCallerForTest replaces the burn caller; use only in tests.
func (s *SpendService) SetBurnCallerForTest(bc BurnCaller) { s.burnCaller = bc }

// Redeem burns `amount` $TACHI from the user's on-chain wallet and deducts
// the same amount from tachi_balances. Returns the new balance.
func (s *SpendService) Redeem(ctx context.Context, userID uuid.UUID, amount int64) (int64, error) {
	if amount <= 0 {
		return 0, ErrSpendAmountInvalid
	}

	var reservation spendReservation
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		reservation, err = s.reserveSpend(tx, userID, amount)
		return err
	}); err != nil {
		return 0, err
	}

	burnCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if _, err := s.burnCaller.BurnOnChain(burnCtx, reservation.fromAddr, reservation.amount); err != nil {
		rollbackErr := s.db.Transaction(func(tx *gorm.DB) error {
			return s.rollbackSpendReservation(tx, userID, reservation.amount)
		})
		if rollbackErr != nil {
			return 0, fmt.Errorf("%w; rollback spend reservation: %v", err, rollbackErr)
		}
		return 0, err
	}

	return reservation.newBalance, nil
}

func (s *SpendService) reserveSpend(tx *gorm.DB, userID uuid.UUID, amount int64) (spendReservation, error) {
	var tb models.TachiBalance
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&tb).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return spendReservation{}, ErrSpendInsufficientBalance
		}
		return spendReservation{}, err
	}
	if tb.Balance < amount {
		return spendReservation{}, ErrSpendInsufficientBalance
	}

	fromAddr, err := s.resolveWalletAddress(tx, userID)
	if err != nil {
		return spendReservation{}, err
	}

	newBalance := tb.Balance - amount
	if err := tx.Model(&models.TachiBalance{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"balance":    newBalance,
			"updated_at": time.Now(),
		}).Error; err != nil {
		return spendReservation{}, err
	}

	return spendReservation{
		fromAddr:   fromAddr,
		amount:     amount,
		newBalance: newBalance,
	}, nil
}

func (s *SpendService) rollbackSpendReservation(tx *gorm.DB, userID uuid.UUID, amount int64) error {
	return tx.Model(&models.TachiBalance{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"balance":    gorm.Expr("balance + ?", amount),
			"updated_at": time.Now(),
		}).Error
}

func (s *SpendService) resolveWalletAddress(db *gorm.DB, userID uuid.UUID) (string, error) {
	var authProvider models.AuthProvider
	err := db.Where("user_id = ? AND provider = ?", userID, models.ProviderWeb3).
		Order("created_at ASC").
		First(&authProvider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrSpendWalletNotLinked
		}
		return "", err
	}
	if !common.IsHexAddress(authProvider.ProviderID) {
		return "", fmt.Errorf("invalid linked wallet address: %s", authProvider.ProviderID)
	}
	return common.HexToAddress(authProvider.ProviderID).Hex(), nil
}

// BurnOnChain implements BurnCaller using the real TachiToken contract.
func (s *SpendService) BurnOnChain(ctx context.Context, fromAddr string, amount int64) (string, error) {
	if s.tachiToken == nil {
		return "", ErrSpendContractConfig
	}
	if !common.IsHexAddress(fromAddr) {
		return "", fmt.Errorf("invalid wallet address: %s", fromAddr)
	}
	if amount <= 0 {
		return "", ErrSpendAmountInvalid
	}

	signerKey, err := parseSignerKey(s.contractCfg.SepoliaSignerKey)
	if err != nil {
		return "", err
	}

	return s.tachiToken.Burn(ctx, common.HexToAddress(fromAddr), big.NewInt(amount), signerKey)
}
```

- [ ] **Step 2：執行測試，確認全部通過**

```bash
cd backend && go test ./internal/services/ -run "TestRedeem" -v
```

期望輸出：

```text
--- PASS: TestRedeem_Success (0.00s)
--- PASS: TestRedeem_InsufficientBalance (0.00s)
--- PASS: TestRedeem_WalletNotLinked (0.00s)
--- PASS: TestRedeem_BurnFailureRollback (0.00s)
PASS
```

- [ ] **Step 3：執行完整測試，確認無 regression**

```bash
cd backend && go test ./...
```

期望：`ok` 或 `no test files` 對每個 package，無 FAIL

- [ ] **Step 4：Commit**

```bash
git add backend/internal/services/spend_service.go
git commit -m "feat: implement SpendService.Redeem with reserve-then-burn flow

refs #186

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 4：實作 `SpendHandler`

**Files:**
- Create: `backend/internal/handlers/spend_handler.go`

- [ ] **Step 1：建立 `spend_handler.go`**

建立 `backend/internal/handlers/spend_handler.go`：

```go
package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tachigo/tachigo/internal/middleware"
	"github.com/tachigo/tachigo/internal/services"
)

type SpendHandler struct {
	spendSvc *services.SpendService
}

func NewSpendHandler(spendSvc *services.SpendService) *SpendHandler {
	return &SpendHandler{spendSvc: spendSvc}
}

type redeemRequest struct {
	Amount int64 `json:"amount"`
}

type redeemResponse struct {
	Balance int64 `json:"balance"`
}

// Redeem godoc
// @Summary      Redeem $TACHI for a discount coupon
// @Tags         spend
// @Accept       json
// @Produce      json
// @Param        body body redeemRequest true "Amount to burn"
// @Success      200 {object} Response{data=redeemResponse}
// @Failure      400 {object} Response
// @Failure      401 {object} Response
// @Failure      500 {object} Response
// @Security     BearerAuth
// @Router       /spend/redeem [post]
func (h *SpendHandler) Redeem(c *gin.Context) {
	claims := middleware.MustClaims(c)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		badRequest(c, "invalid user id")
		return
	}

	var req redeemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid request body: "+err.Error())
		return
	}
	if req.Amount <= 0 {
		badRequest(c, "amount must be > 0")
		return
	}

	newBalance, err := h.spendSvc.Redeem(c.Request.Context(), userID, req.Amount)
	if err != nil {
		if errors.Is(err, services.ErrSpendInsufficientBalance) {
			badRequest(c, err.Error())
			return
		}
		if errors.Is(err, services.ErrSpendWalletNotLinked) {
			badRequest(c, err.Error())
			return
		}
		if errors.Is(err, services.ErrSpendAmountInvalid) {
			badRequest(c, err.Error())
			return
		}
		internal(c)
		return
	}

	ok(c, redeemResponse{Balance: newBalance})
}
```

- [ ] **Step 2：確認編譯**

```bash
cd backend && go build ./internal/handlers/...
```

期望：無錯誤

- [ ] **Step 3：Commit**

```bash
git add backend/internal/handlers/spend_handler.go
git commit -m "feat: add SpendHandler POST /spend/redeem

refs #186

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 5：Wire router 與 main

**Files:**
- Modify: `backend/internal/router/router.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1：更新 `router.go`**

在 `router.go` 的 `New()` 函式簽名加入 `spendSvc *services.SpendService` 參數（放在 `claimSvc` 後面）：

```go
func New(
	authSvc *services.AuthService,
	userSvc *services.UserService,
	addrSvc *services.AddressService,
	extSvc *services.ExtensionService,
	emailAuthSvc *services.EmailAuthService,
	watchSvc *services.WatchService,
	channelConfigSvc *services.ChannelConfigService,
	pointsSvc *services.PointsService,
	airdropSvc *services.AirdropService,
	streamerSvc *services.StreamerService,
	agencySvc *services.AgencyService,
	claimSvc *services.ClaimService,
	spendSvc *services.SpendService,
	agencyHandler *handlers.AgencyHandler,
	allowedOrigins []string,
	internalRouterConfig ...InternalRouterConfig,
) *gin.Engine {
```

在函式內，`claimH` 初始化後加入：

```go
spendH := handlers.NewSpendHandler(spendSvc)
```

在 `protected` group 內加入新路由（放在 claim 路由附近）：

```go
// $TACHI spend (burn)
protected.POST("spend/redeem", spendH.Redeem)
```

- [ ] **Step 2：更新 `main.go`**

在 `claimSvc := services.NewClaimService(...)` 後加入：

```go
spendSvc := services.NewSpendService(db, cfg.Contract, ethClient)
```

在 `router.New(...)` 呼叫中，`claimSvc` 後加入 `spendSvc`：

```go
r := router.New(
	authSvc,
	userSvc,
	addrSvc,
	extSvc,
	emailAuthSvc,
	watchSvc,
	channelConfigSvc,
	pointsSvc,
	airdropSvc,
	streamerSvc,
	agencySvc,
	claimSvc,
	spendSvc,
	agencyH,
	allowedOrigins,
	router.InternalRouterConfig{DB: db, Config: cfg},
)
```

- [ ] **Step 3：確認 router_test.go 編譯（需要更新 test helper）**

查看 `backend/internal/router/router_test.go` 的 `New()` 呼叫，加入 `nil`（spendSvc）佔位：

```bash
cd backend && grep -n "router.New\|New(" internal/router/router_test.go | head -10
```

若測試檔案中有呼叫 `router.New()`，對應位置加入 `nil` 作為 `spendSvc`。

- [ ] **Step 4：完整 build + test**

```bash
cd backend && go build ./...
```

期望：無錯誤

```bash
cd backend && go test ./...
```

期望：所有 package PASS，無 FAIL

- [ ] **Step 5：Commit**

```bash
git add backend/internal/router/router.go backend/cmd/server/main.go
git commit -m "feat: wire SpendService and POST /api/v1/spend/redeem route

closes #186

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 6：最終驗證與 push

- [ ] **Step 1：最終驗證**

```bash
cd backend && go build ./... && go test ./...
```

期望：`go build ./...` 無錯誤，`go test ./...` 全部 PASS

- [ ] **Step 2：Push**

```bash
git push -u origin feat/spend-service
```
