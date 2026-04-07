# PR Scope Policy

這份文件說明 `tachigo` 的 PR 邊界規則，以及 GitHub 上需要如何設定，避免再次出現超大包、跨 scope、review 失焦，或前端建立在未落地 backend contract 上的 PR。

## 目標

- 一個 PR 只做一個明確問題
- reviewer 不需要先讀完整包才知道該不該擋
- 超出 scope 的 PR 先被自動擋下，再談 review
- 重型 CI 不要浪費在明顯不該進 review 的 PR 上

## 自動規則

repo 目前有一個 GitHub Actions workflow：

- `.github/workflows/pr-scope-police.yml`

它會在 PR 開啟、更新、編輯時檢查以下規則：

- PR title 必須以 `[backend]` / `[frontend]` / `[contract]` / `[discussion]` 開頭
- PR body 必須包含 issue / PR 編號，例如 `#123`
- PR body 必須包含 `Source of truth`
- PR body 必須包含 `Depends on PR`
- PR body 必須明確標記 backend contract 是否已經在 `develop`
- PR body 必須包含 `本 PR 明確不做`
- PR 變更檔案數超過 `35` 個時 fail
- PR diff 超過 `1800` 行時 fail
- PR 不可同時改多個 product surface：
  - `backend/`
  - `dashboard/`
  - `tachimint/`
- `[backend]` PR 不可修改 `dashboard/` 或 `tachimint/`
- `[frontend]` PR 不可修改 `backend/`
- `[contract]` PR 不可修改 `backend/` / `dashboard/` / `tachimint/`
- `[frontend]` PR 若依賴尚未 merge 的 backend contract，會被 dependency gate 擋下

## 自動處置

當 PR 違反規則時：

- `PR Scope Police` check 會 fail
- PR 會收到一則可更新的 sticky comment
- 嚴重 scope 違規會自動加上 `scope-violation` label
- 依賴未落地的前端 PR 會自動加上 `blocked-by-dependency` label
- 若屬於嚴重違規，PR 會被自動關閉

目前視為嚴重違規的情況：

- 檔案數超過 `35`
- diff 超過 `1800` 行
- 同時改多個 product surface
- `[backend]` PR 去改前端
- `[frontend]` PR 去改 backend
- `[contract]` PR 去改 backend / dashboard / tachimint

## 例外機制

若真的有不可拆的特殊 PR，可加上：

- `scope-exception`

這個 label 會 bypass `PR Scope Police`。

使用原則：

- 只有 maintainer 可以加
- 預設不加
- 只有在「同一張票的必要前置真的無法拆開」時才使用
- 不能把它當成超大包 PR 的常態逃生門

## CI Gate

repo 的 CI 目前改成：

- PR 先跑 `PR Scope Police`
- `.github/workflows/ci.yml` 也會直接跑在 PR 上，但會先經過一個輕量 `Scope gate`
- 只有目前符合同一套 scope 規則、且沒有被 dependency gate 擋住的 PR，才會繼續跑 backend / frontend / dashboard 的 docker build 與測試
- 若 `[frontend]` PR 依賴尚未 merge 的 backend contract，重型 CI 會直接跳過

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

### Required Checks

至少設成 required：

- `PR Scope Police / Scope police`
- `CI / Backend tests`
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

若 PR 雖然通過自動檢查，但 reviewer 仍判斷 scope 已混掉：

- 可以直接要求拆 PR
- 可以要求作者把必要前置與功能實作拆開
- 不需要因為 CI 全綠就接受超出 issue 範圍的內容

## 常見範例

應該被擋：

- `[backend]` PR 同時修改 `backend/` 與 `dashboard/`
- `[frontend]` PR 依賴 `#123` 的 backend contract，但 `#123` 還沒 merge 到 `develop`
- 一張票只做 dashboard UI，PR 卻順手改 migration、router、service、docs
- PR 改了 50 個檔案，混入多個 issue 的工作

可以接受：

- `[backend]` PR 只改 `backend/`，且有清楚的 source of truth
- `[frontend]` PR 只改 `dashboard/`，必要測試一起補齊
- `[discussion]` PR 只改文件，不碰產品程式碼

## 後續調整

若之後規則太寬或太嚴，可優先調整這些門檻：

- `hardMaxChangedFiles`
- `hardMaxDiffLines`
- product surface 定義
- 哪些違規屬於 auto-close

修改位置：

- `.github/workflows/pr-scope-police.yml`
