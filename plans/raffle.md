# 抽獎系統 — 核心資料模型與 API

狀態：已完成

refs #227

## 背景

實況主需要在 Dashboard 建立抽獎活動、匯入訂閱名單、直播中逐一抽獎，中獎者透過連結填寫收件資訊。

## 架構決策

### 資料模型

- **Raffle**：一場抽獎活動（owner: streamer user_id）
- **RaffleEntry**：名單條目（可對應有帳號 user 或僅有 twitch_login）
- **RaffleDraw**：每次抽出的結果，含一次性 claim_token（UUID）與過期時間（7 天）
- **RaffleClaim**：中獎者填寫的收件資訊（1:1 with RaffleDraw）

### 所有權模型

Raffle.UserID = JWT claims.UserID（RoleStreamer）  
endpoint 驗證：若 raffle.UserID ≠ JWT user，回 403。

### CSV 匯入邏輯

- CSV 第一欄為 twitch_login
- 比對 auth_providers（provider='twitch', provider_id=twitch_login）
- 有帳號 → 設 user_id 並匯入；無帳號 → 跳過（計入 skipped）
- 跳過空行與 header
- (raffle_id, twitch_login) 唯一約束防重複匯入

### DrawNext 邏輯

- 從 raffle_entries 中隨機取一筆「尚未出現在 raffle_draws 中的 entry」
- 建立 RaffleDraw，claim_token = uuid v7，expires_at = 現在 + 7 天
- 若所有 entry 都已抽過 → ErrRaffleExhausted

### ClaimByToken

- 找到 draw by token
- 若 claim_expires_at < now → 410 Gone
- 回傳 draw + entry 資訊

### SubmitClaim

- 同上驗證
- draw_id uniqueIndex 防重複提交

## 檔案清單

| 檔案 | 說明 |
|---|---|
| `backend/internal/models/raffle.go` | 四個 model |
| `backend/internal/services/raffle_service.go` | RaffleService |
| `backend/internal/handlers/raffle_handler.go` | RaffleHandler |
| `backend/internal/router/router.go` | 新增 raffle routes |
| `backend/cmd/server/main.go` | AutoMigrate + wire |
| `backend/internal/handlers/testutil_test.go` | 補 SQLite DDL |
| `backend/internal/handlers/raffle_handler_test.go` | handler 測試 |

## 實作 Checklist

- [x] `backend/internal/models/raffle.go` — Raffle、RaffleEntry、RaffleDraw、RaffleClaim 四個模型，含 uniqueIndex 約束
- [x] `backend/internal/services/raffle_service.go` — 全部 8 個 Service 方法，DrawNext 使用 transaction
- [x] `backend/internal/handlers/raffle_handler.go` — 10 個 HTTP 端點（Dashboard 7 + 公開 2 + Extension 1）
- [x] `backend/internal/router/router.go` — routes 掛載
- [x] `backend/cmd/server/main.go` — AutoMigrate + service wire
- [x] `backend/internal/handlers/testutil_test.go` — SQLite DDL（含 unique 約束）
- [x] `backend/internal/handlers/raffle_handler_test.go` — 10 個 handler 單元測試

## 驗證方式

```bash
docker compose run --no-deps --rm app go test ./...
```

所有測試應全數通過。

## API 路由

### Dashboard（JWT + RoleStreamer）

```
POST   /api/v1/dashboard/raffles
GET    /api/v1/dashboard/raffles
GET    /api/v1/dashboard/raffles/:id
POST   /api/v1/dashboard/raffles/:id/entries/import-csv
POST   /api/v1/dashboard/raffles/:id/draws
GET    /api/v1/dashboard/raffles/:id/draws
POST   /api/v1/dashboard/raffles/:id/complete
```

### 公開（無需登入）

```
GET    /api/v1/claim/:token
POST   /api/v1/claim/:token
```

### Extension（JWT viewer）

```
GET    /api/v1/extension/raffles/:id/result
```
