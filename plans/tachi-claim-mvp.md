# $TACHI Claim MVP 實作計畫

> 狀態：待實作
> 最後更新：2026-04-07
> 相關 Issue：待開（見下方）
> 依賴：#59 #64（watch-to-points，已完成）

---

## 背景

watch-to-points 流程（#59/#64）已完成，用戶可透過觀看直播累積 `points_ledgers.spendable_balance`。

目前缺少的是「T-Point → $TACHI」的兌換出口：用戶沒有辦法把累積的 T-Point 換成 $TACHI，整條 tokenomics 流程因此斷在這裡。

本計畫實作 **Phase 1 DB-only Claim**，不觸及鏈上 mint，為 Phase 2 Router Contract 上鏈預留介面。

---

## 架構決策

### Phase 1 簡化方案（本計畫範圍）

```
用戶 POST /users/me/points/claim
  → 驗證 spendable_balance 足夠
  → DB transaction：
      1. points_ledgers.spendable_balance -= amount（跨所有頻道加總）
      2. points_transactions 寫一筆 source='claim', delta=-amount
      3. tachi_balances.balance += amount（新表）
  → 回傳新的 tachi_balances.balance
```

### Phase 2 升級路徑（本計畫不實作，留介面）

- `tachi_balances` 將作為鏈上餘額的 DB 鏡像
- Claim 改為呼叫 Soulbound ERC-20 合約 mint，並扣除 1% 手續費 Burn
- `ClaimService.Claim()` 介面設計需支援未來傳入 `wallet_address` 參數

---

## 資料層

### 新增 migration：`tachi_balances`

```sql
CREATE TABLE tachi_balances (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    balance    NUMERIC(20, 6) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT tachi_balances_user_id_key UNIQUE (user_id),
    CONSTRAINT tachi_balances_balance_non_negative CHECK (balance >= 0)
);
```

### `points_transactions.source` 新增枚舉值

現有 source 值需確認後補入 `'claim'`（若為 string 欄位直接使用即可；若為 enum 需補 migration）。

---

## 介面規格

### ClaimService

```go
type ClaimService interface {
    // Claim 將指定用戶所有頻道的 spendable_balance 加總後轉換為 tachi_balances
    // amount <= 0 表示 claim 全部
    // Phase 2：新增 walletAddress string 參數用於 on-chain mint
    Claim(ctx context.Context, userID uuid.UUID, amount decimal.Decimal) (newBalance decimal.Decimal, err error)

    // GetTachiBalance 查詢用戶目前 $TACHI 餘額
    GetTachiBalance(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error)
}
```

### API

```
POST /api/v1/users/me/points/claim
Authorization: Bearer <tachigo JWT>

Request body（可選，省略則 claim 全部）：
{
  "amount": "100.000000"   // 要 claim 的 T-Point 數量
}

Response 200：
{
  "tachi_balance": "100.000000"
}

Error cases：
- 400：amount <= 0 或格式錯誤
- 422：spendable_balance 不足
- 401：未登入
```

```
GET /api/v1/users/me/tachi/balance
Authorization: Bearer <tachigo JWT>

Response 200：
{
  "tachi_balance": "100.000000"
}
```

---

## 實作 Checklist

### 資料層
- [ ] 新增 `backend/migrations/XXX_tachi_balances.sql`
- [ ] 新增 `backend/internal/models/tachi_balance.go`（`TachiBalance` struct + AutoMigrate 註冊）
- [ ] 確認 `points_transactions.source` 支援 `'claim'` 值

### Service 層
- [ ] 新增 `backend/internal/services/claim_service.go`
  - [ ] `Claim()`：DB transaction，跨頻道加總 spendable_balance → tachi_balances
  - [ ] `GetTachiBalance()`：查詢 tachi_balances

### Handler 層
- [ ] 新增 `backend/internal/handlers/claim_handler.go`
  - [ ] `ClaimHandler.Claim()` → `POST /users/me/points/claim`
  - [ ] `ClaimHandler.GetTachiBalance()` → `GET /users/me/tachi/balance`
- [ ] 在 `router.go` 接線（需 JWTAuth）

### 測試
- [ ] `ClaimService` 單元測試（含 spendable_balance 不足、跨頻道加總、DB transaction rollback）
- [ ] Handler 整合測試

### 文件
- [ ] 更新 `docs/tokenomics.md` Phase 1 功能表的狀態（🟡 → ✅）
- [ ] Swagger 補 claim / tachi balance 兩支 API

---

## 驗證方式

- `go build ./...` 通過
- `go test ./...` 通過
- 手動驗證：觀看累積 T-Point → Claim → `GET /users/me/tachi/balance` 回傳正確餘額
- spendable_balance 扣除後不可為負數（CHECK constraint 保護）

---

## 不在本計畫範圍

| 功能 | 原因 |
|------|------|
| 1% Claim 手續費 Burn | Phase 2，需鏈上 mint 配合 |
| 7 日未 Claim 自動回收財庫 | Phase 2，需 Cron Job |
| Soulbound ERC-20 on-chain mint | Phase 2，需 Foundry 合約部署 |
| 跨頻道點數個別 Claim | 設計決策：Phase 1 全頻道加總一次兌換，簡化流程 |
