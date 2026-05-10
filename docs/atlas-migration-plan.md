# Atlas Migration Implementation Plan

> **給 agentic workers：** 執行本計劃時必須使用 `superpowers:subagent-driven-development` 或 `superpowers:executing-plans`，逐項完成 checkbox (`- [ ]`) 步驟。

**目標：** 將 `services/api` 的 schema 管理由 GORM `AutoMigrate` 遷移到 Atlas，同時不削弱既有資料庫 invariant，也不送出不安全的 baseline。

**架構：** Atlas 遷移原本規劃拆成多個小型、可獨立 review 的 PR：先導入 tooling 與 CI validation，再盤點現有 schema source，接著決定並驗證 baseline 策略，最後移除 runtime `AutoMigrate` 與手寫 schema patches。2026-05-10 後續決策改為由 PR `#588` 一次交付完整 runtime migration path，因為專案尚未正式上線，保留半套 migration ownership 反而會增加風險。

**Tech Stack：** Go、GORM、PostgreSQL、Atlas、`atlas-provider-gorm`、GitHub Actions。

---

## Source Of Truth

- GitHub issue：`#463` `[backend] Migration 工具遷移：GORM AutoMigrate → Atlas`
- 前置討論：`#214`
- Migration 目錄：`services/api/migrations/`
- Atlas 設定檔：`services/api/atlas.hcl`
- 目前 runtime schema 入口：`services/api/cmd/server/main.go`
- 目前 GORM models：`services/api/internal/models/`

本文件只是執行計劃，不代表 `#463` 已完成，也不得把文件本身當成 implementation source of truth。

## Guardrails

- 不得把 tooling、baseline SQL、`AutoMigrate` removal 合併在同一個 PR。
- `docs/atlas-schema-reconciliation.md` review 前，不得產生大型 baseline migration。
- 不得因為 `atlas migrate lint` 通過就宣稱 baseline 安全；lint 驗證 migration mechanics，不驗證 production data compatibility。
- 不得移除 GORM model structs；`atlas-provider-gorm` 仍需要讀取它們。
- 不得把 production deploy automation 或 rollback automation 混進 `#463`；issue 已明確排除。
- 任何改變 runtime schema 行為的 PR，都必須在 PR body 補 staging 驗證說明。
- CI 使用 official/latest Atlas CLI 驗證 `external_schema` / GORM loader，並對 ephemeral PostgreSQL 執行完整 `atlas migrate apply`；不再使用 `migrate lint` 或 Community binary。

## PR 拆分順序

### PR 1：Atlas Tooling Only

**目的：** 加入最小 Atlas toolchain，讓開發者可以從 GORM schema source 產生 diff，CI 可以驗證 loader 與 migration directory。

**Files：**

- Create：`services/api/atlas.hcl`
- Create：`services/api/cmd/loader/main.go`
- Create：`services/api/tools.go`
- Modify：`services/api/go.mod`
- Modify：`services/api/go.sum`
- Modify：`.github/workflows/ci.yml`
- Optional：`.gitattributes`，僅在這顆 PR 開始導入 Atlas checksum 檔時需要

**明確不做：**

- 不新增 `020_atlas_baseline.sql`
- 不新增 `atlas.sum`，除非這顆 PR 明確開始讓 Atlas 管理 migration directory checksum
- 不移除 `AutoMigrate`
- 不清理 model tags，除非是 loader compilation 的必要前置

**Implementation Steps：**

- [ ] 新增 `services/api/tools.go`，使用 `//go:build tools`，並以 blank import 錨定 `ariga.io/atlas-provider-gorm/gormschema` 或所選 provider 版本需要的 package path。
- [ ] 新增 `services/api/cmd/loader/main.go`，載入目前所有 GORM model structs。
- [ ] 將 GORM 無法表達的 custom PostgreSQL schema objects 放進 loader output，或集中在清楚命名的 loader helper。
- [ ] 新增 `services/api/atlas.hcl`，使用 `external_schema` program 執行 loader。
- [ ] 新增 CI job，驗證 Atlas 可以 inspect GORM loader，並把 migration directory 套到 clean PostgreSQL。
- [ ] 確認 CI migration apply 只寫入 job 內 ephemeral database，不會對共用 database apply migration。

**Verification：**

```bash
cd services/api
go mod tidy
go run ./cmd/loader/main.go > /tmp/tachigo-gorm-schema.sql
atlas schema inspect --env gorm --url env://src --format '{{ sql . }}' > /tmp/tachigo-atlas-inspect-schema.sql
```

Expected：loader 可編譯、可輸出 PostgreSQL DDL，且 Atlas 可用 `env://src` 讀取 GORM schema source；這個 smoke test 不應寫入 migration file 或 `atlas.sum`。

**Commit Message：**

```text
chore: add atlas migration tooling

refs #463

Co-Authored-By: Codex <codex[bot]@openai.com>
```

### PR 2：Schema Reconciliation

**目的：** 在撰寫 baseline SQL 前，先把隱性的 schema history 變成可 review 的 reconciliation 文件。

**Files：**

- Modify：`docs/atlas-schema-reconciliation.md`

**Implementation Steps：**

- [ ] 將 `services/api/migrations/001-019` 依序 apply 到乾淨 PostgreSQL dev database。
- [ ] 用另一個乾淨 PostgreSQL dev database 啟動目前 server，讓 `AutoMigrate` 與 runtime patches 跑完。
- [ ] 用 `pg_dump --schema-only --no-owner --no-privileges` dump 兩邊 schema。
- [ ] 用 Atlas 或文字 diff 比對兩個 schema state。
- [ ] 將每個差異填進 reconciliation table，並寫出明確決策。
- [ ] 每個差異都標成 `preserve`、`drop after review`、`model drift`、`runtime patch`、`data migration` 或 `out of scope`。

**Verification：**

```bash
cd services/api
docker compose run --no-deps --rm app go test ./...
```

Expected：文件 PR 不改 runtime，後端測試應通過；若本機 Docker 不可用，PR body 必須明確說明未跑原因。

**Commit Message：**

```text
docs: reconcile atlas migration baseline inputs

refs #463

Co-Authored-By: Codex <codex[bot]@openai.com>
```

### PR 3：Baseline Strategy And Migration Directory Ownership

**目的：** 建立第一個 Atlas-owned migration state，同時避免破壞既有環境。

**Files：**

- Create or modify：`services/api/migrations/020_atlas_baseline.sql`
- Create or modify：`services/api/migrations/atlas.sum`
- Modify：`docs/atlas-schema-reconciliation.md`

**Baseline Decision：**

採 **Apply-safe reconciliation**。`001-019` 繼續作為歷史 migration，`020_atlas_reconcile_current_schema.sql` 補齊 GORM AutoMigrate/runtime patch 曾經隱式建立的 current schema，並使用 `IF NOT EXISTS` / guarded `DO $$` blocks 避免對既有 DB 重打完整 schema。

**Implementation Steps：**

- [ ] 根據已 review 的 reconciliation 文件產生候選 baseline。
- [ ] 驗證 baseline 可套到 empty database。
- [ ] 驗證 baseline 可套到已跑過目前 server `AutoMigrate` 的 database。
- [ ] 驗證 baseline 保留 GORM tags 無法表達的 partial indexes 與 constraints。
- [ ] 最終 SQL review 完成後，才重新產生 `atlas.sum`。

**Verification：**

```bash
cd services/api
atlas schema inspect --env gorm --url env://src --format '{{ sql . }}' > /tmp/tachigo-atlas-inspect-schema.sql
atlas migrate apply --dir "file://migrations" --url "$ATLAS_VERIFY_DATABASE_URL"
docker compose run --no-deps --rm app go test ./...
```

Expected：Atlas 可以 inspect loader、完整 migration directory 可套到乾淨 PostgreSQL、後端測試通過，且 PR body 明確寫出採用哪個 baseline strategy。

**Commit Message：**

```text
chore: add atlas baseline migration

refs #463

Co-Authored-By: Codex <codex[bot]@openai.com>
```

### PR 4：Remove Runtime AutoMigrate After Reconciliation Migration（歷史計畫；已由 PR `#588` 完成）

**目的：** 讓 Atlas 成為 runtime schema owner。

**Files：**

- Modify：`services/api/cmd/server/main.go`
- Modify or add：`services/api/cmd/server` 相關測試，若現有 coverage 無法驗證 startup migration 行為
- Modify：`docs/atlas-schema-reconciliation.md`

**Preconditions：**

- PR 1、PR 2、PR 3 已 merge。
- Reconciliation migration 已保留 runtime schema patches 與 high-risk historical invariants。
- PR body 必須列出移除哪些 runtime patches，以及等價 Atlas migration 在哪裡。

**Implementation Steps：**

- [x] 從 server startup 移除 `db.AutoMigrate(...)`（PR `#588`）。
- [ ] 移除已由 Atlas migration 表達的 manual schema patches。
- [ ] 只在仍有必要且有文件說明時，保留非 schema 的 runtime data repair code。
- [ ] 新增 startup guard 或 log，明確表示 API process 不再執行 schema DDL。

**Verification：**

```bash
cd services/api
docker compose run --no-deps --rm app go test ./...
```

Expected：後端測試通過，且 server startup 不再執行 schema DDL。

**Commit Message：**

```text
chore: remove gorm automigrate from server startup

refs #463

Co-Authored-By: Codex <codex[bot]@openai.com>
```

## CI 策略決策（2026-05-10）

**決策：移除 `migrate lint`，改用 official/latest Atlas + ephemeral Postgres apply。**

背景：`atlas migrate lint` 從 v0.38 起限 Pro 授權。原本 pin 在 v0.37.0 是為了繼續免費用 lint。經 Claude Code + Codex 討論，新專案不值得為此維護版本 pin。

### 決定採用的 CI 方案

| 步驟 | 做法 | 理由 |
|---|---|---|
| Atlas 版本 | official/latest，不 pin | 解除版本鎖，不依賴 Community binary |
| GORM loader 驗證 | `atlas schema inspect --env gorm --url env://src` | 確認 external_schema + loader 正常運作 |
| Migration apply | `atlas migrate apply --dir file://migrations --url postgres://...` 對 ephemeral Postgres | 比 lint 更接近真實；apply 本身會驗 atlas.sum checksum |
| Checksum drift | apply 失敗即可抓到，不另跑 `hash --check`（該 flag 不存在） | apply 已內建此保護 |
| Destructive DDL guard | CI grep `DROP TABLE`、`TRUNCATE TABLE`、`DROP COLUMN`，需 `-- atlas:nolint` allowlist 才過 | 取代 lint 對危險 DDL 的部分保護，且更透明 |

**明確不做：**
- 不使用 `migrate lint`（Pro gate）
- 不使用 Community binary（不支援 `external_schema`）
- 不 pin 舊版 official binary

### 已實作項目

- [x] `ci.yml` 的 `atlas-migration-tooling` job 改成 apply 方案（移除舊版 pin、Community binary 與 lint step）
- [x] 加 ephemeral Postgres service 給 Atlas apply job 用
- [x] Docker image 的 dev/runtime stages 都透過可覆寫的 `ATLAS_VERSION=1.2.0` Atlas stage 帶入版本化 CLI，entrypoint 會先執行 `atlas migrate apply`，`docker-compose.yml` 提供 `ATLAS_DATABASE_URL`
- [x] `services/api/Makefile` 加入 `make migrate`，提供本機 CLI apply path
- [x] 加 destructive DDL grep guard；需要同行或前一行 `-- atlas:nolint` 才允許 destructive statement

### 運作邊界

- Fresh DB / 新 Docker volume：API Docker entrypoint 會先套用 `001-020`，再啟動 `air`（dev target）或 `/tachigo`（runtime target）；API binary 本身不再執行 schema DDL。
- 既有 dev DB / 舊 volume：若 schema 是早期 `AutoMigrate` 建出來、但沒有 `atlas_schema_revisions`，Atlas 不能自動判斷已套用哪些歷史 migration；本專案尚未正式上線，建議 reset dev volume 或由 operator 手動 baseline。
- Production deploy automation 仍不屬於 `#463` 範圍；上線前需要在部署 workflow/runbook 中重用同一個 `atlas migrate apply` 流程。

## Issue #463 Acceptance Checklist

- [x] `services/api/atlas.hcl` 存在，且 `atlas migrate diff` 可使用 GORM loader。
- [x] CI 有 Atlas migration 驗證，且 job 被 backend gate 納入。
- [x] `services/api/migrations/001-019` 與 GORM model drift 已記錄或修正。
- [x] Baseline strategy 明確，採 guarded apply-safe reconciliation migration。
- [x] `AutoMigrate` 已從 server startup 移除；server 不再執行 schema DDL。
- [x] Migration runner 存在：Docker image entrypoint 與 `make migrate` 都有 `atlas migrate apply` path，確保 fresh DB / 新 volume 可正確 bootstrap schema。
- [x] CI 改成 official/latest Atlas + ephemeral Postgres apply（移除舊版 pin、Community binary 與 migrate lint）。

> **2026-05-10 狀態**：working tree 已補上 CI apply guardrail、Docker entrypoint migration runner 與 `make migrate`。這解除 fresh DB / 新 volume 的啟動 blocker；既有舊 dev volume 若沒有 Atlas revision history，仍需 reset 或手動 baseline，不能假裝 Atlas 會自動接管任意歷史狀態。

## Atlas References

- External schema loading：<https://atlasgo.io/atlas-schema/external>
- Versioned migration diff：<https://atlasgo.io/versioned/diff>

## Review Checklist

- [ ] PR diff 沒超過專案 scope limit；若超過，PR body 有有效 scope exception 理由。
- [ ] PR 沒混合 schema migration、service logic、handler/router、frontend changes。
- [ ] Partial unique indexes 被保留，或有 issue-backed decision 說明為何移除。
- [ ] Custom PostgreSQL enum definitions 保留相同 label set，並依 `docs/atlas-schema-reconciliation.md` 的 enum ordering decision 處理。
- [ ] `go mod tidy` 不會移除 Atlas tooling dependencies。
- [ ] CI apply job 對 ephemeral Postgres 跑完整 migration 001–020 無錯誤。
