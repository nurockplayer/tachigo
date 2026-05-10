# Atlas Schema Reconciliation

本文件追蹤 `services/api` 從 GORM `AutoMigrate` 遷移到 Atlas 前，必須先 reconcile 的 schema sources。

這份文件刻意保守。標記為「必須保留」的項目，代表它已存在於目前程式碼或 migration 中；Atlas 接管後不得讓該 invariant 消失。

## Inputs

| Source | Path | Role | Current Status |
|---|---|---|---|
| Historical SQL migrations | `services/api/migrations/001_init.sql` through `019_coupon_redemptions.sql` | 手寫 schema history | 檔案存在於 repo；issue `#463` 指的是正式 migration tool / 自動流程尚未建立。本文件的 validation run 以手動 sequential replay 方式把 `001-019` 套到 clean DB。 |
| GORM models | `services/api/internal/models/*.go` | Application schema model | 過去是 `AutoMigrate` source；本階段也作為 Atlas GORM loader 的 schema source。包含部分沒有完整出現在 `001-019` 的表。 |
| Runtime schema patches | `services/api/cmd/server/main.go` | Startup-time DDL 與 data repair | Schema DDL 已由 Atlas migration / loader 補齊目標 state；API startup 移除留給後續 runtime ownership PR。 |
| Atlas config | `services/api/atlas.hcl` | schema diff entrypoint | PR `#491` 已加入，使用 `external_schema` 執行 `go run -mod=mod ./cmd/loader`。 |
| Closed attempt | PR `#476` | 前一次 implementation attempt | 已關閉，原因是 tooling、baseline、runtime migration strategy 被包在同一顆 PR，且 baseline 假設不安全。 |

## Validation Run: 2026-05-04

本次 reconciliation 以 `develop` 上 PR `#491` merge 後的狀態驗證，commit 為 `d5e9b9e`。Historical migrations state 是用 `psql` 逐檔、依檔名順序手動套用 `services/api/migrations/001-019`；這不代表 repo 已有產品化 migration runner。

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

除 enum ordering 以外，本文件的 drift 分類以「目前 runtime AutoMigrate + runtime patches」作為暫定 comparison target；baseline PR 仍必須在 PR body 明確選擇 migrated order、fresh-create order，或接受雙態 enum ordering。

| Comparison | Result | Decision |
|---|---|---|
| Runtime AutoMigrate vs Atlas loader tables | Exact match. | Loader is a usable table/column source for current models. |
| Runtime AutoMigrate vs Atlas loader columns | Exact match. | Column-level runtime drift is not currently between runtime and loader; major column drift is between historical SQL and runtime/loader. |
| Runtime AutoMigrate vs Atlas loader enum labels/order | Exact match: `viewer`, `streamer`, `agency`, `admin`. | This is current fresh-create runtime order, not proof that production/migrated enum order should be rebuilt. |
| Runtime AutoMigrate vs Atlas loader constraints | Loader now preserves runtime `fk_streamers_agency_user_id` plus claim composite invariants. | Loader is the desired schema source for Atlas diff after migration `020`. |
| Runtime AutoMigrate vs Atlas loader indexes | Loader adds `idx_auth_providers_provider_provider_id_active` and `idx_auth_providers_web3_user_active`; runtime fresh-create does not. | Preserve these historical soft-delete invariants in Atlas; do not remove just because runtime fresh-create lacks them. |

## Verified Reconciliation Gaps

| Area | Evidence From Validation | Classification | Decision Before Baseline |
|---|---|---|---|
| Tables absent from `001-019` | Runtime/loader have `broadcast_time_logs`, `broadcast_time_stats`, `watch_time_stats`, `raffles`, `raffle_entries`, `raffle_draws`, `raffle_claims`; migration-only DB does not. | model drift | Baseline must account for these tables. For existing AutoMigrate-created environments, emitting bare `CREATE TABLE` without guards is unsafe. |
| `user_role` enum order | Migration path produces `viewer`, `streamer`, `admin`, `agency`; runtime/loader fresh-create produces `viewer`, `streamer`, `agency`, `admin`. | false-drift risk | Preserve the label set. Do not create enum rebuild or reorder migration solely to normalize order. PR body for baseline must say whether it targets migrated order, fresh-create order, or explicitly accepts both. |
| `streamers.agency_user_id` FK | Runtime has `fk_streamers_agency_user_id`; migration `011_streamers_agency.sql` and runtime `applyStreamerAgencyMigration` both create it. | runtime patch migrated to Atlas | Preserved in Atlas loader and migration `020`; server startup can remove this DDL in the runtime ownership PR. |
| Auth provider soft-delete uniqueness | Migration `014_auth_provider_partial_unique.sql` and loader preserve `idx_auth_providers_provider_provider_id_active` and `idx_auth_providers_web3_user_active`; runtime fresh-create lacks them. | historical invariant not in runtime fresh-create | Keep these indexes in Atlas. They protect wallet/login rebinding semantics for soft-deleted auth providers. |
| Claim composite consistency | Migration `016_claims_composite_fk.sql` adds `claim_user_id`, composite FKs `fk_claim_items_claim_user`, `fk_claim_items_ledger_user`, `fk_claim_items_tx_ledger`, and supporting unique indexes. Runtime/loader keep `claim_user_id` but do not recreate the composite FKs/supporting unique indexes. | historical invariant not in runtime/loader | Do not drop silently. Either port these invariants into Atlas desired schema or open an issue-backed service decision to remove them. |
| Claim transaction hash uniqueness | Migration `015_claims.sql` creates partial unique `idx_claims_tx_hash_not_null`; runtime/loader do not. | historical invariant not in runtime/loader | Treat as preserve-by-default until claim service review confirms duplicate tx hashes are harmless. |
| User-owned row delete behavior | Several historical FKs use `ON DELETE CASCADE` (`auth_providers`, `shipping_addresses`, `refresh_tokens`, `claims`, `streamers`); runtime/loader GORM FKs often omit cascade. | behavior drift | Baseline must not rewrite cascade behavior without an explicit data-retention decision. |
| Timestamp nullability/defaults | Historical SQL frequently uses `NOT NULL DEFAULT now()` for `created_at` / `updated_at`; runtime/loader GORM schema leaves many timestamp columns nullable without defaults. | column drift | Baseline should avoid generating nullability/default churn until production/staging catalog confirms the actual deployed state and service-level expectations. |
| Integer width drift | `channel_configs.multiplier` and `daily_airdrop_limit` are `integer` in historical migrations and `bigint` in runtime/loader. | column drift | Choose one canonical type in baseline PR and document compatibility. Do not narrow production data without proof. |
| Constraint/index naming drift | Many semantically equivalent unique constraints/indexes have different names between historical SQL and GORM output, e.g. `users_email_key` vs `idx_users_email`, `streamers_user_id_channel_id_key` vs `idx_streamers_user_channel`, `tachi_balances_user_id_key` vs `idx_tachi_balances_user_id`. | name drift | Avoid rename-only churn unless Atlas requires it; prefer importing actual deployed names or using guarded SQL. |
| `tachi_balances.user_id` FK duplication | Runtime/loader fresh-create have both GORM-generated `fk_tachi_balances_user` and manual `fk_tachi_balances_user_id`; historical migration has a single FK under a generated name. | runtime patch overlap | Baseline must pick a canonical target or explicitly tolerate duplicate equivalent FKs while AutoMigrate still runs. |

## Current Runtime DDL In `cmd/server/main.go`

Server startup 目前仍同時負責 schema DDL 與一次性 data repair。`020_atlas_reconcile_current_schema.sql` 與 loader custom SQL 已補齊目標 schema state；下一個 runtime ownership PR 才能移除這些 schema DDL。

| Runtime Behavior | Current Code | Reconciliation Decision |
|---|---|---|
| 建立 `user_role` enum | `initializeUserRoleEnum` fresh-create 目前建立 `('viewer', 'streamer', 'agency', 'admin')`，但既有資料庫若由 `001_init.sql` 建立後再跑 `004_rbac_roles.sql`，實際 order 是 `('viewer', 'streamer', 'admin', 'agency')`；型別已存在時只會補 `agency`，不會重排。 | 必須保留 label set。Atlas reconciliation 不得把 enum order mismatch 視為必須重建 enum 的 drift；baseline 應以 production/migrated order 為準，或明確接受雙態。 |
| 執行 GORM `AutoMigrate` | `db.AutoMigrate(...)` 套用所有 models。 | `020` 已補齊 migration directory target；移除留給 runtime ownership PR。 |
| 補 `tachi_balances.user_id` FK | 手寫 `ALTER TABLE tachi_balances ADD CONSTRAINT fk_tachi_balances_user_id`。 | 已由 Atlas migration / loader 保留；後續 runtime PR 可移除。 |
| 補 coupon redemption checks | `ensureCouponRedemptionRuntimeSchema` 增加 amount/status constraints。 | 已由 Atlas migration 保留；後續 runtime PR 可移除。 |
| 補 coupon compensation index | `coupon_redemptions(status)` 的 partial index，條件為 status = `compensation-needed`。 | 已由 Atlas migration 保留；後續 runtime PR 可移除。 |
| 補 active watch session uniqueness | Partial unique index `idx_watch_sessions_active_user_channel`。 | 已由 historical migration / Atlas schema source 保留；這是 concurrency invariant，不只是效能 index。 |
| 補 points ledger uniqueness | Unique index `idx_points_ledgers_user_channel`。 | 已由 Atlas migration 保留；watch heartbeat upsert 依賴 `ON CONFLICT (user_id, channel_id)`。 |
| 補 external transaction uniqueness | Partial unique index `idx_points_transactions_external_transaction_id`。 | 已由 Atlas migration 保留；除非另開 issue 移除 idempotency guarantee，否則必須保留。 |
| 補 streamer uniqueness | Unique index `idx_streamers_user_channel`。 | 已由 Atlas migration 保留，除非有等價 constraint 取代。 |
| 執行 streamer agency migration | `applyStreamerAgencyMigration(db)`。 | 已由 migrations `010`、`011`、`020` 與 loader custom SQL 覆蓋；後續 runtime PR 可移除。 |
| Hash raffle claim tokens | 仍保留於 server startup，將 36-char token 一次性轉成 SHA-256 hex。 | 與 schema migration 分開處理。只有 production data 仍需要此 repair 時才保留。 |

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
| Production/staging catalog | This document only proves clean migration path, clean runtime path, and loader path. | The actual deployed database may match runtime, historical SQL, or a mixed state after years of AutoMigrate/runtime patches. | Before production deploy, dump staging/current schema and compare it with the selected baseline target or create an explicit baseline procedure. |
| Loader completeness | Loader preserves `fk_streamers_agency_user_id`, runtime partial indexes, claim tx hash uniqueness, and claim composite FKs. | Any future GORM-only diff could still miss invariants not represented by tags. | Keep non-GORM invariants in loader custom SQL and tests. |
| Historical invariants | Claim composite FKs, claim tx hash uniqueness, cascade behavior, and auth soft-delete indexes are not all expressible from plain GORM tags. | Atlas may generate destructive drift if these are removed from desired schema. | Migration `020` preserves the reviewed high-risk invariants; future removals need issue-backed service decisions. |
| Runtime data migration | Server startup hashes 36-char raffle claim tokens. | Schema migration work could leave unrelated data repair hidden in runtime forever. | Decide separately whether production still needs this repair; do not bury it in baseline SQL. |

## Reconciliation Procedure

以下程序記錄 2026-05-04 產生 reconciliation 文件時的驗證方式。本階段日常驗證以 Atlas loader inspect 與 migration apply 為準；「runtime AutoMigrate state」比較仍代表移除 runtime DDL 前的 server startup target。

產生任何 baseline migration 前，先完成以下程序：

1. 準備三個乾淨 PostgreSQL database，並用相同 PostgreSQL major version、role、`search_path=public` 產生 dump：

   - `MIGRATIONS_DATABASE_URL`：重播 `services/api/migrations/001-019`。
   - `AUTOMIGRATE_DATABASE_URL`：啟動目前 server，讓 runtime `AutoMigrate` 與 patches 跑完。
   - `LOADER_DATABASE_URL`：套用 Atlas GORM loader 輸出的 desired schema SQL。

2. 建立 historical migrations state。repo 目前沒有正式 migration runner；此步驟是 reconciliation 專用的手動 sequential replay。此 loop 刻意只重播 `001-019`，避免未來新增 `020` 以後的 Atlas migration 污染 historical baseline：

   ```bash
   cd services/api
   for file in migrations/00[1-9]_*.sql migrations/01[0-9]_*.sql; do
     psql "$MIGRATIONS_DATABASE_URL" -v ON_ERROR_STOP=1 -f "$file"
   done
   ```

3. Dump migration-only schema：

   ```bash
   pg_dump --schema-only --no-owner --no-privileges --schema=public "$MIGRATIONS_DATABASE_URL" > /tmp/tachigo-sql-migrations-schema.sql
   ```

4. 建立 runtime AutoMigrate state。用第二個乾淨 DB 啟動當時尚未移除 `AutoMigrate` 的 API，等待 server startup 完成後停止 process；startup 會執行 enum init、`AutoMigrate`、manual schema patches 與 runtime data repair：

   ```bash
   cd services/api
   APP_ENV=development DATABASE_URL="$AUTOMIGRATE_DATABASE_URL" go run ./cmd/server
   ```

5. Dump AutoMigrate/runtime schema：

   ```bash
   pg_dump --schema-only --no-owner --no-privileges --schema=public "$AUTOMIGRATE_DATABASE_URL" > /tmp/tachigo-automigrate-schema.sql
   ```

6. 建立 Atlas GORM loader state。先產生 loader SQL，再套到第三個乾淨 DB：

   ```bash
   cd services/api
   go run ./cmd/loader/main.go > /tmp/tachigo-gorm-loader-schema.sql
   psql "$LOADER_DATABASE_URL" -v ON_ERROR_STOP=1 -f /tmp/tachigo-gorm-loader-schema.sql
   ```

7. Dump Atlas loader schema：

   ```bash
   pg_dump --schema-only --no-owner --no-privileges --schema=public "$LOADER_DATABASE_URL" > /tmp/tachigo-loader-schema.sql
   ```

8. 比對三份 dump，並分類每個差異：

   ```bash
   diff -u /tmp/tachigo-sql-migrations-schema.sql /tmp/tachigo-automigrate-schema.sql
   diff -u /tmp/tachigo-automigrate-schema.sql /tmp/tachigo-loader-schema.sql
   diff -u /tmp/tachigo-sql-migrations-schema.sql /tmp/tachigo-loader-schema.sql
   ```

   Runtime vs loader diff 是必要步驟，因為 `fk_streamers_agency_user_id` 這類缺口只會在這組比較中浮現。

9. 需要穩定分類 table、column、constraint、index、enum drift 時，從三個 DB 匯出 catalog lists，再比較 lists 而不是只看 raw `pg_dump`：

   ```bash
   psql "$DATABASE_URL" -At -F '|' -c "SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE' ORDER BY table_name"
   psql "$DATABASE_URL" -At -F '|' -c "SELECT table_name, column_name, data_type, udt_name, COALESCE(character_maximum_length::text,''), COALESCE(numeric_precision::text,''), COALESCE(numeric_scale::text,''), is_nullable, COALESCE(column_default,'') FROM information_schema.columns WHERE table_schema='public' ORDER BY table_name, ordinal_position"
   psql "$DATABASE_URL" -At -F '|' -c "SELECT conrelid::regclass::text, conname, contype, pg_get_constraintdef(oid) FROM pg_constraint WHERE connamespace = 'public'::regnamespace ORDER BY 1, 2"
   psql "$DATABASE_URL" -At -F '|' -c "SELECT tablename, indexname, indexdef FROM pg_indexes WHERE schemaname='public' ORDER BY tablename, indexname"
   psql "$DATABASE_URL" -At -F '|' -c "SELECT t.typname, e.enumsortorder, e.enumlabel FROM pg_type t JOIN pg_enum e ON t.oid=e.enumtypid JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname='public' ORDER BY t.typname, e.enumsortorder"
   ```

10. 在新增 `020_atlas_baseline.sql` 前，先把最終決策更新回本文件。

## Baseline Strategy Decision Record

Baseline strategy is **apply-safe reconciliation**. The repo keeps `001-019` as historical migration history and adds `020_atlas_reconcile_current_schema.sql` as a guarded reconciliation migration. New databases can apply `001-020`; existing databases shaped by AutoMigrate/runtime patches should not receive a full schema replay.

| Strategy | Use When | Required Proof |
|---|---|---|
| Import baseline | 既有環境已經透過 AutoMigrate 與 runtime patches 擁有目標 schema。 | Atlas history 可標記既有狀態，不會把 unsafe `CREATE TABLE` 重套到既有 DB。 |
| Apply-safe baseline | 既有環境需要 SQL 從已知 current state reconcile。 | Migration 可同時套用到 clean migration DB 與已跑過目前 `AutoMigrate` 的 DB；本 repo 採用此策略。 |

## Completion Criteria

- [x] `001-019` 中的 high-risk table/index invariants 已被保留、明確替代，或列為後續 issue-backed service decision。
- [x] `cmd/server/main.go` 中的 schema patches 已進入 Atlas migration/loader；legacy raffle token hash 保留為非 schema runtime data repair。
- [x] Raffle/watch/broadcast tables 已有 guarded reconciliation migration，且有考慮 AutoMigrate 可能已建表。
- [x] Atlas loader output 包含或保留 GORM 無法表達的 high-risk invariants。
- [ ] `AutoMigrate` 已在 Atlas reconciliation migration 落地後從 server startup 移除。（runtime ownership PR scope）
