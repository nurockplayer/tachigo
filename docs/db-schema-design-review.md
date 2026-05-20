# DB Schema 設計審查

> 日期：2026-05-17
> 範圍：`services/api/migrations/`、`services/api/internal/models/`、Atlas loader、DB write path service、測試用手寫 schema。
> 性質：設計審查與修正方向，不是已核准的 production migration。

## 結論

目前最大的問題不是單一欄位設錯，而是 schema source of truth 還沒有完全收斂。正式 runtime 已交給 Atlas migrations，但實際 schema 仍同時受到四種來源影響：歷史 SQL migrations、GORM model tags、`cmd/loader` 的 custom SQL、測試裡重複手寫的 SQLite/PostgreSQL DDL。這讓「未來 Atlas diff 誤刪歷史 invariant」與「測試 schema 以為有保護、正式 schema 其實沒有」成為主要風險。

必須先做的不是一次大改所有表，而是先把 schema contract 鎖住：列出每張表的 canonical FK、delete behavior、check constraints、unique/index、timestamp/nullability，再用測試與 CI 擋住 drift。之後再用多個小 PR 加回缺的 DB 保護。

## Schema 來源現況

| 來源 | 路徑 | 目前角色 | 風險 |
|---|---|---|---|
| 歷史 SQL migrations | `services/api/migrations/001_init.sql` 到 `020_atlas_reconcile_current_schema.sql` | Runtime migration history | 早期 migration 與目前 loader 在 FK、cascade、timestamp default、integer width 上不完全一致。 |
| GORM models | `services/api/internal/models/*.go` | Atlas loader 的主要 schema source | GORM tags 表達不了 partial index、部分 composite FK、部分 delete behavior；有些 model 沒有 relationship 欄位，因此 loader 不會產生 FK。 |
| Loader custom SQL | `services/api/cmd/loader/main.go` | 補 GORM 無法表達的 invariant | 目前只補部分高風險 invariant，沒有完整覆蓋所有歷史 FK / cascade / checks。 |
| 測試手寫 schema | `services/api/internal/services/testutil_test.go`、`services/api/internal/handlers/testutil_test.go`、`services/api/internal/services/testutil_pg_test.go` | SQLite/PG test schema | 與 migrations/loader 有 drift，會讓測試保護面與 production schema 不一致。 |

## P0 問題

### 1. FK 與 delete behavior 沒有 canonical matrix

Atlas loader 目前會輸出部分 FK，例如 `points_transactions.ledger_id`、`claim_items.*`、raffle 相關 FK；但缺少多個 domain-critical 關聯的 FK 或 delete behavior preservation。

高風險例子：

| 欄位 | 歷史/測試期待 | Loader 現況 | 影響 |
|---|---|---|---|
| `points_ledgers.user_id` | `003_watch_points.sql` 有 `REFERENCES users(id)` | loader 沒有 FK | Ledger 可孤兒化；claim composite FK 只能證明 ledger 與 claim user 對齊，不能證明 user 存在。 |
| `watch_sessions.user_id` | `003_watch_points.sql` 有 `REFERENCES users(id)` | loader 沒有 FK | Session / active viewer 可孤兒化，airdrop 與 stats 會吃到不合法 user。 |
| `streamers.user_id` | `006_streamers.sql` 有 `ON DELETE CASCADE` | loader 沒有 `user_id` FK，只補 `agency_user_id` FK | Dashboard/agency 權限可能指向不存在的 streamer user。 |
| `email_verifications.user_id` | `002_email_auth.sql` 有 `ON DELETE CASCADE` | loader 沒有 FK | Token table 可殘留孤兒列。 |
| `watch_time_stats.user_id` | test schema 有些版本期待 FK | loader / migration 020 沒有 FK | 累積統計可與 users 脫鉤。 |
| `broadcast_time_stats.streamer_id` / `broadcast_time_logs.streamer_id` | service 以 `auth_providers.user_id` 填入 | loader / migration 020 沒有 FK | broadcaster 統計可孤兒化。 |

同時，部分歷史 FK 有 `ON DELETE CASCADE`，但 loader 產生的 GORM FK 沒有 cascade，例如 `auth_providers`、`shipping_addresses`、`refresh_tokens`、`claims`。如果未來把 loader 當唯一 desired schema，可能會把既有 delete behavior 視為 drift。

### 2. Points ledger 缺少 DB-level accounting constraints

`points_ledgers` 與 `points_transactions` 是最核心帳本，但 schema 目前主要依賴 service transaction 保護。

目前缺口：

- `points_ledgers.cumulative_total >= 0`
- `points_ledgers.spendable_balance >= 0`
- `points_ledgers.spendable_balance <= points_ledgers.cumulative_total`
- `points_transactions.delta <> 0`
- `points_transactions.balance_after >= 0`
- `points_transactions.source` allowed values
- `watch_time` / `click` transaction 必須帶 `watch_session_id`，其他 source 應維持 null

`points_transactions.source` 目前是自由字串；Go constants 已經有 `bits`、`t_point`、`watch_time`、`click`、`spend`、`claim`、`airdrop`，但 DB 沒擋拼字錯誤或未定義 source。這會讓歷史帳本很難審計。

### 3. Channel ownership 模型有雙軌與 uniqueness 缺口

目前存在兩條 agency/channel 表示法：

- `streamers.agency_user_id`
- `agency_streamers(agency_id, channel_id)`

`020_atlas_reconcile_current_schema.sql` 會從 `agency_streamers` backfill `streamers.agency_user_id`，也會擋「要 backfill 的 channel 同時對到多個 agency」的情況。但 schema 之後仍允許 `agency_streamers` 繼續寫入多個 agency 指向同一個 `channel_id`。

另外，`streamers` 只保證 `(user_id, channel_id)` 唯一，沒有保證 `channel_id` 全域唯一。可是 `AgencyService.ListStreamerUserIDs` 查到同一 channel 對多個 user 時會回 `ErrDuplicateChannelID`，代表 service 假設「一個 channel 只屬於一個 streamer user」。這個假設目前沒有被 DB 強制。

### 4. `$TACHI` balance 型別語意不一致

`tachi_balances.balance` 在 SQL / model 裡是 `NUMERIC(20,6)`，Go 欄位與 service 卻用 `int64` 當 whole-token balance。`ClaimService` 讀取時甚至用 `SELECT CAST(balance AS BIGINT)`。

這代表目前有三種語意混在一起：

- DB 看起來允許 6 位小數。
- Go service 只支援整數 token。
- ERC-20 實際上有 18 decimals，鏈上呼叫用 `amount * 10^18`。

如果 DB 出現小數，Go 讀取會被 cast 成整數；如果未來要支援 raw units，`NUMERIC(20,6)` 又不夠。這需要明確選一種 canonical representation。

## P1 問題

### 5. Raffle schema under-constrained

`RaffleStatus` 與 `RaffleSource` 在 Go 端有 enum constants，但 DB 沒有 check constraints。`raffle_draws.claim_expires_at`、`raffle_draws.drawn_at`、`raffle_claims.submitted_at` 在 model 是 `time.Time`，test schema 多數也視為 `NOT NULL`，但 loader / migration 020 產生 nullable columns。

`raffle_draws.claim_token` 已改存 SHA-256 hex digest，但 schema 只用 `varchar(255)`，沒有保護長度或格式。這不是立即 blocker，但會弱化 token repair 後的資料保證。

### 6. Coupon redemption 缺少 burn idempotency constraint

`claims.tx_hash` 有 partial unique index，但 `coupon_redemptions.tx_hash` 沒有 unique constraint。Spend flow 在 burn 成功後建立 redemption 記錄，若同一 tx hash 被重放或 retry 寫入兩次，DB 不會擋。

至少需要決定 `tx_hash` 是否全域唯一；若同一 tx 可合法對多 coupon，也應建立更明確的 composite uniqueness 與狀態轉移規則。

### 7. Identity uniqueness 是 case-sensitive

`users.email`、`users.username`、`auth_providers(provider, provider_id)` 都是一般 unique / partial unique。PostgreSQL 預設 case-sensitive，因此同一 email 或 wallet address 可能用不同大小寫繞過 uniqueness。

Web3 service 目前會 canonicalize address，但 DB constraint 本身沒有防線。Email provider 與 agency email 也有同樣問題。

### 8. Timestamp nullability/default drift 還沒有收束

歷史 SQL 常用 `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()` / `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`；loader 多數 timestamp 是 nullable、無 default。測試 helper 又在不同 package 裡各自選擇不同 DDL。

這會造成兩種問題：

- 直接 SQL / migration / repair script 可能寫出 null timestamps。
- 測試環境與 production 行為不一致。

### 9. 手寫 test schema 重複且互相 drift

目前至少有 services test、handlers test、router test、PG test 四套手寫 schema。它們不是完全一致，例如 `watch_sessions.user_id` 是否有 FK、`tachi_balances.balance` 是 integer 還是 numeric、raffle nullable/default 都不一致。

這些 helper 本身已經成為第五份 schema source。它們適合快速 SQLite 測試，但不應該再承擔 schema 真相。

### 10. 高成長查詢缺少 composite index

目前多數 index 是單欄位。幾個已知 service 查詢會隨交易量變大而吃力：

| 查詢 | 目前行為 | 建議 index |
|---|---|---|
| `ListTransactions` | `WHERE ledger_id = ? ORDER BY created_at DESC, id DESC LIMIT 50` | `points_transactions(ledger_id, created_at DESC, id DESC)` |
| `AirdropService.todayTotal` | join ledger 後依 `source` + `created_at` 篩日區間 | `points_transactions(source, created_at, ledger_id)` 或 airdrop partial index |
| `GetBroadcastStats` | `WHERE streamer_id = ? AND channel_id = ? AND recorded_at >= ?` | `broadcast_time_logs(streamer_id, channel_id, recorded_at)` |
| compensation queue | `coupon_redemptions.status = 'compensation-needed'` | 已有 partial index，應保留 |

### 11. UUID v7 policy 尚未完整落到 schema 與 models

`docs/uuid-v7.md` 已定義正式 model 的 primary key 優先用 UUID v7。實際上多個舊 model 的 `BeforeCreate` 仍使用 `uuid.New()`，DB default 也都是 `gen_random_uuid()` 作 fallback。

這不是帳務正確性 blocker，但會讓高寫入表的 B-tree locality 不一致。應與既有 `plans/uuid-v7-migration.md` 對齊，不要混進 schema integrity PR。

## 建議修正順序

1. 先建立 canonical schema contract：每張表的 FK、delete behavior、checks、unique/index、timestamp policy。
2. 加 CI drift guard：migrations apply 後的 PostgreSQL schema 必須與 loader desired schema 在「允許差異清單」外保持一致。
3. 補 referential integrity：先加缺的 FK 與 delete behavior preservation，使用 data audit + `NOT VALID` / `VALIDATE CONSTRAINT` 降低風險。
4. 補 ledger checks：先擋不可能合法的資料，再討論更細的 transaction direction 模型。
5. 收斂 channel ownership：決定 `streamers.agency_user_id` 是否取代 `agency_streamers`，並加上 channel uniqueness。
6. 決定 `$TACHI` balance representation：whole-token `BIGINT` 或 raw-unit `NUMERIC(78,0)` / decimal wrapper，不能繼續 `NUMERIC(20,6)` + `int64 cast`。
7. 補 raffle/coupon/auth constraints 與 composite indexes。
8. 收斂測試 schema：保留 SQLite helper，但由一份 shared schema helper 產生；PostgreSQL-specific invariant 必須有 PG test。

詳細拆分與執行步驟見 [`plans/db-schema-remediation.md`](../plans/db-schema-remediation.md)。
