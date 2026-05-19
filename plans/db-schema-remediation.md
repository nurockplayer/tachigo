# DB Schema Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **Revision 2026-05-20**：補上 Opus review 指出的 blocker 修補：(1) PR 2 加歷史 CASCADE FK 保留 step 2b、(2) PR 3 加 `click` source 決策 gate、(3) PR 5 加 BIGINT 溢位 preflight 與產品 sign-off、(4) PR 6 加 claim_token preflight、(5) PR 6 case-insensitive identity 改為 audit + normalize 兩步、(6) PR 6 同步 drop 既有 case-sensitive web3 unique index。Contract 新增 External Transaction ID Rules、Watch Session Context Rules、Known Unaddressed Concerns 三節。CI drift guard 加 artifact upload；Merge Gate 加 loader dry-run 與本地 atlas diff 檢查。

**Goal:** 將 tachigo DB schema 從「migrations / loader / model / test helper 多源 drift」收斂成可審計、可驗證、可逐步強化的 canonical schema contract。

**Architecture:** 先建立 schema contract 與 drift guard，再用多個小 PR 逐步補 FK、checks、ownership uniqueness、balance 型別決策與高成長查詢 index。每個 schema PR 都必須包含 data preflight、Atlas loader preservation、migration apply 驗證與 PostgreSQL-specific tests。

**Tech Stack:** Go、GORM、PostgreSQL 16、Atlas、SQLite test helpers、GitHub Actions

**Review:** `docs/db-schema-design-review.md`

---

## 拆分原則

這項工作會同時碰 schema、service assumptions、test helpers、CI。不要做成單一 PR。

| PR | 目的 | 主要檔案 | 驗收 |
|---|---|---|---|
| PR 1 | 建立 schema contract 與 drift guard | `docs/db-schema-contract.md`、`services/api/cmd/loader/main_test.go`、`.github/workflows/ci.yml` | 能列出 canonical constraints，CI 能偵測 migration-applied schema 與 loader desired schema 的非預期差異。 |
| PR 2 | 補 referential integrity | `services/api/migrations/021_schema_fk_contract.sql`、`services/api/cmd/loader/main.go`、models/tests | 缺失 FK 不再 drift；data audit 先擋孤兒資料。 |
| PR 3 | 補 points ledger accounting checks | `services/api/migrations/022_points_accounting_checks.sql`、`services/api/cmd/loader/main.go`、points tests | 負餘額、非法 source、錯誤 watch context 不能寫入。 |
| PR 4 | 收斂 channel ownership | `services/api/migrations/023_channel_ownership_constraints.sql`、agency/streamer service tests | 同一 channel 不會對到多個 streamer 或 agency。 |
| PR 5 | 決定 `$TACHI` balance 型別 | `services/api/migrations/024_tachi_balance_units.sql`、`models/tachi_balance.go`、claim/spend tests | DB 與 Go 對 balance 單位一致，沒有 cast truncation。 |
| PR 6 | 補 raffle/coupon/auth constraints 與 indexes | migrations、loader、raffle/spend/auth tests | 狀態 enum、tx hash idempotency、case-insensitive identity、composite indexes 被 DB 保護。 |
| PR 7 | 收斂 test schema | `internal/testschema` 或 shared test helper、services/handlers/router tests | 測試 schema 不再多份 drift。 |

## PR 1：Schema Contract 與 Drift Guard

**Files:**
- Create: `docs/db-schema-contract.md`
- Modify: `services/api/cmd/loader/main_test.go`
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1：建立 contract 文件**

Create `docs/db-schema-contract.md`，內容先放以下表格，後續 PR 只允許在此表格中新增或修改已決策的 invariant：

```markdown
# DB Schema Contract

> Last reviewed: 2026-05-17

This file is the canonical schema contract for `services/api`.
Atlas migrations own runtime schema changes. GORM loader output must preserve every invariant listed here.

## Referential Integrity

| Table | Column(s) | References | Delete behavior | Owner |
|---|---|---|---|---|
| auth_providers | user_id | users(id) | ON DELETE CASCADE | auth |
| shipping_addresses | user_id | users(id) | ON DELETE CASCADE | user |
| refresh_tokens | user_id | users(id) | ON DELETE CASCADE | auth |
| email_verifications | user_id | users(id) | ON DELETE CASCADE | email auth |
| watch_sessions | user_id | users(id) | NO ACTION | watch |
| points_ledgers | user_id | users(id) | NO ACTION | points |
| points_transactions | ledger_id | points_ledgers(id) | NO ACTION | points |
| streamers | user_id | users(id) | ON DELETE CASCADE | agency/dashboard |
| streamers | agency_user_id | users(id) | NO ACTION | agency/dashboard |
| agency_streamers | agency_id | users(id) | ON DELETE CASCADE | agency/dashboard |
| watch_time_stats | user_id | users(id) | NO ACTION | points |
| broadcast_time_stats | streamer_id | users(id) | NO ACTION | points |
| broadcast_time_logs | streamer_id | users(id) | NO ACTION | points |
| claims | user_id | users(id) | ON DELETE CASCADE | claim |
| claim_items | (claim_id, claim_user_id) | claims(id, user_id) | ON DELETE CASCADE | claim |
| claim_items | (ledger_id, claim_user_id) | points_ledgers(id, user_id) | NO ACTION | claim |
| claim_items | (points_transaction_id, ledger_id) | points_transactions(id, ledger_id) | NO ACTION | claim |
| tachi_balances | user_id | users(id) | ON DELETE CASCADE | claim/spend |
| coupon_redemptions | user_id | users(id) | ON DELETE CASCADE | spend |
| raffles | user_id | users(id) | NO ACTION | raffle |
| raffle_entries | raffle_id | raffles(id) | ON DELETE CASCADE | raffle |
| raffle_entries | user_id | users(id) | ON DELETE SET NULL | raffle |
| raffle_draws | (entry_id, raffle_id) | raffle_entries(id, raffle_id) | ON DELETE CASCADE | raffle |
| raffle_claims | draw_id | raffle_draws(id) | ON DELETE CASCADE | raffle |

## Points Accounting Checks

| Table | Constraint |
|---|---|
| points_ledgers | cumulative_total >= 0 |
| points_ledgers | spendable_balance >= 0 |
| points_ledgers | spendable_balance <= cumulative_total |
| points_transactions | delta <> 0 |
| points_transactions | balance_after >= 0 |
| points_transactions | source IN ('bits','t_point','watch_time','click','spend','claim','airdrop') |
| points_transactions | source IN ('watch_time','click') requires watch_session_id IS NOT NULL; all other sources require watch_session_id IS NULL |

## External Transaction ID Rules

| Source | external_transaction_id |
|---|---|
| bits | required, globally unique (partial unique index already exists) |
| t_point / spend / claim / airdrop / watch_time / click | must be NULL |

## Watch Session Context Rules

| Source | watch_session_id | Product confirmed |
|---|---|---|
| watch_time | required | yes (see models/points.go:46) |
| click | TBD — needs handler audit before adding CHECK | no |
| bits / t_point / spend / claim / airdrop | must be NULL | partial (claim/airdrop need explicit sign-off) |

## Known Unaddressed Concerns

- Retention / partition strategy for `points_transactions`, `broadcast_time_logs`, `watch_sessions` (unbounded growth).
- `$TACHI` raw-unit representation (current decision: whole-token BIGINT; raw 78-digit option not implemented).
- UUID v7 backfill on existing rows — tracked separately in `plans/uuid-v7-migration.md`.
```

- [ ] **Step 2：在 loader test 鎖住 custom schema 必備 invariant**

Add this helper to `services/api/cmd/loader/main_test.go`:

```go
func TestAtlasLoaderContainsSchemaContractInvariants(t *testing.T) {
	stmts, err := loadAtlasSchema()
	if err != nil {
		t.Fatalf("load atlas schema: %v", err)
	}
	sql := normalizeSQL(stmts)
	for _, want := range []string{
		"FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE",
		"FOREIGN KEY (user_id) REFERENCES users(id)",
		"FOREIGN KEY (streamer_id) REFERENCES users(id)",
		"CHECK (cumulative_total >= 0)",
		"CHECK (spendable_balance >= 0)",
		"CHECK (spendable_balance <= cumulative_total)",
		"CHECK (delta <> 0)",
		"CHECK (balance_after >= 0)",
		"CHECK (source IN ('bits','t_point','watch_time','click','spend','claim','airdrop'))",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("loader SQL missing schema contract invariant %q:\n%s", want, sql)
		}
	}
}
```

Run:

```bash
rtk go test ./cmd/loader -run TestAtlasLoaderContainsSchemaContractInvariants -v
```

**預設行為：** PR 1 land 時用 `t.Skip("enable after PR 2/3")` 開頭，這樣 PR 1（contract + CI drift guard）可獨立 merge，後續 PR 2/3 把對應 invariant 補進 loader 時順手拿掉 skip。這比讓 PR 1 卡到 PR 2/3 都 ready 更安全。

```go
func TestAtlasLoaderContainsSchemaContractInvariants(t *testing.T) {
    t.Skip("enable after PR 2/3 add invariants to atlasCustomPostgresConstraints()")
    // ... existing body
}
```

- [ ] **Step 3：新增 CI drift guard**

In `.github/workflows/ci.yml`, extend the Atlas job after `Verify Atlas migrations apply to clean PostgreSQL`:

```yaml
      - name: Check migrated schema does not drift from Atlas loader
        id: drift_check
        working-directory: services/api
        env:
          MIGRATED_DATABASE_URL: postgres://postgres:postgres@localhost:5432/tachigo?sslmode=disable
        run: |
          atlas schema diff \
            --env gorm \
            --from "$MIGRATED_DATABASE_URL" \
            --to env://src \
            --dev-url docker://postgres/16/dev?search_path=public \
            --format '{{ sql . "  " }}' > /tmp/tachigo-schema-drift.sql
          if [ -s /tmp/tachigo-schema-drift.sql ]; then
            echo "::error::Schema drift detected. See uploaded schema-drift-sql artifact."
            cat /tmp/tachigo-schema-drift.sql
            exit 1
          fi

      - name: Upload schema drift artifact on failure
        if: failure() && steps.drift_check.outcome == 'failure'
        uses: actions/upload-artifact@v4
        with:
          name: schema-drift-sql
          path: /tmp/tachigo-schema-drift.sql
          if-no-files-found: ignore
```

Run:

```bash
rtk go test ./cmd/loader ./cmd/server -count=1
```

Expected: `PASS`.

## PR 2：Referential Integrity

**Files:**
- Create: `services/api/migrations/021_schema_fk_contract.sql`
- Modify: `services/api/cmd/loader/main.go`
- Modify: `services/api/cmd/loader/main_test.go`
- Modify: `services/api/internal/models/auth_provider.go` (add `constraint:OnDelete:CASCADE`)
- Modify: `services/api/internal/models/email_auth.go`
- Modify: `services/api/internal/models/points.go`
- Modify: `services/api/internal/models/streamer.go`
- Modify: `services/api/internal/models/watch_session.go`
- Modify: `services/api/internal/models/watch_stats.go`
- Modify: `services/api/internal/models/refresh_token.go` (cascade preservation)
- Modify: `services/api/internal/models/address.go` (cascade preservation)
- Modify: `services/api/internal/models/claim.go` (cascade preservation on claims.user_id)
- Modify: `services/api/internal/models/coupon_redemption.go` (cascade preservation)
- Modify: `services/api/internal/models/agency_streamer.go` (cascade preservation)

- [ ] **Step 1：新增 data preflight migration**

Create `services/api/migrations/021_schema_fk_contract.sql` with orphan checks before adding constraints:

```sql
-- Canonical referential integrity contract.

DO $$
DECLARE
    orphan_points_ledgers INTEGER;
    orphan_watch_sessions INTEGER;
    orphan_streamers INTEGER;
    orphan_email_verifications INTEGER;
    orphan_watch_time_stats INTEGER;
    orphan_broadcast_time_stats INTEGER;
    orphan_broadcast_time_logs INTEGER;
BEGIN
    SELECT COUNT(*) INTO orphan_points_ledgers
    FROM points_ledgers pl LEFT JOIN users u ON u.id = pl.user_id
    WHERE u.id IS NULL;

    SELECT COUNT(*) INTO orphan_watch_sessions
    FROM watch_sessions ws LEFT JOIN users u ON u.id = ws.user_id
    WHERE u.id IS NULL;

    SELECT COUNT(*) INTO orphan_streamers
    FROM streamers s LEFT JOIN users u ON u.id = s.user_id
    WHERE u.id IS NULL;

    SELECT COUNT(*) INTO orphan_email_verifications
    FROM email_verifications ev LEFT JOIN users u ON u.id = ev.user_id
    WHERE u.id IS NULL;

    SELECT COUNT(*) INTO orphan_watch_time_stats
    FROM watch_time_stats wts LEFT JOIN users u ON u.id = wts.user_id
    WHERE u.id IS NULL;

    SELECT COUNT(*) INTO orphan_broadcast_time_stats
    FROM broadcast_time_stats bts LEFT JOIN users u ON u.id = bts.streamer_id
    WHERE u.id IS NULL;

    SELECT COUNT(*) INTO orphan_broadcast_time_logs
    FROM broadcast_time_logs btl LEFT JOIN users u ON u.id = btl.streamer_id
    WHERE u.id IS NULL;

    IF orphan_points_ledgers > 0
       OR orphan_watch_sessions > 0
       OR orphan_streamers > 0
       OR orphan_email_verifications > 0
       OR orphan_watch_time_stats > 0
       OR orphan_broadcast_time_stats > 0
       OR orphan_broadcast_time_logs > 0 THEN
        RAISE EXCEPTION
            'migration 021 blocked: orphan rows detected (points_ledgers=%, watch_sessions=%, streamers=%, email_verifications=%, watch_time_stats=%, broadcast_time_stats=%, broadcast_time_logs=%)',
            orphan_points_ledgers,
            orphan_watch_sessions,
            orphan_streamers,
            orphan_email_verifications,
            orphan_watch_time_stats,
            orphan_broadcast_time_stats,
            orphan_broadcast_time_logs;
    END IF;
END $$;
```

- [ ] **Step 2：在同一 migration 補缺的 FK**

Append:

```sql
ALTER TABLE points_ledgers
    ADD CONSTRAINT fk_points_ledgers_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;
ALTER TABLE points_ledgers VALIDATE CONSTRAINT fk_points_ledgers_user_id;

ALTER TABLE watch_sessions
    ADD CONSTRAINT fk_watch_sessions_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;
ALTER TABLE watch_sessions VALIDATE CONSTRAINT fk_watch_sessions_user_id;

ALTER TABLE streamers
    ADD CONSTRAINT fk_streamers_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE NOT VALID;
ALTER TABLE streamers VALIDATE CONSTRAINT fk_streamers_user_id;

ALTER TABLE email_verifications
    ADD CONSTRAINT fk_email_verifications_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE NOT VALID;
ALTER TABLE email_verifications VALIDATE CONSTRAINT fk_email_verifications_user_id;

ALTER TABLE watch_time_stats
    ADD CONSTRAINT fk_watch_time_stats_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) NOT VALID;
ALTER TABLE watch_time_stats VALIDATE CONSTRAINT fk_watch_time_stats_user_id;

ALTER TABLE broadcast_time_stats
    ADD CONSTRAINT fk_broadcast_time_stats_streamer_id
    FOREIGN KEY (streamer_id) REFERENCES users(id) NOT VALID;
ALTER TABLE broadcast_time_stats VALIDATE CONSTRAINT fk_broadcast_time_stats_streamer_id;

ALTER TABLE broadcast_time_logs
    ADD CONSTRAINT fk_broadcast_time_logs_streamer_id
    FOREIGN KEY (streamer_id) REFERENCES users(id) NOT VALID;
ALTER TABLE broadcast_time_logs VALIDATE CONSTRAINT fk_broadcast_time_logs_streamer_id;
```

- [ ] **Step 2b：保留歷史 CASCADE delete behavior**

設計審查 §1 指出 `auth_providers`、`shipping_addresses`、`refresh_tokens`、`claims`、`coupon_redemptions`、`agency_streamers` 等歷史 migration 帶 `ON DELETE CASCADE`，但 loader 產生的 GORM FK 沒 cascade。若不在 PR 2 一起處理，未來 Atlas diff 會把這些 cascade 視為 drift。

先 dump 一份 staging schema 確認既有 constraint 名稱：

```bash
pg_dump --schema-only --no-owner "$STAGING_DATABASE_URL" \
  | rg "FOREIGN KEY.*REFERENCES users" > /tmp/existing-user-fks.sql
```

然後對每張表做 DROP + 重新 ADD WITH CASCADE（用 dump 結果替換實際 constraint 名稱）：

```sql
-- auth_providers
ALTER TABLE auth_providers DROP CONSTRAINT IF EXISTS fk_auth_providers_user;
ALTER TABLE auth_providers
    ADD CONSTRAINT fk_auth_providers_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE NOT VALID;
ALTER TABLE auth_providers VALIDATE CONSTRAINT fk_auth_providers_user_id;

-- refresh_tokens
ALTER TABLE refresh_tokens DROP CONSTRAINT IF EXISTS fk_refresh_tokens_user;
ALTER TABLE refresh_tokens
    ADD CONSTRAINT fk_refresh_tokens_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE NOT VALID;
ALTER TABLE refresh_tokens VALIDATE CONSTRAINT fk_refresh_tokens_user_id;

-- shipping_addresses
ALTER TABLE shipping_addresses DROP CONSTRAINT IF EXISTS fk_shipping_addresses_user;
ALTER TABLE shipping_addresses
    ADD CONSTRAINT fk_shipping_addresses_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE NOT VALID;
ALTER TABLE shipping_addresses VALIDATE CONSTRAINT fk_shipping_addresses_user_id;

-- claims (note: claim_items 已有 composite FK，這條只保 claims.user_id 本身)
ALTER TABLE claims DROP CONSTRAINT IF EXISTS fk_claims_user;
ALTER TABLE claims
    ADD CONSTRAINT fk_claims_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE NOT VALID;
ALTER TABLE claims VALIDATE CONSTRAINT fk_claims_user_id;

-- coupon_redemptions
ALTER TABLE coupon_redemptions DROP CONSTRAINT IF EXISTS fk_coupon_redemptions_user;
ALTER TABLE coupon_redemptions
    ADD CONSTRAINT fk_coupon_redemptions_user_id
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE NOT VALID;
ALTER TABLE coupon_redemptions VALIDATE CONSTRAINT fk_coupon_redemptions_user_id;

-- agency_streamers
ALTER TABLE agency_streamers DROP CONSTRAINT IF EXISTS fk_agency_streamers_agency;
ALTER TABLE agency_streamers
    ADD CONSTRAINT fk_agency_streamers_agency_id
    FOREIGN KEY (agency_id) REFERENCES users(id) ON DELETE CASCADE NOT VALID;
ALTER TABLE agency_streamers VALIDATE CONSTRAINT fk_agency_streamers_agency_id;
```

> 對應 model 端必須同步把 GORM relationship 寫成 `constraint:OnDelete:CASCADE`（如 `auth_provider.go`、`refresh_token.go`、`address.go`、`claim.go`），否則 loader desired schema 仍會輸出沒有 cascade 的 FK，CI drift guard 會擋下。

Before merge, inspect staging/current catalog. If any equivalent FK already exists under another name, replace the raw `ADD CONSTRAINT` with guarded SQL that does not create duplicates.

- [ ] **Step 3：Preserve these FKs in Atlas loader**

Add the same `ALTER TABLE ... ADD CONSTRAINT` statements to `atlasCustomPostgresConstraints()` in `services/api/cmd/loader/main.go`.

- [ ] **Step 4：Run focused tests**

```bash
rtk go test ./cmd/loader ./cmd/server -count=1
```

Expected: `PASS`.

```bash
rtk atlas migrate hash --dir file://migrations
```

Expected: `atlas.sum` updated.

## PR 3：Points Accounting Checks

**Files:**
- Create: `services/api/migrations/022_points_accounting_checks.sql`
- Modify: `services/api/cmd/loader/main.go`
- Modify: `services/api/cmd/loader/main_test.go`
- Modify: `services/api/internal/services/claim_schema_test.go`

- [ ] **Step 1：新增 preflight**

Create `services/api/migrations/022_points_accounting_checks.sql`:

```sql
DO $$
DECLARE
    bad_ledger_count INTEGER;
    bad_tx_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO bad_ledger_count
    FROM points_ledgers
    WHERE cumulative_total < 0
       OR spendable_balance < 0
       OR spendable_balance > cumulative_total;

    SELECT COUNT(*) INTO bad_tx_count
    FROM points_transactions
    WHERE delta = 0
       OR balance_after < 0
       OR source NOT IN ('bits','t_point','watch_time','click','spend','claim','airdrop')
       OR (source IN ('watch_time','click') AND watch_session_id IS NULL)
       OR (source NOT IN ('watch_time','click') AND watch_session_id IS NOT NULL);

    IF bad_ledger_count > 0 OR bad_tx_count > 0 THEN
        RAISE EXCEPTION
            'migration 022 blocked: invalid points accounting rows detected (ledgers=%, transactions=%)',
            bad_ledger_count,
            bad_tx_count;
    END IF;
END $$;
```

- [ ] **Step 2：新增 constraints**

Append:

```sql
ALTER TABLE points_ledgers
    ADD CONSTRAINT chk_points_ledgers_cumulative_total_gte_0
    CHECK (cumulative_total >= 0);

ALTER TABLE points_ledgers
    ADD CONSTRAINT chk_points_ledgers_spendable_balance_gte_0
    CHECK (spendable_balance >= 0);

ALTER TABLE points_ledgers
    ADD CONSTRAINT chk_points_ledgers_spendable_lte_cumulative
    CHECK (spendable_balance <= cumulative_total);

ALTER TABLE points_transactions
    ADD CONSTRAINT chk_points_transactions_delta_nonzero
    CHECK (delta <> 0);

ALTER TABLE points_transactions
    ADD CONSTRAINT chk_points_transactions_balance_after_gte_0
    CHECK (balance_after >= 0);

ALTER TABLE points_transactions
    ADD CONSTRAINT chk_points_transactions_source
    CHECK (source IN ('bits','t_point','watch_time','click','spend','claim','airdrop'));

ALTER TABLE points_transactions
    ADD CONSTRAINT chk_points_transactions_watch_context
    CHECK (
        (source IN ('watch_time','click') AND watch_session_id IS NOT NULL)
        OR
        (source NOT IN ('watch_time','click') AND watch_session_id IS NULL)
    );
```

> **Decision gate before adding `chk_points_transactions_watch_context`:**
> `internal/models/points.go` 的 WatchSessionID 規則註解目前只列 `watch_time`/`bits`/`t_point`/`spend`，沒涵蓋 `click`、`claim`、`airdrop`。把 `click` 歸到「必須有 watch_session_id」是這份計畫做的假設，必須先驗證：
>
> ```bash
> rg -n "TxSourceClick|TxSourceClaim|TxSourceAirdrop" services/api/internal/services services/api/internal/handlers
> ```
>
> 並對 staging / production 跑：
>
> ```sql
> SELECT source,
>        COUNT(*) AS total,
>        COUNT(*) FILTER (WHERE watch_session_id IS NULL) AS missing_session
>   FROM points_transactions
>  GROUP BY source ORDER BY source;
> ```
>
> 若任何 `click` row 缺 `watch_session_id`，先補 model comment、修 writer，再回頭加 constraint。產品 owner 必須在 PR description 標註已 sign-off `click` 的 session 規則，否則本 step 不得 merge。

- [ ] **Step 3：Preserve checks in loader**

優先用 GORM `check:` tag 在 model 上表達（參考 `models/tachi_balance.go:15`、`models/coupon_redemption.go:27,29` 的既有 pattern），這樣 Atlas loader 會自動帶出。

可以直接用 GORM tag 表達的（建議放 model）：

- `points_ledgers.cumulative_total >= 0`
- `points_ledgers.spendable_balance >= 0`
- `points_ledgers.spendable_balance <= cumulative_total`（同表跨欄，仍可用 `check:` tag）
- `points_transactions.delta <> 0`
- `points_transactions.balance_after >= 0`

只有以下兩條因 enum-like / 跨欄條件而必須放 loader `atlasCustomPostgresConstraints()`：

- `points_transactions.source IN (...)`
- `points_transactions.watch_context`（含 watch_session_id 與 source 的條件式）

- [ ] **Step 4：Add DB rejection tests**

Add tests under `services/api/internal/services/claim_schema_test.go` or a new `points_schema_test.go` that insert invalid ledgers / transactions and assert DB rejects them.

Run:

```bash
rtk go test ./internal/services -run "Points|ClaimSchema|MigrateTestDB" -count=1
```

Expected: `PASS`.

## PR 4：Channel Ownership Canonicalization

**Files:**
- Create: `services/api/migrations/023_channel_ownership_constraints.sql`
- Modify: `services/api/cmd/loader/main.go`
- Modify: `services/api/internal/services/agency_service_test.go`
- Modify: `services/api/internal/services/streamer_service_test.go`

- [ ] **Step 1：先擋現有 duplicate channel**

Create migration preflight:

```sql
DO $$
DECLARE
    duplicate_streamer_channels INTEGER;
    duplicate_agency_channels INTEGER;
BEGIN
    SELECT COUNT(*) INTO duplicate_streamer_channels
    FROM (
        SELECT channel_id
        FROM streamers
        GROUP BY channel_id
        HAVING COUNT(DISTINCT user_id) > 1
    ) duplicates;

    SELECT COUNT(*) INTO duplicate_agency_channels
    FROM (
        SELECT channel_id
        FROM agency_streamers
        GROUP BY channel_id
        HAVING COUNT(DISTINCT agency_id) > 1
    ) duplicates;

    IF duplicate_streamer_channels > 0 OR duplicate_agency_channels > 0 THEN
        RAISE EXCEPTION
            'migration 023 blocked: duplicate channel ownership detected (streamers=%, agency_streamers=%)',
            duplicate_streamer_channels,
            duplicate_agency_channels;
    END IF;
END $$;
```

- [ ] **Step 2：加唯一性**

Append:

```sql
CREATE UNIQUE INDEX idx_streamers_channel_id_unique
    ON streamers (channel_id);

CREATE UNIQUE INDEX idx_agency_streamers_channel_id_unique
    ON agency_streamers (channel_id);
```

If product decides a channel can belong to multiple agencies, do not add `idx_agency_streamers_channel_id_unique`; instead delete `agency_streamers` as canonical ownership source and make all ownership reads use `streamers.agency_user_id`.

- [ ] **Step 3：Update service tests**

Add tests proving duplicate channel insert fails at DB level and `AgencyService.ListStreamerUserIDs` no longer needs to handle impossible duplicates as normal runtime behavior.

Run:

```bash
rtk go test ./internal/services -run "Agency|Streamer" -count=1
```

Expected: `PASS`.

## PR 5：`$TACHI` Balance Representation

**Files:**
- Create: `docs/tachi-balance-units.md`
- Create: `services/api/migrations/024_tachi_balance_units.sql`
- Modify: `services/api/internal/models/tachi_balance.go`
- Modify: `services/api/internal/services/claim_service.go`
- Modify: `services/api/internal/services/spend_service.go`
- Modify: claim/spend tests

- [ ] **Step 1：做明確產品決策**

Create `docs/tachi-balance-units.md` with one of these decisions, **必須有產品 owner（@tachikochoko 或指定人）的 sign-off 簽名行**，否則本 PR 不得 merge：

```markdown
# TACHI Balance Units

Decision: `tachi_balances.balance` stores whole-token integer balances.

Rationale:
- Current claim/spend APIs accept `int64 amount`.
- Current contract adapter converts whole tokens to raw ERC-20 units with `amount * 10^18`.
- DB must not accept fractional balances while service truncates with `CAST(balance AS BIGINT)`.

Product sign-off:
- Owner: <name>
- Date: YYYY-MM-DD
- Confirmed max single-user balance ≤ 9.22 × 10^18 whole tokens (BIGINT upper bound).
```

If the decision is raw ERC-20 units instead, change the decision to `NUMERIC(78,0)` and update all service request/response semantics in a separate product PR.

- [ ] **Step 2：For whole-token decision, migrate column to BIGINT**

Create:

```sql
DO $$
DECLARE
    fractional_count INTEGER;
    overflow_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO fractional_count
    FROM tachi_balances
    WHERE balance <> trunc(balance);

    SELECT COUNT(*) INTO overflow_count
    FROM tachi_balances
    WHERE balance > 9223372036854775807
       OR balance < -9223372036854775808;

    IF fractional_count > 0 OR overflow_count > 0 THEN
        RAISE EXCEPTION
            'migration 024 blocked: invalid tachi_balances for BIGINT migration (fractional=%, overflow=%)',
            fractional_count,
            overflow_count;
    END IF;
END $$;

ALTER TABLE tachi_balances
    ALTER COLUMN balance TYPE BIGINT
    USING balance::BIGINT;
```

- [ ] **Step 3：Update model and remove cast**

In `services/api/internal/models/tachi_balance.go`, change:

```go
Balance int64 `gorm:"type:bigint;not null;default:0;check:chk_tachi_balance_gte_0,balance >= 0" json:"balance"`
```

In `loadTachiBalanceValue`, change the query to:

```go
"SELECT balance FROM tachi_balances WHERE user_id = ?"
```

Run:

```bash
rtk go test ./internal/services -run "Claim|Spend|TokenUnits" -count=1
```

Expected: `PASS`.

## PR 6：Raffle / Coupon / Auth Constraints 與 Indexes

**Files:**
- Create: `services/api/migrations/025_runtime_constraints_and_indexes.sql`
- Modify: `services/api/cmd/loader/main.go`
- Modify: raffle/spend/auth tests

- [ ] **Step 1：新增 enum-like checks**

先 preflight 確認既有資料符合即將加上的格式 / enum：

```sql
DO $$
DECLARE
    bad_raffle_status INTEGER;
    bad_raffle_source INTEGER;
    bad_claim_token INTEGER;
BEGIN
    SELECT COUNT(*) INTO bad_raffle_status
    FROM raffles WHERE status NOT IN ('draft','active','completed');

    SELECT COUNT(*) INTO bad_raffle_source
    FROM raffles WHERE source NOT IN ('csv','twitch_api');

    SELECT COUNT(*) INTO bad_claim_token
    FROM raffle_draws WHERE claim_token !~ '^[0-9a-f]{64}$';

    IF bad_raffle_status > 0 OR bad_raffle_source > 0 OR bad_claim_token > 0 THEN
        RAISE EXCEPTION
            'migration 025 blocked: invalid raffle data (bad_status=%, bad_source=%, non_sha256_token=%). repair raffle_draws.claim_token format first (see plans/raffle-token-hash-migration.md if not yet applied)',
            bad_raffle_status,
            bad_raffle_source,
            bad_claim_token;
    END IF;
END $$;
```

然後加 constraint：

```sql
ALTER TABLE raffles
    ADD CONSTRAINT chk_raffles_status
    CHECK (status IN ('draft','active','completed'));

ALTER TABLE raffles
    ADD CONSTRAINT chk_raffles_source
    CHECK (source IN ('csv','twitch_api'));

ALTER TABLE raffle_draws
    ADD CONSTRAINT chk_raffle_draws_claim_token_sha256_hex
    CHECK (claim_token ~ '^[0-9a-f]{64}$');
```

- [ ] **Step 2：補 coupon idempotency**

```sql
CREATE UNIQUE INDEX idx_coupon_redemptions_tx_hash_unique
    ON coupon_redemptions (tx_hash);
```

- [ ] **Step 3：補 case-insensitive identity indexes**

> **必須先 audit 與 normalize，否則 unique index 會直接 CREATE 失敗。**

Step 3a — audit 大小寫衝突（若任一查詢回非空，停下來另開 canonicalization PR 與產品決策衝突解法）：

```sql
-- emails with case collisions
SELECT lower(email) AS canonical, COUNT(*) AS dup_count, array_agg(id) AS user_ids
  FROM users WHERE email IS NOT NULL AND deleted_at IS NULL
 GROUP BY lower(email) HAVING COUNT(*) > 1;

-- web3 wallet collisions
SELECT provider, lower(provider_id) AS canonical, COUNT(*) AS dup_count
  FROM auth_providers
 WHERE provider = 'web3' AND deleted_at IS NULL
 GROUP BY provider, lower(provider_id) HAVING COUNT(*) > 1;
```

Step 3b — 在同一個 migration transaction 內 normalize 再加 index：

```sql
UPDATE users
   SET email = lower(email)
 WHERE email IS NOT NULL AND email <> lower(email);

UPDATE auth_providers
   SET provider_id = lower(provider_id)
 WHERE provider = 'web3' AND provider_id <> lower(provider_id);

CREATE UNIQUE INDEX idx_users_email_lower_active
    ON users (lower(email))
    WHERE email IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX idx_auth_providers_web3_provider_id_lower_active
    ON auth_providers (provider, lower(provider_id))
    WHERE provider = 'web3' AND deleted_at IS NULL;
```

Step 3c — 把 web3 從既有 case-sensitive index 排除（避免兩條 unique 指向同一邏輯 invariant）：

```sql
DROP INDEX IF EXISTS idx_auth_providers_provider_provider_id_active;

CREATE UNIQUE INDEX idx_auth_providers_provider_provider_id_active
    ON auth_providers (provider, provider_id)
    WHERE provider <> 'web3' AND deleted_at IS NULL;
```

> 同步更新 `services/api/cmd/loader/main.go:74` 既有 `idx_auth_providers_provider_provider_id_active` 的 WHERE clause，加上 `idx_users_email_lower_active` 與 `idx_auth_providers_web3_provider_id_lower_active`，否則 CI drift guard 會抓到 schema 不一致。

- [ ] **Step 4：補高成長查詢 indexes**

```sql
CREATE INDEX idx_points_transactions_ledger_created_id
    ON points_transactions (ledger_id, created_at DESC, id DESC);

CREATE INDEX idx_points_transactions_source_created_ledger
    ON points_transactions (source, created_at, ledger_id);

CREATE INDEX idx_broadcast_time_logs_streamer_channel_recorded
    ON broadcast_time_logs (streamer_id, channel_id, recorded_at);
```

Run:

```bash
rtk go test ./internal/services ./internal/handlers -count=1
```

Expected: `PASS`.

## PR 7：Test Schema Consolidation

**Files:**
- Create: `services/api/internal/testschema/sqlite.go`
- Create: `services/api/internal/testschema/postgres.go`
- Modify: `services/api/internal/services/testutil_test.go`
- Modify: `services/api/internal/handlers/testutil_test.go`
- Modify: `services/api/internal/router/router_test.go`

- [ ] **Step 1：Create shared SQLite schema helper**

Move the full `stmts := []string{...}` body from `services/api/internal/handlers/testutil_test.go` into `internal/testschema.SQLiteStatements()` without semantic DDL edits in this PR. The helper should start like this and then continue with the same table/index statements currently owned by the handler test helper:

```go
package testschema

func SQLiteStatements() []string {
	return []string{
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE,
			email TEXT UNIQUE,
			password_hash TEXT,
			avatar_url TEXT,
			role TEXT NOT NULL DEFAULT 'viewer',
			is_active INTEGER NOT NULL DEFAULT 1,
			email_verified INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME
		)`,
	}
}
```

Then update services/handlers/router tests to call `testschema.SQLiteStatements()` and delete their duplicated statement slices.

> **PR 不是「純機械搬移」**：設計審查 §9 已點出四份 schema 在 `watch_sessions.user_id` FK、`tachi_balances.balance` 型別、raffle nullable 上互有 drift。合併必選邊，必須 reviewer 確認哪一邊才是 canonical。

Add a verification step in PR description:

```bash
# 列出每份原始 schema 與新 shared schema 的差異
for src in \
  services/api/internal/services/testutil_test.go \
  services/api/internal/handlers/testutil_test.go \
  services/api/internal/router/router_test.go \
  services/api/internal/services/testutil_pg_test.go; do
  echo "=== $src ==="
  git show HEAD:"$src" | rg "CREATE TABLE|CREATE INDEX" | sort > /tmp/before.$$
  rg "CREATE TABLE|CREATE INDEX" services/api/internal/testschema/sqlite.go | sort > /tmp/after.$$
  diff /tmp/before.$$ /tmp/after.$$ || true
done
```

PR description 必須列出每處被收緊或放寬的 invariant，以及為什麼選那一邊。

- [ ] **Step 2：Keep PostgreSQL-specific constraints in PG tests**

PostgreSQL-only features that must stay in PG tests:

- partial indexes
- `NOT VALID` / `VALIDATE CONSTRAINT`
- regex check constraints
- Atlas loader/migration drift checks

Run:

```bash
rtk go test ./internal/services ./internal/handlers ./internal/router -count=1
```

Expected: `PASS`.

## Final Verification

After the last PR:

```bash
rtk go test ./cmd/loader ./cmd/server ./internal/services ./internal/handlers ./internal/router -count=1
```

Expected: `PASS`.

```bash
rtk atlas migrate lint --env gorm
```

Expected: no destructive migration warning unless the statement has an explicit reviewed `-- atlas:nolint`.

```bash
rtk atlas migrate apply --dir file://migrations --url "$ATLAS_DATABASE_URL"
```

Expected: migrations apply cleanly to a fresh PostgreSQL 16 database.

## Merge Gate

Before applying any schema PR to production:

- [ ] Dump staging/current production-like catalog for tables, columns, constraints, indexes, and enum labels.
- [ ] Confirm every data preflight query returns zero invalid rows.
- [ ] Confirm migration lock/validation strategy is acceptable for current table sizes.
- [ ] Confirm rollback path is documented in PR body.
- [ ] Confirm PR scope touches only one remediation PR area.
- [ ] Run loader dry-run and attach output to PR description for reviewer baseline:

  ```bash
  cd services/api && go run ./cmd/loader > /tmp/loader-desired.sql
  diff <(pg_dump --schema-only --no-owner "$STAGING_DATABASE_URL") /tmp/loader-desired.sql \
    | head -200 > /tmp/loader-vs-staging.diff
  ```

- [ ] Confirm CI drift guard (`atlas schema diff`) passes locally before push:

  ```bash
  atlas schema diff --env gorm \
    --from "$LOCAL_MIGRATED_DATABASE_URL" \
    --to env://src \
    --dev-url docker://postgres/16/dev?search_path=public
  ```
  Expected: empty diff.
