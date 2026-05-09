# Atlas Migration Implementation Plan

> **給 agentic workers：** 執行本計劃時必須使用 `superpowers:subagent-driven-development` 或 `superpowers:executing-plans`，逐項完成 checkbox (`- [ ]`) 步驟。

**目標：** 將 `services/api` 的 schema 管理由 GORM `AutoMigrate` 遷移到 Atlas，同時不削弱既有資料庫 invariant，也不送出不安全的 baseline。

**架構：** Atlas 遷移必須拆成多個小型、可獨立 review 的 PR。先導入 tooling 與 CI validation，再盤點現有 schema source，接著決定並驗證 baseline 策略，最後在 staging 驗證後才移除 runtime `AutoMigrate` 與手寫 schema patches。

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

## PR 拆分順序

### PR 1：Atlas Tooling Only

**目的：** 加入最小 Atlas toolchain，讓開發者可以從 GORM schema source 產生 diff，CI 可以 lint migration 檔。

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
- [ ] 新增 CI job，對 PR 變更的 migration files 執行 Atlas migration lint。
- [ ] 確認 CI lint 是 non-destructive，不會對共用 database apply migration。

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

**Baseline Decision Required Before Coding：**

開始寫 SQL 前，必須在 PR body 選定且記錄其中一種策略：

- **Import baseline：** 將 `001-019` 視為既有 history，避免把 create-table SQL 重套到已經有表的 database。
- **Apply-safe baseline：** 產生可安全套用到已知 staging/current schema 的 SQL，必要時使用 `IF NOT EXISTS` 或 guarded `DO $$` blocks。

**Implementation Steps：**

- [ ] 根據已 review 的 reconciliation 文件產生候選 baseline。
- [ ] 驗證 baseline 可套到 empty database。
- [ ] 驗證 baseline 可套到已跑過目前 server `AutoMigrate` 的 database。
- [ ] 驗證 baseline 保留 GORM tags 無法表達的 partial indexes 與 constraints。
- [ ] 最終 SQL review 完成後，才重新產生 `atlas.sum`。

**Verification：**

```bash
cd services/api
atlas migrate lint --dev-url "postgres://postgres:postgres@localhost:5432/atlas_dev?sslmode=disable" --dir "file://migrations" --latest 1
docker compose run --no-deps --rm app go test ./...
```

Expected：Atlas lint 通過、後端測試通過，且 PR body 明確寫出採用哪個 baseline strategy。

**Commit Message：**

```text
chore: add atlas baseline migration

refs #463

Co-Authored-By: Codex <codex[bot]@openai.com>
```

### PR 4：Remove Runtime AutoMigrate After Staging Validation

**目的：** 讓 Atlas 成為 runtime schema owner。

**Files：**

- Modify：`services/api/cmd/server/main.go`
- Modify or add：`services/api/cmd/server` 相關測試，若現有 coverage 無法驗證 startup migration 行為
- Modify：`docs/atlas-schema-reconciliation.md`

**Preconditions：**

- PR 1、PR 2、PR 3 已 merge。
- Staging 已驗證 baseline strategy。
- PR body 必須列出移除哪些 runtime patches，以及等價 Atlas migration 在哪裡。

**Implementation Steps：**

- [ ] 從 server startup 移除 `db.AutoMigrate(...)`。
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

## Issue #463 Acceptance Checklist

- [ ] `services/api/atlas.hcl` 存在，且 `atlas migrate diff` 可使用 GORM loader。
- [ ] CI 有 Atlas migration lint，且 job pass。
- [ ] `services/api/migrations/001-019` 與 GORM model drift 已記錄或修正。
- [ ] Baseline strategy 明確，且已針對目標 schema state 驗證。
- [ ] `AutoMigrate` 策略在移除前已明確化，且 removal 只在 baseline validation 後執行。

## Atlas References

- External schema loading：<https://atlasgo.io/atlas-schema/external>
- Versioned migration diff：<https://atlasgo.io/versioned/diff>
- Migration lint：<https://atlasgo.io/versioned/lint>

## Review Checklist

- [ ] PR diff 沒超過專案 scope limit；若超過，PR body 有有效 scope exception 理由。
- [ ] PR 沒混合 schema migration、service logic、handler/router、frontend changes。
- [ ] Partial unique indexes 被保留，或有 issue-backed decision 說明為何移除。
- [ ] Custom PostgreSQL enum definitions 保留相同 label set，並依 `docs/atlas-schema-reconciliation.md` 的 enum ordering decision 處理；不得只因 runtime fresh-create order 與 production/migrated order 不同，就產生 enum rebuild 或 false-drift migration。
- [ ] `go mod tidy` 不會移除 Atlas tooling dependencies。
- [ ] CI lint 驗證 migration directory；本地驗證另行驗證 GORM loader path。
