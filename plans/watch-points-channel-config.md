# Watch-to-Points 補強 + Channel Config

> **狀態：** 待實作
> **關聯 Issue：** refs #59
> **最後更新：** 2026-04-01

---

## 背景

本次變更涵蓋四個目標：

1. **points_ledgers 改為 per-channel 帳本** — 原設計為全平台共用，與「頻道點數獨立」需求不符
2. **Session 結束機制** — server 無法主動偵測 client 斷線，採兩層機制：client best-effort `EndSession` + server stale session 偵測
3. **Heartbeat 補上 `<20s ignore` 防作弊規則** — 原實作只有 cap 30s，缺少快速重送的擋掉機制
4. **Channel Config 可調發點速率** — 讓實況主 / 經紀公司可在 Dashboard 動態調整 `seconds_per_point`（例如工商時段加速發點）

設計細節見 [docs/watch-to-points-design.md](../docs/watch-to-points-design.md)。

---

## 架構決策

| 項目 | 決策 | 理由 |
|---|---|---|
| Session 結束偵測 | 兩層：client best-effort `EndSession` + server stale 偵測 | `beforeunload` 不保證送出，server 需保底 |
| `staleThreshold` | 2 分鐘 | Twitch 網路不穩，4 次錯過才斷線，避免誤判 |
| Stale 清理時機 | 下次 `StartSession` 時觸發，不做 cron | MVP 降低複雜度 |
| `maxDeltaPerHeartbeat` | 30 秒 | 斷線後重連不補算過多 |
| `<20s ignore` | 直接 return，不更新 `last_heartbeat_at` | 正常 30s 間隔不觸發，只擋異常重送 |
| `seconds_per_point` 每次查 DB | 是，不加 cache | PK lookup 夠快，MVP 不過度設計 |
| Streamer 只能改自己頻道？ | 否，MVP 依角色授權 | 降低複雜度 |

---

## 待實作項目

### 1. Session 結束機制

- [ ] `backend/internal/services/watch_service.go`
  - `StartSession()`：偵測 stale session（`last_heartbeat_at > staleThreshold(2 分鐘)`）→ `is_active=false, ended_at=now()`，再建新 session
  - `EndSession()`：best-effort，冪等（無 active session 視為成功，不回傳錯誤）

### 2. LoginWithExtension 改為 find-only

- [ ] `backend/internal/services/extension_service.go` — 移除 find-or-create 分支，找不到時回傳 `ErrUserNotFound`
- [ ] `backend/internal/handlers/extension_handler.go` — 判斷 `ErrUserNotFound` → 401，提示先至 tachigo 登入並連結 Twitch

### 2. Schema — points_ledgers 改為 per-channel

- [ ] `backend/migrations/003_watch_points.sql` — `points_ledgers` 加入 `channel_id`，unique 改為 `(twitch_user_id, channel_id)`
- [ ] `backend/internal/models/points.go` — `PointsLedger` struct 加入 `ChannelID`，移除單欄 `UNIQUE` tag

### 3. Service / Handler — per-channel 帳本 + `<20s ignore` + `seconds_per_point`

- [ ] `backend/internal/services/watch_service.go`
  - `Heartbeat()`：SELECT FOR UPDATE 後加 `< 20s → return` 早退
  - `Heartbeat()`：SQL upsert 帶入 `channel_id`
  - `Heartbeat()`：以 `getSecondsPerPoint()` 取代硬編碼 `60`
  - `GetBalance()`：加入 `channelID` 參數
  - 新增 `getSecondsPerPoint(db, channelID) int64` helper
- [ ] `backend/internal/handlers/watch_handler.go` — `GetBalance` 從 Extension JWT claims 取 `ChannelID` 傳入 service

### 4. Migration — channel_configs

- [ ] `backend/migrations/004_channel_config.sql` — 建立 `channel_configs` 表
- [ ] `backend/internal/models/channel_config.go` — `ChannelConfig` struct

### 5. Dashboard API

- [ ] `backend/internal/middleware/auth.go` — 新增 `RequireRole(roles ...UserRole)`
- [ ] `backend/internal/handlers/channel_config_handler.go` — `UpdateChannelConfig` handler
- [ ] `backend/internal/router/router.go` — 新增 `/dashboard/` route group（`JWTAuth` + `RequireRole(Admin, Streamer)`）

### 6. Wiring

- [ ] `backend/cmd/server/main.go`
  - `AutoMigrate` 加入 `&models.ChannelConfig{}`
  - 手動建 `idx_points_ledgers_user_channel` unique index
  - 初始化 `ChannelConfigHandler`，傳入 `router.New()`

---

## 驗證方式

```bash
docker compose run --no-deps --rm app go test ./...
```

手動流程：

1. 呼叫 `start` 建立 session → 停止送 heartbeat → 等 2 分鐘 → 再次呼叫 `start` → 確認舊 session 被關閉（`is_active=false`），新 session 建立
2. 呼叫 `start` → 呼叫 `end` → 確認 session `is_active=false`；再呼叫一次 `end` → 確認回傳成功（冪等）
3. 以 Streamer/Admin 帳號登入取得 JWT
2. `PUT /api/v1/dashboard/channels/<channel_id>/config` body `{"seconds_per_point": 10}`
3. 用 Extension JWT 觸發 heartbeat → 確認 30s 內發 3 點
4. 連送兩次 heartbeat（間隔 <20s）→ 確認第二次 `points_earned: 0`
5. `GET /extension/watch/balance` → 確認只回傳該頻道餘額
6. 改回 `seconds_per_point: 60` → 確認回到正常發點
