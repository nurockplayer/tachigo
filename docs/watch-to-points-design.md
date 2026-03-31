# Watch-to-Points 設計文件

> **狀態：** 實作中（refs #59）
> **最後更新：** 2026-03-31

---

## 概述

觀眾在 Twitch Extension 觀看直播時定時發送 heartbeat，後端累積觀看秒數，每 60 秒發放 1 點。點數記錄在全平台共用帳本（`points_ledgers`），不區分頻道。

---

## 認證流程

```
觀眾開啟 Extension
  → 前端呼叫 POST /api/v1/extension/auth/login（帶 Extension JWT）
  → 後端 LoginWithExtension：找到或建立 users 記錄，回傳 tachigo JWT
  → 前端存下 tachigo JWT

觀眾開始觀看
  → 前端呼叫 POST /api/v1/extension/watch/start（帶 tachigo JWT + channel_id）
  → 定時呼叫 POST /api/v1/extension/watch/heartbeat（每 30 秒）
  → 離開時盡力呼叫 POST /api/v1/extension/watch/end（見「Session 結束機制」）
```

**為什麼先登入才能累積點數：**
- 使用 `users.id UUID` 作為識別鍵，確保 FK 完整性
- 點數帳本直接與帳號系統關聯，Phase 2 claim 上鏈不需要額外橋接
- 避免匿名觀眾累積後無法認領的問題

---

## 資料模型

### watch_sessions

| 欄位 | 型別 | 說明 |
|---|---|---|
| `id` | UUID PK | |
| `user_id` | UUID FK → users.id | 識別觀眾（需已登入） |
| `channel_id` | VARCHAR(255) | 頻道 ID（來自前端） |
| `accumulated_seconds` | BIGINT default 0 | 本 session 累積觀看秒數 |
| `rewarded_seconds` | BIGINT default 0 | 已換算為點數的秒數 |
| `last_heartbeat_at` | TIMESTAMPTZ NOT NULL default now() | 最後 heartbeat 時間 |
| `is_active` | BOOLEAN NOT NULL default true | 是否為進行中 session |
| `ended_at` | TIMESTAMPTZ NULL | session 結束時間 |

**Partial unique index：** `(user_id, channel_id) WHERE is_active = true`
→ 每個觀眾在每個頻道同時只能有一個活躍 session

**Session lifecycle：**
```
active  : is_active = true,  ended_at = NULL
finished: is_active = false, ended_at = <timestamp>
```

### points_ledgers

全平台共用帳本，每位觀眾只有一本，不區分頻道。

| 欄位 | 型別 | 說明 |
|---|---|---|
| `id` | UUID PK | |
| `user_id` | UUID UNIQUE FK → users.id | |
| `cumulative_total` | BIGINT default 0 | 歷史累積點數（只增不減） |
| `spendable_balance` | BIGINT default 0 | 可消費點數餘額 |

### points_transactions

| 欄位 | 型別 | 說明 |
|---|---|---|
| `id` | UUID PK | |
| `ledger_id` | UUID FK → points_ledgers.id | |
| `watch_session_id` | UUID NULL | watch_time 來源必填，bits/spend 為 NULL |
| `source` | VARCHAR(50) | `watch_time` / `bits` / `spend` |
| `delta` | BIGINT | 變動量（正 = 獲得，負 = 消費） |
| `balance_after` | BIGINT | 交易後餘額快照 |
| `note` | TEXT NULL | 備註 |

---

## API 端點

所有 watch 端點使用標準 tachigo JWT（`Authorization: Bearer <tachigo_jwt>`），由 `JWTAuth` middleware 保護。

| Method | Path | 說明 | Body |
|---|---|---|---|
| POST | `/api/v1/extension/watch/start` | 開始或取回活躍 session | `{ "channel_id": "..." }` |
| POST | `/api/v1/extension/watch/heartbeat` | 更新計時，達門檻時發點 | `{ "channel_id": "..." }` |
| POST | `/api/v1/extension/watch/end` | 主動結束 session（盡力送出） | `{ "channel_id": "..." }` |
| GET | `/api/v1/extension/watch/balance` | 查詢點數餘額 | — |

---

## 發點邏輯

```
每次 heartbeat：
  delta = min(now - last_heartbeat_at, 30s)  ← 上限 30 秒，防止長時間斷線後補算過多
  accumulated_seconds += delta
  pending = accumulated_seconds - rewarded_seconds
  points_to_award = pending / 60             ← 每 60 秒發 1 點
  rewarded_seconds += points_to_award * 60

若 points_to_award > 0：
  → Atomic upsert points_ledgers
  → 寫入 points_transactions（帶 watch_session_id）
```

---

## Session 結束機制

Server **無法主動偵測** client 斷線，因此採用兩層機制：

### 層一：Client 主動呼叫 EndSession（理想路徑）

前端在以下時機呼叫 `POST /extension/watch/end`：
- 觀眾關閉 Extension 面板
- 瀏覽器 `beforeunload` event

**限制：** `beforeunload` 不保證成功送出（例如手機強制關閉、網路中斷），因此這條路徑屬於「盡力送出（best-effort）」。

### 層二：Server 偵測 Stale Session（保底機制）

若某 session 的 `last_heartbeat_at` 超過 `staleThreshold`（目前 2 分鐘）未更新，下次該觀眾在同頻道呼叫 `StartSession` 時，server 會自動：
1. 將舊 session 設為 `is_active = false, ended_at = now()`
2. 建立新 session

**結果：** 即使 client 沒有主動結束，session 最多延遲 `staleThreshold` 才會被關閉。這段時間不會繼續累積秒數（因為沒有 heartbeat 進來）。

### 設計取捨

| 考量 | 決策 |
|---|---|
| staleThreshold 設多少？ | 目前 2 分鐘，heartbeat 每 30 秒一次，等於允許錯過 4 次 heartbeat |
| 是否需要主動清理任務（cron）？ | MVP 不做，stale 只在下次 StartSession 時觸發 |
| `ended_at` 精確度 | 非精確值，反映「最後一次 heartbeat 後的關閉時間」 |

---

## 並發安全機制

| 問題 | 解法 |
|---|---|
| 兩個 heartbeat 同時進來，double-award | `SELECT FOR UPDATE` 鎖住 session 列 |
| 兩個 start 同時進來，違反 partial unique index | Transaction + Savepoint + fallback 查詢 |
| balance 更新衝突 | `INSERT ... ON CONFLICT DO UPDATE` atomic upsert |

---

## Phase 2 預告

- 前端 Extension UI：顯示餘額、heartbeat 狀態
- `GET /api/v1/points/balance`：一般帳號查詢端點
- `GET /api/v1/points/transactions`：交易記錄
- Bits 發點整合（`source = "bits"`）
- Claim 上鏈（`spendable_balance → Soulbound ERC-20 mint`）
- Stale session 定期清理 cron job（非 MVP）

---

## 相關 Issues / PR

- [Issue #59](https://github.com/nurockplayer/tachigo/issues/59) — watch-to-points MVP 主票
- [Issue #58](https://github.com/nurockplayer/tachigo/issues/58) — auth_providers 設計討論
- [PR #52](https://github.com/nurockplayer/tachigo/pull/52) — Phase 1 & 2 實作
