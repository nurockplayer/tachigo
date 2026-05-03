# GitHub Actions 補強路線

> **狀態：** proposal / implementation plan draft
> **最後更新：** 2026-05-04

## 背景

repo 目前已有 9 個 `.github/workflows` 檔案，其中 8 個是 GitHub Actions workflow，1 個是 workflow regression test：

| 檔案 | 定位 |
|---|---|
| `.github/workflows/ci.yml` | 主要 CI、scope gate、backend/frontend/dashboard/contracts 測試、workflow regression、commit message regression |
| `.github/workflows/ci.test.mjs` | `ci.yml` / scope policy / auto-merge 等 workflow 行為的 Node regression test |
| `.github/workflows/pr-scope-police.yml` | PR metadata、diff 大小、product surface、dependency gate 檢查 |
| `.github/workflows/codex-review-flag.yml` | 管理 `needs-codex-review` / `changes-requested` label |
| `.github/workflows/codex-review-slack.yml` | CI 成功且 PR 有 `needs-codex-review` label 時通知 Slack review queue |
| `.github/workflows/auto-merge.yml` | 非 Dependabot PR opened/reopened/ready_for_review 時啟用 GitHub auto-merge |
| `.github/workflows/dependabot-automerge.yml` | Dependabot PR 的保守 auto-merge policy |
| `.github/workflows/notify-rebase-needed.yml` | PR merge 後通知其他有 conflict 的 PR rebase |
| `.github/workflows/weekly-release-pr.yml` | 每週或手動建立 `develop -> main` release PR |

這套系統已經涵蓋「PR 治理」與「review/merge 流程」的大部分骨架。下一步不應再補一堆泛用 workflow，而是針對目前缺口補高訊號自動化。

## 目前已覆蓋

- PR scope gate：title prefix、issue/reference、source of truth、dependency、non-goals、diff 大小、product surface。
- Heavy CI gate：scope 不合格或 dependency blocked 時不浪費 backend/frontend/dashboard 重型 CI。
- Backend CI：Docker image build、Go test、Go vet、integration tests。
- Frontend CI：extension / dashboard docker build、lint、test、build、extension i18n check。
- Contract CI：Foundry install、build、test、format check。
- Workflow regression：用 `.github/workflows/ci.test.mjs` 固定 workflow policy 不被誤改。
- Commit message 檢查：`ci.yml` 已有 `commit-message-regression` 與 `pr-commit-messages` job。
- Review label lifecycle：PR opened/synchronize/ready_for_review、Codex review submitted/dismissed、scheduled fallback。
- Auto-merge：一般 PR 與 Dependabot 分流處理。
- Release PR：定期從 `develop` promote 到 `main`。

## 發現的文件 / 設定 drift

### Dependabot Go module 路徑

PR A 修正前，`.github/dependabot.yml` 與 `docs/dependabot-update-policy.md` 還寫 Go module directory 為 `/backend`，但 repo 現況是：

- Go module：`services/api/go.mod`
- 舊路徑 `backend/` 不存在

這會讓 Go Dependabot update 無法正確掃描。這不是新增 workflow，而是應先修正的既有 automation drift。PR A 完成後，這段可視為已處理的背景紀錄。

建議先開一個小 PR：

- `.github/dependabot.yml`：把 gomod directory 從 `/backend` 改成 `/services/api`
- `docs/dependabot-update-policy.md`：同步把 `/backend` 改成 `/services/api`

## 推薦實作順序

### 1. 修正 Dependabot Go directory drift

**優先級：P0**

目的：恢復 Go module dependency update。

修改範圍：

- `.github/dependabot.yml`
- `docs/dependabot-update-policy.md`

驗收方式：

- `dependabot.yml` syntax 仍合法。
- 文件中的 Go module path 與 `services/api/go.mod` 一致。
- 下次 Dependabot schedule 能針對 Go module 建 PR。

本項不需要新增 workflow。

### 2. Draft PR auto-ready

**優先級：P1**

目的：讓 Codex task PR 可以先用 draft 發出跑 CI，CI 綠後自動轉 ready for review，再交給既有 `needs-codex-review` 流程。

Repo 內既有設計文件（目前存在於 `develop`；若後續搬移或改名，PR B 必須同步更新本 reference）：

- `docs/draft-pr-auto-ready.md`

建議新增：

- `.github/workflows/auto-ready-pr.yml`

觸發：

- `pull_request`：`opened` / `synchronize` / `reopened` / `labeled` / `edited`
- `workflow_run`：監聽 required CI workflow completed
- `schedule`：fallback，避免外部 status 或漏事件造成 draft 卡住
- `workflow_dispatch`：人工補跑

核心規則：

- 只處理 base 為 `develop` / `main` 的同 repo PR。
- 只處理 draft PR。
- 只處理有 `auto-ready` label 的 PR。
- 排除 `dependabot[bot]` / `renovate[bot]`。
- 重新查 live PR head SHA，不信任 stale event payload。
- 以 workflow 內維護的 `required_checks` 清單作為 readiness gate。
- 不讀 fork 產物、不 checkout PR head。
- mark ready 前再次驗證 PR number、head SHA、base branch、draft state、label state、author deny-list。

`required_checks` 初版必須在 PR B 內明確寫入，不留給實作時臨場判斷。以下清單以 2026-05-04 重新查詢 GitHub branch protection API 的 live required contexts 為準；實作 PR 前仍必須再用 GitHub UI、`gh pr checks`，或 branch protection API 驗證 exact context name，若 GitHub 顯示含 workflow prefix，需以實際顯示字串為準：

| Branch | Required check contexts |
|---|---|
| `develop` | `Scope gate` |
| `develop` | `Backend CI (gate)` |
| `develop` | `Frontend build` |
| `develop` | `Dashboard build` |
| `develop` | `Contracts build` |
| `main` | `Scope police` |

不要把 `Workflow regression tests`、`Commit message regression tests`、`PR commit messages`、`Check backend CI cache wiring`、`Notify Discord on failure`、auto-ready 自己的 check、或非 required 的外部 review/status 放進第一版 readiness gate。CodeRabbit 只有在 branch protection / ruleset 也設成 required 時，才可同步加入。

驗收方式：

- draft PR + `auto-ready` label + CI 全綠後自動 ready。
- CI 失敗、pending、cancelled、stale SHA 時不 ready。
- 無 label 的 draft PR 保持 draft。
- Dependabot / Renovate PR 不受影響。

### 3. Workflow health check

**優先級：P1**

目的：保護越來越複雜的 GitHub Actions 本身，避免 YAML、permissions、trigger 或 policy regression 靜悄悄壞掉。

建議新增：

- `.github/workflows/workflow-health.yml`

觸發：

- `pull_request` paths:
  - `.github/workflows/**`
  - `.github/dependabot.yml`
  - `infra/scripts/check-*.sh`
  - `infra/scripts/*commit*.sh`
- `workflow_dispatch`

建議 job：

| Job | 目的 |
|---|---|
| `actionlint` | 檢查 workflow YAML、expression、shell 常見錯誤 |
| `workflow-regression` | 跑 `node --test .github/workflows/ci.test.mjs`，可從 `ci.yml` 抽出或保留雙跑 |
| `dependabot-config` | parse `.github/dependabot.yml`，並檢查 configured directory 真的存在 |
| `permission-sanity` | 檢查高權限 workflow 是否有明確用途與 path/actor guard |

`permission-sanity` 第一版採 allowlist，不做模糊判斷。凡 workflow 宣告 `contents: write`、`pull-requests: write`、`issues: write`、`actions: write`，都必須在 allowlist 中列出用途與 guard；新增高權限 workflow 但未更新 allowlist 時直接 fail。

| Workflow | Write permission | 必要 guard |
|---|---|---|
| `pr-scope-police.yml` | `pull-requests: write` / `issues: write` | 只跑 `pull_request` 到 `main` / `develop`，sticky comment、label、嚴重違規 close 都必須基於同一份 scope evaluation |
| `codex-review-flag.yml` | `pull-requests: write` / `issues: write` | 只處理 `main` / `develop` PR；review event 只接受指定 reviewer；draft PR 不加 review label |
| `codex-review-slack.yml` | `actions: write` | 只在 CI success 或 `needs-codex-review` label 後通知 Slack；必須驗證 PR open、非 draft、base branch、label、head SHA、dedup cache |
| `auto-merge.yml` | `contents: write` / `pull-requests: write` | 排除 draft 與 Dependabot；只啟用 GitHub auto-merge，不直接 merge |
| `dependabot-automerge.yml` | `contents: write` / `pull-requests: write` | 只允許 `dependabot[bot]` actor；必須通過 metadata policy 或 `safe-to-automerge` label |
| `notify-rebase-needed.yml` | `pull-requests: write` / `issues: write` | 只在 merge 到 `develop` 後留言；不得修改 PR branch |
| `weekly-release-pr.yml` | `pull-requests: write` / `actions: write` | 只建立 `develop -> main` release PR；若已存在 open release PR 就 no-op |
| `auto-ready-pr.yml`（未來） | `pull-requests: write` / `checks: read` / `statuses: read` | 只處理同 repo draft PR、`auto-ready` label、非 dependency bot、live head SHA 與 required checks 全部成功 |

明確禁止第一版 workflow health check 接受未審查的 `pull_request_target`、repo-wide `contents: write`，或沒有 base branch / actor / label guard 的 public mutation workflow。

驗收方式：

- 修改 workflow 時 health check 必跑。
- 故意把 Dependabot directory 改成不存在路徑時會 fail。
- 故意放錯 workflow YAML expression 時會 fail。
- 新增高權限 workflow 但未更新 permission allowlist 時會 fail。

### 4. Security / supply-chain gate

**優先級：P2**

目的：補足 Web3 / auth / secret 邊界的安全訊號。這類掃描容易吵，建議先以 scheduled + annotation/report 開始，再決定哪些項目變 required。

候選 workflow：

- `.github/workflows/security-scan.yml`

建議分階段：

| 階段 | 掃描 | 行為 |
|---|---|---|
| 1 | `govulncheck ./...` | PR + schedule；高信心 Go vulnerability 直接 fail |
| 2 | secret scan | PR + schedule；先 annotation，不自動 close |
| 3 | dependency review | PR；只擋 known vulnerable production dependency |
| 4 | CodeQL | schedule + push；視訊號品質再納入 required checks |

注意事項：

- 不要把測試用 fake secret 當成 blocker；需設定 allowlist。
- Solidity / Foundry 的安全掃描另開 dedicated issue，不混進第一版。
- 若 secret scan 會讀完整 git history，需確認 runtime 成本與 false positive。

驗收方式：

- `services/api` 執行 `govulncheck ./...` 成功。
- 測試用 secret allowlist 不讓 CI 長期紅燈。
- 真實 private key / JWT secret pattern 能被掃出。

### 5. Migration guard

**優先級：P2**

目的：把 DB migration high blocker 的人工審查重點前移到 PR check，尤其是 rewards / balances 類表格的 double credit、precision、transaction/race 風險。

候選實作：

- 在 `pr-scope-police.yml` 增加 migration-specific comment section；或
- 新增 `.github/workflows/migration-guard.yml`，只在 `services/api/migrations/**/*.sql` 變更時觸發。

建議第一版只做 heuristic guard，不做 SQL parser：

| 檢查 | 行為 |
|---|---|
| `DROP TABLE` / `DROP COLUMN` / `ALTER COLUMN TYPE` / `RENAME` | fail，要求 reviewer 確認破壞性變更與 rollout |
| `ADD COLUMN ... NOT NULL` 無 `DEFAULT` | fail，要求 backfill/default 策略 |
| rewards / balances / points 相關 migration | sticky comment 提醒 idempotency、precision、row-level lock、transaction |
| 沒有 rollback / deploy order note | comment，不先 fail |

破壞性 migration 需要 escape hatch，但不能默默繞過。第一版使用 maintainer-only label：

- Label：`migration-exception`
- 作用：只把 destructive heuristic 從 fail 降為 warning；仍保留 sticky comment 與 reviewer checklist。
- 使用條件：PR body 必須明確寫出 destructive 變更原因、deploy order、rollback / recovery path、backfill/default 策略、lock duration 評估。
- 禁止用途：不得用來跳過 unrelated scope、測試失敗、或 rewards / balances / points 的 double credit / precision / race condition 提醒。

`migration-exception` 只能由 maintainer 手動加。workflow 不需也不應嘗試判斷 GitHub label 是誰加的；操作規則寫在文件與 PR review policy，workflow 只根據 label 進行降級。

驗收方式：

- additive migration 通過。
- destructive migration 會 fail 並輸出具體原因。
- destructive migration 加 `migration-exception` 後降為 warning，且 sticky comment 仍列出 rollout checklist。
- rewards/balances 相關 migration 會出現審查提醒。

### 6. Release PR changelog enhancement

**優先級：P3**

目的：讓 `weekly-release-pr.yml` 產生的 release PR 更容易 review，不只顯示 commits ahead。

建議改動：

- 在建立 release PR body 時列出 `main..develop` 的 merged PR。
- 依 title prefix 分組：
  - `[backend]`
  - `[frontend]`
  - `[contract]`
  - `[infra]`
  - `[chore]`
  - `[discussion]`
- 顯示 PR number、title、author、merge commit。
- 若有 `changes-requested` / `scope-violation` label 的 open PR，不阻擋 release，但在 release body 加 warning。

驗收方式：

- release PR body 可直接看出本週 release 包含哪些 PR。
- 若 `develop` 沒有比 `main` ahead，維持 no-op。
- 若已有 open release PR，維持 no-op。

## 暫不建議

| 項目 | 原因 |
|---|---|
| 另做 commit-lint workflow | `ci.yml` 已有 `pr-commit-messages` 與 regression test，先避免重複 check |
| 大型 stale auto-close | 對 active PR / issue 容易誤傷，且不是目前最大痛點 |
| PR size label | Scope Police 已有 diff threshold 與 sticky snapshot，先不增加 label noise |
| CHANGELOG 全自動產生 | release PR changelog enhancement 更貼近目前流程 |
| 自動修改 branch protection / ruleset | 權限高、blast radius 大，先維持文件 + workflow config 同步 |

## 建議拆 PR

### PR A：Dependabot directory drift

目的：恢復 Go Dependabot。

修改：

- `.github/dependabot.yml`
- `docs/dependabot-update-policy.md`

驗收：

- YAML 可解析。
- directory 指向 `services/api`。

### PR B：Auto-ready workflow

目的：實作 `docs/draft-pr-auto-ready.md` 已定義的 draft PR auto-ready。

修改：

- `.github/workflows/auto-ready-pr.yml`
- `.github/workflows/ci.test.mjs` 或新增 dedicated regression test
- 視需要更新 `docs/draft-pr-auto-ready.md`
- 在 workflow 內寫入 `develop` / `main` 的 exact `required_checks` context 清單

驗收：

- 測試 opt-in draft PR flow。
- stale SHA / missing label / bot author / failed check 都不會 ready。
- required check context 與 GitHub UI / `gh pr checks` 實際顯示一致。

### PR C：Workflow health check

目的：保護 workflow / dependabot config 自身。

修改：

- `.github/workflows/workflow-health.yml`
- workflow regression 或 config validation script
- 高權限 workflow permission allowlist

驗收：

- workflow path 變更會跑 health check。
- actionlint / config validation 能擋住常見錯誤。
- 新增 write permission workflow 但未更新 allowlist 時會 fail。

### PR D：Security scan first pass

目的：先導入高訊號安全掃描。

修改：

- `.github/workflows/security-scan.yml`
- 視需要新增 allowlist 文件

驗收：

- `govulncheck` 正常跑。
- fake secret 不造成長期紅燈。

### PR E：Migration guard

目的：自動提示/阻擋高風險 SQL migration。

修改：

- `.github/workflows/migration-guard.yml` 或 `pr-scope-police.yml`
- regression test
- `migration-exception` label policy 文件或 sticky comment wording

驗收：

- additive migration pass。
- destructive migration fail。
- `migration-exception` 只降級 destructive heuristic，不移除 migration checklist。

## 本文件明確不做

- 直接實作任何 workflow。
- 修改 branch protection 或 repository ruleset。
- 開啟新的 required check。
- 自動 approve、merge、comment、close GitHub PR。
- 把 security scan / migration guard 的所有 finding 直接升級為 blocker。
