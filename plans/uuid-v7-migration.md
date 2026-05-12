# UUID v7 Migration

> **狀態：** 部分完成，剩餘正式環境 hooks 待遷移
> **關聯 Issue：** refs #61, refs #354
> **最後更新：** 2026-05-13

---

## 背景

UUID v7 的技術原理與策略維護在 [docs/uuid-v7.md](../docs/uuid-v7.md)。本文件只追蹤實作狀態，避免與策略文件重複。

目前 `services/api` 已有部分 model 與 service 建立資料列的路徑改用 `uuid.NewV7()`，但仍有正式環境 `BeforeCreate` hook 使用 `uuid.New()`。本 plan 因此保留在 active `plans/`，暫不歸檔。

---

## 已完成項目

### 已使用 UUID v7 的 model BeforeCreate hooks

- [x] `services/api/internal/models/claim.go`（Claim、ClaimItem）
- [x] `services/api/internal/models/coupon_redemption.go`
- [x] `services/api/internal/models/points.go`（PointsLedger、PointsTransaction）
- [x] `services/api/internal/models/raffle.go`（Raffle、RaffleEntry、RaffleDraw、RaffleClaim）
- [x] `services/api/internal/models/streamer.go`
- [x] `services/api/internal/models/watch_session.go`
- [x] `services/api/internal/models/watch_stats.go`（WatchTimeStat、BroadcastTimeStat、BroadcastTimeLog）

### 已使用 UUID v7 的 service 正式資料列 ID 建立路徑

- [x] `services/api/internal/services/watch_service.go` 使用 `newUUID()` 建立 watch session 與 ledger/transaction IDs。
- [x] `services/api/internal/services/points_service.go` 使用 `newUUID()` 處理直接 SQL insert。

---

## 待實作項目

### 仍使用 UUID v4 的 model BeforeCreate hooks

- [ ] `services/api/internal/models/user.go`
- [ ] `services/api/internal/models/auth_provider.go`
- [ ] `services/api/internal/models/address.go`
- [ ] `services/api/internal/models/refresh_token.go`（RefreshToken、Web3Nonce）
- [ ] `services/api/internal/models/email_auth.go`（EmailVerification、PasswordReset）

### Service 層直接賦值

- [x] `services/api/internal/services` 底下已無非測試用的 `ID: uuid.New()` 直接賦值。

### 不需更動

- 測試檔案中的 `uuid.New()`：測試 ID 無須時序性。
- `uuid.New()` 作為 `uuid.NewV7()` 發生錯誤後的 fallback。

---

## 驗證方式

```bash
rtk rg -n "ID:\\s*(uuid.New|newUUID)|uuid.NewV7|uuid.New\\(\\)" services/api/internal/services -g '!**/*_test.go'
rtk rg -n "func .*BeforeCreate|uuid.NewV7|uuid.New\\(\\)" services/api/internal/models -g '!**/*_test.go'
rtk docker compose run --no-deps --rm app go test ./...
```

後續實作 PR 應補 model-level tests，確認新建 production rows 的 ID 使用 UUID v7（`id.Version() == 7`）。
