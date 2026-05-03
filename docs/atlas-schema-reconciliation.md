# Atlas Schema Reconciliation

本文件追蹤 `services/api` 從 GORM `AutoMigrate` 遷移到 Atlas 前，必須先 reconcile 的 schema sources。

這份文件刻意保守。標記為「必須保留」的項目，代表它已存在於目前程式碼或 migration 中；Atlas 接管後不得讓該 invariant 消失。

## Inputs

| Source | Path | Role | Current Status |
|---|---|---|---|
| Historical SQL migrations | `services/api/migrations/001_init.sql` through `019_coupon_redemptions.sql` | 手寫 schema history | 檔案存在於 repo，但 issue `#463` 已說明目前沒有 migration tool 執行它們。 |
| GORM models | `services/api/internal/models/*.go` | Application schema model | 目前 `AutoMigrate` source。包含部分沒有完整出現在 `001-019` 的表。 |
| Runtime schema patches | `services/api/cmd/server/main.go` | Startup-time DDL 與 data repair | API startup 仍會執行。這些 patch 必須遷移到 Atlas，或明確保留為非 schema runtime work。 |
| Atlas config | `services/api/atlas.hcl` | 未來 schema diff entrypoint | 本文件建立時，`develop` 尚未有此檔案。 |
| Closed attempt | PR `#476` | 前一次 implementation attempt | 已關閉，原因是 tooling、baseline、runtime migration strategy 被包在同一顆 PR，且 baseline 假設不安全。 |

## Current Runtime DDL In `cmd/server/main.go`

| Runtime Behavior | Current Code | Reconciliation Decision |
|---|---|---|
| 建立 `user_role` enum | `initializeUserRoleEnum` 建立 `('viewer', 'streamer', 'agency', 'admin')`，並在缺少時補上 `agency`。 | 必須保留。Atlas loader 與 migrations 必須使用相同 label set 與 ordering。 |
| 執行 GORM `AutoMigrate` | `db.AutoMigrate(...)` 套用所有 models。 | 只能在 Atlas baseline 已於 staging 驗證後移除。 |
| 補 `tachi_balances.user_id` FK | 手寫 `ALTER TABLE tachi_balances ADD CONSTRAINT fk_tachi_balances_user_id`。 | 必須以 Atlas migration 或 model association 保留。 |
| 補 coupon redemption checks | `ensureCouponRedemptionRuntimeSchema` 增加 amount/status constraints。 | 必須保留於 Atlas migration。 |
| 補 coupon compensation index | `coupon_redemptions(status)` 的 partial index，條件為 status = `compensation-needed`。 | 若 query behavior 仍依賴它，必須保留。 |
| 補 active watch session uniqueness | Partial unique index `idx_watch_sessions_active_user_channel`。 | 必須保留。這是 concurrency invariant，不只是效能 index。 |
| 補 points ledger uniqueness | Unique index `idx_points_ledgers_user_channel`。 | 必須保留。watch heartbeat upsert 依賴 `ON CONFLICT (user_id, channel_id)`。 |
| 補 external transaction uniqueness | Partial unique index `idx_points_transactions_external_transaction_id`。 | 除非另開 issue 移除 idempotency guarantee，否則必須保留。 |
| 補 streamer uniqueness | Unique index `idx_streamers_user_channel`。 | 必須保留，除非有等價 constraint 取代。 |
| 執行 streamer agency migration | `applyStreamerAgencyMigration(db)`。 | 需與 migrations `010`、`011` 比對，缺漏部分要保留到 Atlas。 |
| Hash raffle claim tokens | 將 36-char token 一次性轉成 SHA-256 hex。 | 與 schema migration 分開處理。只有 production data 仍需要此 repair 時才保留。 |

## Historical Migration Inventory

| Migration | Main Schema Surface | Notes For Atlas Baseline |
|---|---|---|
| `001_init.sql` | `user_role`、`users`、`auth_providers`、`shipping_addresses`、`refresh_tokens`、`web3_nonces` | Enum 最初只有 `viewer`、`streamer`、`admin`；目前 runtime 另補 `agency`。 |
| `002_email_auth.sql` | Email verification 與 password reset tables | 比對 unique constraints 與 token indexes 是否符合 GORM tags。 |
| `003_watch_points.sql` | `watch_sessions`、`points_ledgers`、`points_transactions` | 包含 active-session partial unique index 與 ledger uniqueness，都是必須保留的 invariants。 |
| `004_rbac_roles.sql` | Role 相關變更 | 確認它是否已由目前 enum handling 表達。 |
| `005_channel_config.sql` | `channel_configs` | 比對 numeric defaults 與 timestamp nullability。 |
| `006_streamers.sql` | `streamers` | Runtime 也建立 `idx_streamers_user_channel`；需確認最終 desired uniqueness。 |
| `007_click_boost.sql` | Watch session click boost fields | 與 `WatchSession` model 比對。 |
| `008_points_transaction_sku.sql` | `points_transactions.sku` | 與 `PointsTransaction` 的 nullable/type 比對。 |
| `009_channel_config_multiplier.sql` | `channel_configs.multiplier` | 與 model type/default 比對。 |
| `010_agency_streamers.sql` | `agency_streamers` | 與 `AgencyStreamer` model 及 runtime agency migration 比對。 |
| `011_streamers_agency.sql` | `streamers.agency_user_id` | 必須 reconcile FK names 與 delete behavior。 |
| `012_tachi_balances.sql` | `tachi_balances` | Runtime 另補 FK；Atlas 接管前必須由 migration 擁有。 |
| `013_airdrop.sql` | Channel config airdrop fields | 與 model defaults/limits 比對。 |
| `014_auth_provider_partial_unique.sql` | Soft-delete-aware auth provider uniqueness | 必須保留 partial unique indexes。 |
| `015_claims.sql` | `claims`、`claim_items` | 比對 checks、uniqueness、FK behavior、query indexes。 |
| `016_claims_composite_fk.sql` | Composite claim/ledger/transaction FKs | 未經 service-level review 不得移除。 |
| `017_claim_finalize_failed.sql` | Claim status check expansion | 確認 final status enum/check 符合 service behavior。 |
| `018_points_transaction_external_id.sql` | External transaction ID idempotency | 必須保留 non-null external transaction ID 的 partial unique index。 |
| `019_coupon_redemptions.sql` | Coupon redemption table、checks、compensation index | Runtime 有部分重複補強；Atlas 最終應成為單一 owner。 |

## Known Gaps To Verify

| Area | Evidence | Risk | Required Decision |
|---|---|---|---|
| Raffle tables | Models 包含 `Raffle`、`RaffleEntry`、`RaffleDraw`、`RaffleClaim`；`001-019` 沒有建立 raffle tables。 | Baseline 若產生裸 `CREATE TABLE`，在 AutoMigrate 已建表的環境會失敗。 | 決定採 import baseline，或產生 apply-safe guarded SQL。 |
| Watch and broadcast stats | Models 包含 `WatchTimeStat`、`BroadcastTimeStat`、`BroadcastTimeLog`；`001-019` 沒有 dedicated SQL migration。 | 與 raffle tables 有相同 table-exists risk。 | 產生 SQL 前先決定 baseline strategy。 |
| Partial unique indexes | 多個 invariants 以 raw SQL 存在，因 GORM tags 無法完整表達。 | Atlas provider 若沒有看到 desired schema，可能產生 drop。 | 用 Atlas desired schema 或手寫 migration 明確保留。 |
| Enum ordering | Historical SQL、runtime enum creation、Atlas loader 必須一致。 | PostgreSQL enum ordering 影響 comparison 與 drift detection。 | 使用 canonical order：`viewer`、`streamer`、`agency`、`admin`，對齊目前 runtime code。 |
| Runtime data migration | Server 啟動時會 hash 36-char raffle claim tokens。 | Schema migration 工作可能讓不相關 data repair 永遠留在 runtime。 | 另行判斷 production 是否仍需要；不得藏在 Atlas tooling PR。 |
| Dependency anchoring | Atlas provider 可能只被 loader command import。 | 若只存在 build-ignored loader，`go mod tidy` 可能移除 tool dependency。 | Tooling PR 新增 `services/api/tools.go`。 |

## Reconciliation Procedure

產生任何 baseline migration 前，先完成以下程序：

1. 建立乾淨 PostgreSQL database，依序套用 `services/api/migrations/001-019`。
2. Dump migration-only schema：

   ```bash
   pg_dump --schema-only --no-owner --no-privileges "$MIGRATIONS_DATABASE_URL" > /tmp/tachigo-sql-migrations-schema.sql
   ```

3. 建立第二個乾淨 PostgreSQL database，啟動目前 API，讓 GORM `AutoMigrate` 與 runtime patches 執行完。
4. Dump AutoMigrate/runtime schema：

   ```bash
   pg_dump --schema-only --no-owner --no-privileges "$AUTOMIGRATE_DATABASE_URL" > /tmp/tachigo-automigrate-schema.sql
   ```

5. 比對兩份 dump，並分類每個差異：

   ```bash
   diff -u /tmp/tachigo-sql-migrations-schema.sql /tmp/tachigo-automigrate-schema.sql
   ```

6. 在新增 `020_atlas_baseline.sql` 前，先把最終決策更新回本文件。

## Baseline Strategy Decision Record

Baseline PR commit SQL 前，必須先在 PR body 選定 baseline strategy。

| Strategy | Use When | Required Proof |
|---|---|---|
| Import baseline | 既有環境已經透過 AutoMigrate 與 runtime patches 擁有目標 schema。 | Atlas history 可標記既有狀態，不會把 unsafe `CREATE TABLE` 重套到既有 DB。 |
| Apply-safe baseline | 既有環境需要 SQL 從已知 current state reconcile。 | Migration 可同時套用到 clean migration DB 與已跑過目前 `AutoMigrate` 的 DB。 |

## Completion Criteria

- [ ] `001-019` 中每個 table/index 都已被保留、明確替代，或用 issue-backed rationale 明確移除。
- [ ] `cmd/server/main.go` 中每個 runtime schema patch 都已進入 Atlas，或被記錄為刻意保留的非 schema runtime behavior。
- [ ] Raffle/watch/broadcast tables 已有安全 baseline strategy，且有考慮 AutoMigrate 可能已建表。
- [ ] Atlas loader output 包含或保留 GORM 無法表達的 invariants。
- [ ] `AutoMigrate` removal 被 baseline staging validation 阻擋，不能提前執行。
