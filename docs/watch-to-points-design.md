# Watch-to-Points 設計文件

> **狀態：** 實作中（refs #59）
> **最後更新：** 2026-04-01

---

## 概述

觀眾在 Twitch Extension 觀看直播時定時發送 heartbeat（每 30 秒），後端累積觀看秒數並依頻道設定的速率發點。點數記錄在 **per-channel 帳本**（`points_ledgers`），每位觀眾在每個頻道各有獨立餘額，頻道點數彼此不互通。兌換時才將頻道點數轉換為統一平台幣並上鏈 mint（Phase 2）。

---

## 認證流程

```
前置條件：觀眾必須先在 tachigo 完成登入並授權連結 Twitch 帳號

觀眾開啟 Extension
  → 前端呼叫 POST /api/v1/extension/auth/login（帶 Extension JWT）
  → 後端 LoginWithExtension：
      以 Extension JWT 中的 twitch_user_id 查詢已連結的 tachigo 帳號
      找到 → 回傳 tachigo JWT
      找不到 → 401，提示觀眾先至 tachigo 登入並授權 Twitch
  → 前端存下 tachigo JWT

觀眾開始觀看
  → 前端呼叫 POST /api/v1/extension/watch/start（帶 tachigo JWT）
  → 定時呼叫 POST /api/v1/extension/watch/heartbeat（每 30 秒）
  → 離開時盡力呼叫 POST /api/v1/extension/watch/end（見「Session 結束機制」）
```

**為什麼必須先登入 tachigo 再授權 Twitch：**

- 帳號主體是 tachigo 使用者，Twitch 是附掛的 auth provider
- 確保點數帳本有明確的 `user_id` 歸屬，Phase 2 claim 上鏈不需額外橋接
- 避免匿名觀眾累積點數後無法認領的問題

---

## 資料模型

### 三張表的職責

```
watch_sessions ──heartbeat──▶ points_transactions ◀── points_ledgers
                                                           ▲
                                                      atomic upsert
```

| 表 | 職責 |
|---|---|
| `watch_sessions` | 記錄「現在誰在看哪個頻道」，是暫時的觀看狀態 |
| `points_ledgers` | 每位觀眾在每個頻道的點數帳戶，存當下餘額 |
| `points_transactions` | 不可修改的流水帳，每次發點都記一筆 |

### watch_sessions

| 欄位 | 型別 | 說明 |
|---|---|---|
| `id` | UUID PK | |
| `user_id` | UUID NOT NULL FK → users.id | 觀眾的 tachigo 帳號 ID |
| `channel_id` | VARCHAR(255) NOT NULL | 頻道 ID |
| `accumulated_seconds` | BIGINT default 0 | 本 session 累積觀看秒數 |
| `rewarded_seconds` | BIGINT default 0 | 已換算為點數的秒數（防止重複發） |
| `last_heartbeat_at` | TIMESTAMPTZ NOT NULL default now() | 最後 heartbeat 時間 |
| `is_active` | BOOLEAN NOT NULL default true | 是否為進行中 session |
| `ended_at` | TIMESTAMPTZ NULL | session 結束時間 |

**Partial unique index**（GORM 不支援，在 `main.go` 手動建）：

```sql
CREATE UNIQUE INDEX idx_watch_sessions_active_user_channel
  ON watch_sessions (user_id, channel_id)
  WHERE is_active = TRUE;
```

同一個觀眾在同一個頻道只能有一個 active session，歷史 session（`is_active = false`）不受限，保留查詢用。

**Session lifecycle：**

```
active  : is_active = true,  ended_at = NULL
finished: is_active = false, ended_at = <timestamp>
```

### points_ledgers

**每位觀眾 × 每個頻道各有一本獨立帳本。** 頻道點數彼此不互通；Phase 2 兌換時才將頻道點數轉換為統一平台幣並上鏈 mint。

| 欄位 | 型別 | 說明 |
|---|---|---|
| `id` | UUID PK | |
| `user_id` | UUID NOT NULL FK → users.id | 觀眾的 tachigo 帳號 ID |
| `channel_id` | VARCHAR(255) NOT NULL | 頻道識別碼 |
| `cumulative_total` | BIGINT default 0 | 歷史累積點數（只增不減，用於成就、統計） |
| `spendable_balance` | BIGINT default 0 | 可消費點數餘額 |

**Unique index：** `(user_id, channel_id)`

`spendable_balance` 與 `cumulative_total` 會分叉：花掉點數後 `spendable_balance` 下降，`cumulative_total` 不變。

### points_transactions

| 欄位 | 型別 | 說明 |
|---|---|---|
| `id` | UUID PK | |
| `ledger_id` | UUID FK → points_ledgers.id | |
| `watch_session_id` | UUID NULL | `watch_time` 來源必填，`bits` / `spend` 為 NULL |
| `source` | VARCHAR(50) | `watch_time` / `bits` / `spend` |
| `delta` | BIGINT | 變動量（正 = 獲得，負 = 消費） |
| `balance_after` | BIGINT | 交易後餘額快照（查歷史不用重算） |
| `note` | TEXT NULL | 備註 |

**`watch_session_id` 無 FK constraint**：session 可能被清除或封存，不設 FK 避免 transaction 歷史變 orphan。

`source` 與 `watch_session_id` 的規則：

| source | watch_session_id |
|---|---|
| `watch_time` | 一定有值（指向觸發這次發點的 session） |
| `bits` | NULL |
| `spend` | NULL |

---

## API 端點

所有 watch 端點使用標準 tachigo JWT（`Authorization: Bearer <tachigo_jwt>`），由 `JWTAuth` middleware 保護。`user_id` 從 JWT claims 取出；`channel_id` 由前端透過 request body 傳入（tachigo JWT 不含頻道資訊）。

| Method | Path | 說明 | Body / Param |
|---|---|---|---|
| POST | `/api/v1/extension/watch/start` | 開始或取回活躍 session | `{ "channel_id": "..." }` |
| POST | `/api/v1/extension/watch/heartbeat` | 更新計時，達門檻時發點 | `{ "channel_id": "..." }` |
| POST | `/api/v1/extension/watch/end` | 主動結束 session（盡力送出） | `{ "channel_id": "..." }` |
| GET | `/api/v1/extension/watch/balance` | 查詢當前頻道點數餘額 | `?channel_id=...` |

---

## 發點邏輯

```text
每次 heartbeat：
  若 now - last_heartbeat_at < 20s → 忽略（視為重送），直接回傳 points_earned: 0

  secondsPerPoint = channel_configs[channel_id].seconds_per_point  ← 從 DB 讀取，預設 60
  delta = min(now - last_heartbeat_at, 30s)  ← 上限 30 秒，防止長時間斷線後補算過多
  accumulated_seconds += delta
  pending = accumulated_seconds - rewarded_seconds
  points_to_award = pending / secondsPerPoint
  rewarded_seconds += points_to_award * secondsPerPoint

若 points_to_award > 0：
  → Atomic upsert points_ledgers（以 twitch_user_id + channel_id 定位帳本）
  → 寫入 points_transactions（帶 watch_session_id）
```

`rewarded_seconds` 的設計讓「已發點的秒數」與「已累積秒數」分開追蹤，跨多個 heartbeat 也不會漏發或重複發。

**防作弊機制：**

| 規則 | 值 | 目的 |
|---|---|---|
| 最小 heartbeat 間隔 | 20 秒 | 擋掉異常重送，正常 30s 間隔不會觸發 |
| 最大單次 delta | 30 秒 | 斷線後重連不補算過多 |
| `seconds_per_point` 最小值 | 1 | DB constraint，防止除以零或無限發點 |

---

## Session 結束機制

Server **無法主動偵測** client 斷線，因此採用兩層機制：

### 層一：Client 主動呼叫 EndSession（理想路徑）

前端在以下時機呼叫 `POST /extension/watch/end`：

- 觀眾關閉 Extension 面板
- 瀏覽器 `beforeunload` event

**限制：** `beforeunload` 不保證成功送出（例如手機強制關閉、網路中斷），因此這條路徑屬於「盡力送出（best-effort）」。

### 層二：Server 偵測 Stale Session（保底機制）

若某 session 的 `last_heartbeat_at` 超過 `staleThreshold`（2 分鐘）未更新，下次該觀眾在同頻道呼叫 `StartSession` 時，server 會自動：

1. 將舊 session 設為 `is_active = false, ended_at = now()`
2. 建立新 session

**結果：** 即使 client 沒有主動結束，session 最多延遲 2 分鐘才會被關閉。這段時間不會繼續累積秒數（因為沒有 heartbeat 進來）。

### 設計取捨

| 考量 | 決策 |
|---|---|
| staleThreshold 設多少？ | 2 分鐘（heartbeat 每 30 秒，等於允許錯過 4 次）；Twitch 網路不穩，避免誤判 |
| 是否需要主動清理任務（cron）？ | MVP 不做，stale 只在下次 StartSession 時觸發 |
| `ended_at` 精確度 | 非精確值，反映「最後一次 heartbeat 後的關閉時間」 |

---

## 並發安全機制

### Heartbeat — SELECT FOR UPDATE

```go
tx.Clauses(clause.Locking{Strength: "UPDATE"}).
    Where("user_id = ? AND channel_id = ? AND is_active = true", ...).
    First(&session)
```

同一個觀眾的並發 heartbeat 會排隊，不會同時讀到相同狀態並重複發點。

### Heartbeat — Atomic Upsert

```sql
INSERT INTO points_ledgers (id, user_id, channel_id, ...)
VALUES (gen_random_uuid(), ?, ?, ...)
ON CONFLICT (user_id, channel_id) DO UPDATE SET
    spendable_balance = points_ledgers.spendable_balance + EXCLUDED.spendable_balance,
    cumulative_total  = points_ledgers.cumulative_total  + EXCLUDED.cumulative_total,
    updated_at        = NOW()
```

讓 DB 做加法，不在 Go 層 read-modify-write，避免餘額被覆蓋。

### StartSession — Savepoint

`StartSession` 在建立新 session 前先設 savepoint，若 `Create` 因 unique index 衝突失敗（另一個 concurrent request 搶先寫入），回滾到 savepoint 再重新查詢，而不是讓整個 transaction 進入 aborted 狀態（PostgreSQL 的行為）。

| 問題 | 解法 |
|---|---|
| 兩個 heartbeat 同時進來，double-award | `SELECT FOR UPDATE` 鎖住 session 列 |
| 兩個 start 同時進來，違反 partial unique index | Transaction + Savepoint + fallback 查詢 |
| balance 更新衝突 | `INSERT ... ON CONFLICT DO UPDATE` atomic upsert |

---

## Channel Config — 可調發點速率

> **動機：** Demo 與工商時段需要動態調整發點速率，讓經紀公司或實況主可提高觀眾掛台意願。

### 資料模型 — channel_configs

| 欄位 | 型別 | 說明 |
|---|---|---|
| `channel_id` | VARCHAR(255) PK | Twitch 頻道 ID |
| `seconds_per_point` | BIGINT NOT NULL DEFAULT 60 | 幾秒累積 = 1 點（最小值 1） |
| `updated_at` | TIMESTAMPTZ | 最後更新時間 |

無對應此 channel 的設定時，後端 fallback 至預設值 60。

### Dashboard API

| Method | Path | Auth | Body |
|---|---|---|---|
| `PUT` | `/api/v1/dashboard/channels/:channel_id/config` | JWT（Admin 或 Streamer） | `{"seconds_per_point": 10}` |

- 路由掛在 `/api/v1/dashboard/` group，使用既有 `JWTAuth` + 新增 `RequireRole(Admin, Streamer)` middleware
- upsert 語意（不存在則建立，存在則更新）

### 設計取捨 — Channel Config

| 考量 | 決策 |
|---|---|
| `seconds_per_point` 每次 heartbeat 查 DB？ | MVP 直接查（PK lookup 夠快），不加 cache |
| `staleThreshold` / `maxDelta` 要不要開放設定？ | 不開放，這兩個值是安全機制，不是業務參數 |
| Streamer 能改其他人的頻道嗎？ | MVP 不做 channel ownership 驗證，依帳號角色授權 |

---

## 已知限制 / 後續待補

- [ ] Stale session 定期清理 cron job（目前只在 `StartSession` 時觸發關閉）
- [ ] `source` 欄位目前無 CHECK constraint，可視需求補上

---

## Phase 2 預告

- 前端 Extension UI：顯示餘額、heartbeat 狀態
- `GET /api/v1/points/balance`：一般帳號查詢端點
- `GET /api/v1/points/transactions`：交易記錄
- Bits 發點整合（`source = "bits"`）
- Claim 上鏈（`spendable_balance → Soulbound ERC-20 mint`）

---

## 實作計劃備忘

### Issue #61 — UUID v7（隨本次順帶處理）

本次修改以下三個檔案時，同步將 `uuid.New()` 改為 `uuid.New7()`（時序 UUID，避免 B-tree index fragmentation）：

| 檔案 | 改動點 |
|---|---|
| `backend/internal/models/points.go` | `PointsLedger.BeforeCreate`、`PointsTransaction.BeforeCreate` |
| `backend/internal/models/watch_session.go` | `WatchSession.BeforeCreate` |
| `backend/internal/services/watch_service.go` | `ID: uuid.New()` for WatchSession |

其餘 model（`user.go`、`auth_provider.go`、`address.go`、`refresh_token.go`、`email_auth.go`）與 `extension_service.go` 留給 Issue #61 獨立處理。詳見 [docs/uuid-v7.md](uuid-v7.md)。

### PR #62 重疊分析

PR #62（`users.role` VARCHAR → ENUM）也改動了 `backend/cmd/server/main.go`，本次計劃同樣需要修改此檔案。

| 項目 | PR #62 改動 | 本次計劃改動 |
|---|---|---|
| `main.go` | 新增 `CREATE TYPE user_role AS ENUM` block（AutoMigrate 前） | 更新 partial index SQL 欄位名（AutoMigrate 後） |

兩個改動在不同位置，無邏輯衝突。PR #62 merge 後本次 branch 需 rebase 解決 git conflict。

---

## 架構改動對照

本節記錄設計過程中的關鍵決策轉折，說明舊設計、新設計與原因。

| 項目 | 舊設計 | 新設計 | 原因 |
|---|---|---|---|
| 觀眾識別鍵 | `twitch_user_id VARCHAR(255)` | `user_id UUID FK → users.id` | 流程是「先登入才授權」，帳號主體是 tachigo `users`，Twitch 是附掛的 auth provider |
| 點數帳本範圍 | 全平台共用一本 | 每個觀眾 × 每個頻道各自獨立 | 每個實況主的點數互不流通，`points_ledgers` 唯一鍵改為 `(user_id, channel_id)` |
| Watch 路由 middleware | `ExtJWTAuth`（驗 Extension JWT） | `JWTAuth`（驗 tachigo JWT） | watch 端點已要求先登入取得 tachigo JWT，Extension JWT 不再適用 |
| `channel_id` 來源 | 從 Extension JWT claims 直接取出 | 由前端透過 request body 傳入 | tachigo JWT 不含頻道資訊；`balance` 端點用 query param |
| `WatchService` 參數型別 | `twitchUserID string` | `userID uuid.UUID` | 對應識別鍵型別變更，與 `users.id` FK 一致 |
| `GetBalance` 簽名 | `GetBalance(twitchUserID string)` | `GetBalance(userID uuid.UUID, channelID string)` | 帳本改為 per-channel，查詢時需同時提供 user 與 channel |

---

## 相關 Issues / PR

- [Issue #59](https://github.com/nurockplayer/tachigo/issues/59) — watch-to-points MVP 主票
- [Issue #58](https://github.com/nurockplayer/tachigo/issues/58) — auth_providers 設計討論
- [PR #52](https://github.com/nurockplayer/tachigo/pull/52) — Phase 1 & 2 實作
- 實作：[backend/internal/services/watch_service.go](../backend/internal/services/watch_service.go)
- Migration：[backend/migrations/003_watch_points.sql](../backend/migrations/003_watch_points.sql)
