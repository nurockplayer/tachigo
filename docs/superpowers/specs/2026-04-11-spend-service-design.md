# SpendService 設計文件

**Issue**: #186  
**日期**: 2026-04-11  
**狀態**: 待實作

---

## 背景

用戶花費 `$TACHI`（`tachi_balances`）換折價券的消費路徑。呼叫合約 `TachiToken.burn(address, amount)` 銷毀鏈上代幣，再扣除 DB 中的 `tachi_balances.balance`。

依賴：
- #166 合約 `burn()` ABI 已就位
- #174 合約部署到 Sepolia
- #178 後端 env var 已設定

---

## 架構

### 檔案異動

| 檔案 | 操作 |
|---|---|
| `backend/internal/contract/tachi_token.go` | 新增 `Burn()` 方法 |
| `backend/internal/services/spend_service.go` | 新檔：SpendService |
| `backend/internal/handlers/spend_handler.go` | 新檔：SpendHandler |
| `backend/internal/router/router.go` | 注入 SpendService / SpendHandler，掛 route |
| `backend/cmd/api/main.go` | wire SpendService |
| `backend/internal/services/spend_service_test.go` | 新檔：單元測試 |

---

## 介面規格

### BurnCaller interface

```go
type BurnCaller interface {
    BurnOnChain(ctx context.Context, fromAddr string, amount int64) (txHash string, err error)
}
```

### SpendService

```go
type SpendService struct {
    db          *gorm.DB
    contractCfg config.ContractConfig
    tachiToken  *contractpkg.TachiToken
    burnCaller  BurnCaller
}

func NewSpendService(db *gorm.DB, contractCfg config.ContractConfig, ethClient *ethclient.Client) *SpendService

func (s *SpendService) Redeem(ctx context.Context, userID uuid.UUID, amount int64) (newBalance int64, err error)

// SetBurnCallerForTest replaces the burn caller; use only in tests.
func (s *SpendService) SetBurnCallerForTest(bc BurnCaller)
```

### TachiToken.Burn（新增）

```go
func (t *TachiToken) Burn(ctx context.Context, fromAddr common.Address, amount *big.Int, signerKey *ecdsa.PrivateKey) (string, error)
```

---

## API

```
POST /api/v1/spend/redeem
Authorization: Bearer <jwt>
Content-Type: application/json

Body:  { "amount": 100 }        // 必填，> 0
200:   { "balance": 900 }
400:   餘額不足 / 錢包未綁定 / amount <= 0
500:   合約呼叫失敗
```

---

## Transaction 流程（方案 A：reserve-then-burn）

```
1. DB txn (SELECT FOR UPDATE):
   a. 取得 tachi_balances WHERE user_id（lock row）
      → 不存在或 balance < amount → ErrSpendInsufficientBalance (400)
   b. resolveWalletAddress(auth_providers, provider=web3)
      → 找不到 → ErrSpendWalletNotLinked (400)
   c. UPDATE tachi_balances SET balance = balance - amount（reservation）

2. BurnOnChain(walletAddr, amount) — 30s timeout
   → 失敗 → rollback DB（UPDATE balance = balance + amount）→ 500

3. 回傳 newBalance（= reservation 後的值）
```

**關鍵設計決策**：wallet 解析在 DB txn 內執行。若 wallet 找不到，txn 直接 rollback，不需要額外還原 balance。這與 `ClaimService.reserveClaim()` 的模式一致。

---

## 錯誤定義

```go
var (
    ErrSpendAmountInvalid       = errors.New("spend amount must be greater than zero")
    ErrSpendInsufficientBalance = errors.New("insufficient tachi balance")
    ErrSpendWalletNotLinked     = errors.New("web3 wallet not linked")
    ErrSpendContractConfig      = errors.New("spend contract config is incomplete")
)
```

---

## 測試計畫

| 測試名稱 | 情境 | 驗證點 |
|---|---|---|
| `TestRedeem_Success` | 正常流程 | newBalance 正確、BurnCaller 被呼叫 1 次、fromAddr 正確 |
| `TestRedeem_InsufficientBalance` | tachi_balances.balance < amount | 回傳 ErrSpendInsufficientBalance，balance 不變 |
| `TestRedeem_WalletNotLinked` | 無 web3 auth_provider | 回傳 ErrSpendWalletNotLinked，balance 不變 |
| `TestRedeem_BurnFailureRollback` | BurnCaller 回傳 error | balance 還原為扣除前的值 |

使用 SQLite in-memory DB + mock BurnCaller，與 `claim_service_test.go` 相同模式。

---

## 本票明確不做

- 不實作折價券系統本身（coupon 發放、驗證邏輯）
- 不動 ClaimService 或既有記帳架構
- 不做 SIWE 錢包綁定
- 不部署到 mainnet
- 不引入 Gas 補貼機制
