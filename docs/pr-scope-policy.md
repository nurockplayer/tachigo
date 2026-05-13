# PR Scope Policy

這份文件說明 `tachigo` 的 PR 邊界規則，以及 GitHub 上需要如何設定，避免再次出現超大包、跨 scope、review 失焦，或前端建立在未落地 backend contract 上的 PR。

另外，repo 採用 Git Flow：

- 日常功能開發與功能 PR 一律先進 `develop`
- `main` 只接受正式 release promotion
- `.github/workflows/release-pr.yml` 每天檢查一次 `develop -> main` release PR
- 自動 release PR 需距離上次 release 至少 72 小時，且上次 release 後已 merge 至少 10 個 PR
- 若距離上次 release 已滿 7 天，即使 PR 數低於 10 個也可自動開 release PR，避免 `main` 長期落後
- `workflow_dispatch` 可手動開 release PR；automation 只開 PR，不自動 merge
- 目前暫不使用 `release/*` branch
- 待未來有正式部署、freeze window、hotfix/backport 需求時，再升級為完整 release branch 流程

## 目標

- 一個 PR 只做一個明確問題
- reviewer 不需要先讀完整包才知道該不該擋
- 超出 scope 的 PR 先被自動擋下，再談 review
- 重型 CI 不要浪費在明顯不該進 review 的 PR 上

## 自動規則

repo 目前有一個 GitHub Actions workflow：

- `.github/workflows/pr-scope-police.yml`

它會在 PR 開啟、更新、編輯時檢查以下規則：

- PR title 必須以 `[backend]` / `[frontend]` / `[contract]` / `[discussion]` / `[release]` / `[infra]` / `[chore]` 開頭
- 一般 feature PR 的 body 必須包含 issue / PR 編號，例如 `#123`
- PR body 必須包含 `Source of truth`
- PR body 必須包含 `Depends on PR`
- 產品程式碼 PR 必須明確標記 backend contract 是否已經在 `develop`
- PR body 必須包含 `本 PR 明確不做`
- PR 變更檔案數超過 `35` 個時 fail
- PR diff 超過 `600` 行時 warning，超過 `1000` 行時 fail
- PR 不可同時改多個 product surface：
  - backend surface：`services/api/`（舊路徑 `backend/` 仍作為歷史 PR 判斷）
  - frontend surface：`apps/dashboard/`、`apps/extension/`（舊路徑 `dashboard/`、`tachimint/` 仍作為歷史 PR 判斷）
  - contract surface：`contracts/`
- `[backend]` PR 不可修改 frontend surface
- `[frontend]` PR 不可修改 backend surface
- `[contract]` PR 不可修改 backend / frontend surface
- `[frontend]` PR 若依賴尚未 merge 的 backend contract，會被 dependency gate 擋下

未來 monorepo 的共享套件路徑（例如 `packages/shared-types/`、`packages/api-client/`）會被 workflow 辨識，但目前不單獨視為 product surface。是否能與某個 surface 同 PR 出現，仍應以 issue scope 與 reviewer 判斷為準，避免把 shared package 變成順手混改的出口。

### Dependabot maintenance PR

對 `dependabot[bot]` 開的 maintenance PR，`PR Scope Police` 會保留真正有意義的自動檢查：

- product surface 不可混雜
- diff / changed files 不可超過硬上限
- 不再要求人工模板欄位

Dependabot maintenance PR 目前不會套用 frontend/backend 依賴關係用的 dependency gate。這條 gate 依賴人工填寫的 PR 模板欄位（例如 `Depends on PR` 與 `Backend contract already in develop`），而這次規則調整的目的正是不要再要求 bot 補這類 metadata。

因此 Dependabot PR 目前只保留 scope / size 類型的自動檢查，不再要求補齊人工模板欄位，例如：

- title prefix
- `Source of truth`
- `Depends on PR`
- `Backend contract already in develop`
- `本 PR 明確不做`

原因是這些欄位主要服務人工撰寫的 feature / release PR；對 Dependabot dependency bump 而言，持續人工補 metadata 成本高、訊號低，也容易讓 reviewer 浪費時間在固定格式而非實際風險。

### Docs / template / metadata PR

對只修改非產品程式碼的文件、模板或 repo metadata PR，`PR Scope Police` 會降低不相關的策略檢查嚴格度。

目前視為 docs / template / metadata-only 的路徑：

- `docs/`
- `docs/ai/`
- `plans/`
- `infra/`
- `.github/ISSUE_TEMPLATE/`
- `.github/PULL_REQUEST_TEMPLATE.md`
- repo root 的 Markdown 文件，例如 `AGENTS.md`、`CLAUDE.md`、`README.md`
- `.gitignore`
- `.gitattributes`

這類 PR 仍需保留基本 scope 訊號：

- PR title prefix
- issue / PR 編號，例如 `#123`
- `Source of truth`
- `Depends on PR`
- `本 PR 明確不做`
- 檔案數 / diff 大小限制
- product surface 不可混雜

這類 PR 不需要填寫 backend contract 是否已經在 `develop`，因為文件 / template / metadata 改動不引入 frontend 對 backend API 的依賴。

另外，這類 PR 不應再被 product surface 的 inherited 紅燈拖住 review 流程，因此：

- `Scope gate` 會直接略過 backend / frontend / dashboard 的 heavy CI
- 仍保留 `PR Scope Police`、workflow regression 與其他 metadata / policy 檢查
- 若 docs/template PR 因為 restack 需求碰到 `develop` 上的產品線紅燈，應拆成獨立 product fix PR，不可把 inherited 修補留在 docs PR

`Source of truth` 與 `Depends on PR` 可使用半形或全形冒號，例如 `Source of truth:` 或 `Source of truth：`。自動檢查不要求模板文字必須逐字完全相同，但仍要求欄位語意存在。

## 正式 release PR

以下情況視為正式支援的 release promotion PR，而不是 scope exception：

- base branch = `main`
- head branch = `develop`
- title prefix = `[release]`

這類 PR 的性質是把已在 `develop` 收斂完成的內容整批 promotion 到 `main`，因此：

- 不套用一般 feature PR 的檔案數上限 `35`
- 不套用一般 feature PR 的 diff 行數上限 `1000`
- 不套用單一 product surface 限制
- 不會因為大包而被 `PR Scope Police` 自動關閉
- 仍會要求 PR body 補齊基本資訊，例如 `Source of truth`、`Depends on PR`、`Backend contract already in develop`、`本 PR 明確不做`

換句話說，超大包 `develop -> main` PR 在這個流程裡是正式合法路徑，但其他分支組合仍照一般 scope 規則檢查。

## 自動處置

當 PR 違反規則時：

- `PR Scope Police` check 會 fail
- PR 會收到一則可更新的 sticky comment
- 嚴重 scope 違規會自動加上 `scope-violation` label
- 依賴未落地的前端 PR 會自動加上 `blocked-by-dependency` label
- 若屬於嚴重違規，PR 會被自動關閉

目前視為嚴重違規的情況：

- 檔案數超過 `35`
- diff 超過 `1000` 行
- 同時改多個 product surface
- `[backend]` PR 去改 frontend surface
- `[frontend]` PR 去改 backend
- `[contract]` PR 去改 backend / frontend surface

## 例外機制

若真的有不可拆的特殊 PR，可加上：

- `scope-exception`

這個 label 目前會 bypass 一般 scope / size / product-surface checks，沒有額外的例外行數上限。
若 PR 屬於 Codex autonomous PR，`scope-exception` 不會 bypass autonomous delegation gate；PR 仍必須填寫 `Delegation Execution Log`，並列出 worker profile 或 trivial/self-only exception reason。

關於 autonomous PR 的判定與觸發條件：

- `codex` / `codex-automation` / `auto-ready` label 或 `Delegation Execution Log` 在正式欄位（`Source issue delegation plan`、`Actual worker profile(s)`、`Model strength`、`Verification evidence`、`Self-review / exception reason`）有實質內容時，視為 autonomous PR。
- 自動化 PR 還須在同區塊填寫 `Worker session closeout`，且內容不可空白、`n/a`、`none`、`無`、`不適用`。
- 自動化 PR 必須至少有一條 spawn directive，且同時包含 `profile=`、`model=`、`reasoning=`、`controller_fallback=`；若 `controller_fallback=allowed`，同一行必須有非空 `fallback_reason=`。
- `ops_spark` 類任務不得使用高階 controller model，除非同一條 spawn directive 留下 fallback reason。
- 自動化 PR 必須有 `Review conversation closeout` 或 `Final merge gate` evidence；ready-to-merge closeout 不可留下裸 `pending`。
- 若已填 `Review conversation closeout`，autonomous PR 還必須提供 `review_triage_ref`、`root_cause_gate_ref`、`finding_disposition_ref` 三個 evidence ref。剛開 PR 可填 `pending with reason`，但 bare `pending` 與 ready-to-merge closeout 的任何 pending ref 都會被擋下。
- sticky comment 會顯示 `Autonomous evidence snapshot`，列出 delegation log、worker closeout、spawn directives、controller fallback detail、review closeout、final merge gate 與 pending 是否已清除。
- Scope Police 對 review triage 只做薄檢查：確認 ref/pending marker，不解析完整 spec-injector review triage matrix、root-cause schema 或 disposition checker。
- review triage schema/checker 的 authoritative implementation 應留在 `Erick52106/spec-injector#232`、`#233`、`#234`、`#235`；tachigo 不複製該 checker。
- section 內只要是空白、`n/a`、`none`、`無`、`不適用`，或非正式欄位的自由備註，不會啟動 delegation gate。
- 只有 section 標題但欄位空白，不算 autonomous PR，也不會觸發 delegation gate。
- 一般非 autonomous PR 不需要填寫 worker profile；不因 `Delegation Execution Log` 缺漏而被視為流程違規。

Autonomous Worker Profiles v2 的完整 evidence discipline 與 `spec workflow-check` local-only 接入點見 [docs/ai/autonomous-pr-gates.md](ai/autonomous-pr-gates.md)。`spec-injector` 不得把 `.spec-injector/`、spec output 或 private local context commit 進 repo；未使用此工具的 PR 可填人工 checklist / `n/a`。

使用原則：

- 只有 maintainer 可以加
- 預設不加
- 只有在「同一張票的必要前置真的無法拆開」時才使用
- 不能把它當成超大包 PR 的常態逃生門
- 若未來需要保留例外 PR 的行數上限，必須先同步修改 `.github/workflows/pr-scope-police.yml` 與 `.github/workflows/ci.yml`，讓上限真的在 CI 生效

注意：

- 正式 `develop -> main` release PR 不需要使用 `scope-exception`
- `scope-exception` 只保留給非正式 release promotion 的特殊情況

## CI Gate

repo 的 CI 目前改成：

- PR 先跑 `PR Scope Police`
- `.github/workflows/ci.yml` 也會直接跑在 PR 上，但會先經過一個輕量 `Scope gate`
- 只有目前符合同一套 scope 規則、且沒有被 dependency gate 擋住的 PR，才會繼續跑 backend / frontend / dashboard 的 docker build 與測試
- 若 `[frontend]` PR 依賴尚未 merge 的 backend contract，重型 CI 會直接跳過
- 若 PR 是正式 `[release]` 的 `develop -> main` promotion，重型 CI 會照常執行，不因 diff 過大而被 scope gate 跳過
- 若 PR 是 docs / template / metadata-only，重型 product CI 會直接跳過，避免 inherited product failures 造成無限循環

## Conflict / Restack 規則

對 docs / template / metadata-only PR，或任何單一小 scope PR：

- 不要用 `merge develop` 的方式解 conflict
- 正確做法是從最新 `develop` 開新 branch，將原 PR commit `cherry-pick` 過去後重開或更新 PR
- 若 restack 後發現 inherited 的 backend / frontend / dashboard failure，必須拆成獨立 product fix PR，不可把修補留在原本的小 scope PR

原因：

- `merge develop` 會把整個 base branch 的當下狀態搬進 PR，容易把不屬於本 PR 的紅燈一起帶進來
- 小 scope PR 為了讓 inherited CI 轉綠而修改產品程式碼，最後會落入「不修就 CI 紅、修了就 scope 污染」的死循環
- 用最新 `develop` 重開 branch + `cherry-pick` 原 PR commit，可以把衝突處理限制在本 PR 自身範圍

## 本地 PR preflight

開 PR 前可以先在本地跑 metadata preflight，提早檢查 PR title、body template 欄位、dependency PR、backend contract 與 product surface 是否符合上方 Scope Police 規則。

直接檢查既有 PR body 檔案：

```bash
make pr-meta-check TITLE="[chore] Example title" BODY_FILE=/tmp/pr-body.md
```

開 PR 前先檢查，通過後才呼叫 `gh pr create`：

```bash
make pr-open TITLE="[chore] Example title" BODY_FILE=/tmp/pr-body.md
```

Codex task PR 預設應使用 auto-ready 流程：

```bash
make pr-open TITLE="[chore] Example title" BODY_FILE=/tmp/pr-body.md AUTO_READY=1
```

可選參數：

- `BASE`：預設 `develop`
- `HEAD`：預設目前 branch
- `DRAFT=1`：建立 draft PR
- `AUTO_READY=1`：建立 draft PR 並加上 `auto-ready` label，等 required
  checks 通過後由 workflow 自動轉成 Ready for review

Codex task PR 應使用 `AUTO_READY=1`，讓 wrapper 一次建立 draft PR 並加上
`auto-ready` label。若直接使用 `gh pr create`，則使用
`--draft --label auto-ready`。

底層腳本：

- `infra/scripts/pr-metadata-check.sh`
- `infra/scripts/pr-open.sh`

這組本地 preflight 只檢查 PR metadata 與目前 branch diff 的基本 surface 規則；push 前的大型 diff 檢查仍由 `infra/githooks/pre-push` 負責，GitHub 上的最終 gate 仍是 `.github/workflows/pr-scope-police.yml`。

## GitHub 設定

建議在 GitHub repo settings 這樣設定：

### Branch Protection / Ruleset

對 `develop` 與 `main`：

- Require a pull request before merging
- Require approvals: at least 1
- Dismiss stale approvals when new commits are pushed
- Require conversation resolution before merging
- Block direct push
- Block force push
- Block branch deletion

建議補充：

- `main` 僅允許來自 `develop` 的正式 release PR 合併
- release PR title 使用 `[release]`

### Required Checks

至少設成 required：

- `PR Scope Police / Scope police`
- `CI / Backend CI (gate)`
- `CI / Frontend build`
- `CI / Dashboard build`

注意：

- `PR Scope Police` 應該是第一道 gate
- 後面三個 CI checks 會直接出現在 PR 上；若 scope 不合格，job 會在 `Scope gate` 後被略過

## Reviewer 指南

若 `PR Scope Police` 已 fail：

- 不需要先做完整 code review
- 先要求作者拆 PR 或縮 scope
- 若是 dependency block，先要求作者等依賴 merge，或改成 stacked PR
- 只有在 maintainer 明確決定加 `scope-exception` 時，才進一步 review

若 PR 是正式 `[release]` 的 `develop -> main` promotion：

- 不要用 feature PR 的檔案數 / diff 大小標準要求它拆 PR
- review 重點改成 release readiness：CI、branch protection、是否包含不該進版的內容、是否需要延後到下個 release cycle

若 PR 雖然通過自動檢查，但 reviewer 仍判斷 scope 已混掉：

- 可以直接要求拆 PR
- 可以要求作者把必要前置與功能實作拆開
- 不需要因為 CI 全綠就接受超出 issue 範圍的內容

## 常見範例

應該被擋：

- `[backend]` PR 同時修改 `backend/` 與 `apps/dashboard/`
- `[frontend]` PR 依賴 `#123` 的 backend contract，但 `#123` 還沒 merge 到 `develop`
- 一張票只做 dashboard UI，PR 卻順手改 migration、router、service、docs
- PR 改了 50 個檔案，混入多個 issue 的工作

可以接受：

- `[backend]` PR 只改 `backend/`，且有清楚的 source of truth
- `[frontend]` PR 只改 `apps/dashboard/`，必要測試一起補齊
- `[discussion]` PR 只改文件，不碰產品程式碼
- `[release]` PR 從 `develop` 整批 promotion 到 `main`

## 後續調整

若之後規則太寬或太嚴，可優先調整這些門檻：

- `hardMaxChangedFiles`
- `hardMaxDiffLines`
- product surface 定義
- 哪些違規屬於 auto-close

修改位置：

- `.github/workflows/pr-scope-police.yml`
