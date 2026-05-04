# Atlas Schema Reconciliation

本文件追蹤 `services/api` 從 GORM `AutoMigrate` 遷移到 Atlas 前，必須先 reconcile 的 schema sources。

這份文件刻意保守。標記為「必須保留」的項目，代表它已存在於目前程式碼或 migration 中；Atlas 接管後不得讓該 invariant 消失。

## Inputs

| Source | Path | Role | Current Status |
|---|---|---|---|
| Historical SQL migrations | `services/api/migrations/001_init.sql` through `019_coupon_redemptions.sql` | 手寫 schema history | 檔案存在於 repo，但 issue `#463` 已說明目前沒有 migration tool 執行它們。 |
| GORM models | `services/api/internal/models/*.go` | Application schema model | 目前 `AutoMigrate` source。包含部分沒有完整出現在 `001-019` 的表。 |
| Runtime schema patches | `services/api/cmd/server/main.go` | Startup-time DDL 與 data repair | API startup 仍會執行。這些 patch 必須遷移到 Atlas，或明確保留為非 schema runtime work。 |
| Atlas config | `services/api/atlas.hcl` | 未來 schema diff entrypoint | PR `#491` 已加入，使用 `external_schema` 執行 `go run -mod=mod ./cmd/loader`。 |
| Closed attempt | PR `#476` | 前一次 implementation attempt | 已關閉，原因是 tooling、baseline、runtime migration strategy 被包在同一顆 PR，且 baseline 假設不安全。 |

## Validation Run: 2026-05-04

本次 reconciliation 以 `develop` 上 PR `#491` merge 後的狀態驗證，commit 為 `d5e9b9e`。

### Schema States Built

| State | How It Was Built | Dump |
|---|---|---|
| Historical migrations | PostgreSQL 16 clean DB，依序套用 `services/api/migrations/001-019`。 | `/tmp/tachigo-sql-migrations-schema.sql` |
| Runtime AutoMigrate | PostgreSQL 16 clean DB，啟動目前 `cmd/server`，讓 enum init、`AutoMigrate`、manual runtime patches、data repair 跑完後停止 server。 | `/tmp/tachigo-automigrate-schema.sql` |
| Atlas GORM loader | `go run ./cmd/loader/main.go` 輸出 SQL，套到 PostgreSQL 16 clean DB。 | `/tmp/tachigo-loader-schema.sql` |

### Catalog Counts

| State | Tables | Columns | Constraints | Indexes | Enum Labels |
|---|---:|---:|---:|---:|---:|
| Historical migrations | 17 | 135 | 52 | 58 | 4 |
| Runtime AutoMigrate | 24 | 183 | 48 | 75 | 4 |
| Atlas GORM loader | 24 | 183 | 47 | 77 | 4 |

### High-Level Result

| Comparison | Result | Decision |
|---|---|---|
| Runtime AutoMigrate vs Atlas loader tables | Exact match. | Loader is a usable table/column source for current models. |
| Runtime AutoMigrate vs Atlas loader columns | Exact match. | Column-level runtime drift is not currently between runtime and loader; major column drift is between historical SQL and runtime/loader. |
| Runtime AutoMigrate vs Atlas loader enum labels/order | Exact match: `viewer`, `streamer`, `agency`, `admin`. | This is current fresh-create runtime order, not proof that production/migrated enum order should be rebuilt. |
| Runtime AutoMigrate vs Atlas loader constraints | Loader is missing runtime `fk_streamers_agency_user_id`. | Fix loader or add an explicit GORM association before generating baseline from loader. |
| Runtime AutoMigrate vs Atlas loader indexes | Loader adds `idx_auth_providers_provider_provider_id_active` and `idx_auth_providers_web3_user_active`; runtime fresh-create does not. | Preserve these historical soft-delete invariants in Atlas; do not remove just because runtime fresh-create lacks them. |

## Verified Reconciliation Gaps

| Area | Evidence From Validation | Classification | Decision Before Baseline |
|---|---|---|---|
| Tables absent from `001-019` | Runtime/loader have `broadcast_time_logs`, `broadcast_time_stats`, `watch_time_stats`, `raffles`, `raffle_entries`, `raffle_draws`, `raffle_claims`; migration-only DB does not. | model drift | Baseline must account for these tables. For existing AutoMigrate-created environments, emitting bare `CREATE TABLE` without guards is unsafe. |
| `user_role` enum order | Migration path produces `viewer`, `streamer`, `admin`, `agency`; runtime/loader fresh-create produces `viewer`, `streamer`, `agency`, `admin`. | false-drift risk | Preserve the label set. Do not create enum rebuild or reorder migration solely to normalize order. PR body for baseline must say whether it targets migrated order, fresh-create order, or explicitly accepts both. |
| `streamers.agency_user_id` FK | Runtime has `fk_streamers_agency_user_id`; loader does not. Migration `011_streamers_agency.sql` and runtime `applyStreamerAgencyMigration` both create it. | runtime patch missing from loader | Must be added to Atlas desired schema before baseline generation, either in loader custom constraints or via an explicit model association. |
| Auth provider soft-delete uniqueness | Migration `014_auth_provider_partial_unique.sql` and loader preserve `idx_auth_providers_provider_provider_id_active` and `idx_auth_providers_web3_user_active`; runtime fresh-create lacks them. | historical invariant not in runtime fresh-create | Keep these indexes in Atlas. They protect wallet/login rebinding semantics for soft-deleted auth providers. |
| Claim composite consistency | Migration `016_claims_composite_fk.sql` adds `claim_user_id`, composite FKs `fk_claim_items_claim_user`, `fk_claim_items_ledger_user`, `fk_claim_items_tx_ledger`, and supporting unique indexes. Runtime/loader keep `claim_user_id` but do not recreate the composite FKs/supporting unique indexes. | historical invariant not in runtime/loader | Do not drop silently. Either port these invariants into Atlas desired schema or open an issue-backed service decision to remove them. |
| Claim transaction hash uniqueness | Migration `015_claims.sql` creates partial unique `idx_claims_tx_hash_not_null`; runtime/loader do not. | historical invariant not in runtime/loader | Treat as preserve-by-default until claim service review confirms duplicate tx hashes are harmless. |
| User-owned row delete behavior | Several historical FKs use `ON DELETE CASCADE` (`auth_providers`, `shipping_addresses`, `refresh_tokens`, `claims`, `streamers`); runtime/loader GORM FKs often omit cascade. | behavior drift | Baseline must not rewrite cascade behavior without an explicit data-retention decision. |
| Timestamp nullability/defaults | Historical SQL frequently uses `NOT NULL DEFAULT now()` for `created_at` / `updated_at`; runtime/loader GORM schema leaves many timestamp columns nullable without defaults. | column drift | Baseline should avoid generating nullability/default churn until production/staging catalog confirms the actual deployed state and service-level expectations. |
| Integer width drift | `channel_configs.multiplier` and `daily_airdrop_limit` are `integer` in historical migrations and `bigint` in runtime/loader. | column drift | Choose one canonical type in baseline PR and document compatibility. Do not narrow production data without proof. |
| Constraint/index naming drift | Many semantically equivalent unique constraints/indexes have different names between historical SQL and GORM output, e.g. `users_email_key` vs `idx_users_email`, `streamers_user_id_channel_id_key` vs `idx_streamers_user_channel`, `tachi_balances_user_id_key` vs `idx_tachi_balances_user_id`. | name drift | Avoid rename-only churn unless Atlas requires it; prefer importing actual deployed names or using guarded SQL. |
| `tachi_balances.user_id` FK duplication | Runtime/loader fresh-create have both GORM-generated `fk_tachi_balances_user` and manual `fk_tachi_balances_user_id`; historical migration has a single FK under a generated name. | runtime patch overlap | Baseline must pick a canonical target or explicitly tolerate duplicate equivalent FKs while AutoMigrate still runs. |

## Current Runtime DDL In `cmd/server/main.go`

| Runtime Behavior | Current Code | Reconciliation Decision |
|---|---|---|
| 建立 `user_role` enum | `initializeUserRoleEnum` fresh-create 目前建立 `('viewer', 'streamer', 'agency', 'admin')`，但既有資料庫若由 `001_init.sql` 建立後再跑 `004_rbac_roles.sql`，實際 order 是 `('viewer', 'streamer', 'admin', 'agency')`；型別已存在時只會補 `agency`，不會重排。 | 必須保留 label set。Atlas reconciliation 不得把 enum order mismatch 視為必須重建 enum 的 drift；baseline 應以 production/migrated order 為準，或明確接受雙態。 |
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
| `004_rbac_roles.sql` | Role 相關變更 | 以 `ALTER TYPE ... ADD VALUE 'agency'` append 到既有 enum 後方；production/migrated order 會是 `viewer`、`streamer`、`admin`、`agency`。 |
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

## Remaining Baseline Decisions

這些項目已由 2026-05-04 validation run 證實存在；baseline PR 不能再把它們當成待查假設。

| Area | Evidence | Risk | Required Decision |
|---|---|---|---|
| Baseline strategy | Runtime/loader have 24 tables, historical migrations have 17. Some historical constraints are stricter than runtime/loader. | A single naive diff can either fail on existing tables or silently drop historical invariants. | PR body must choose `import baseline` or `apply-safe baseline`, and say which source state represents the deployed target. |
| Production/staging catalog | This document only proves clean migration path, clean runtime path, and loader path. | The actual deployed database may match runtime, historical SQL, or a mixed state after years of AutoMigrate/runtime patches. | Before removing `AutoMigrate`, dump staging/current schema and compare it with the selected baseline target. |
| Loader completeness | Loader currently misses `fk_streamers_agency_user_id`, while runtime creates it. | Atlas-generated baseline could omit a runtime FK that exists in server startup today. | Fix loader desired schema before using it for `atlas migrate diff`. |
| Historical invariants | Claim composite FKs, claim tx hash uniqueness, cascade behavior, and auth soft-delete indexes are not all expressible from plain GORM tags. | Atlas may generate destructive drift if these are absent from desired schema. | Preserve by hand in loader/baseline, or document issue-backed removal decisions before PR3. |
| Runtime data migration | Server startup hashes 36-char raffle claim tokens. | Schema migration work could leave unrelated data repair hidden in runtime forever. | Decide separately whether production still needs this repair; do not bury it in baseline SQL. |

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
