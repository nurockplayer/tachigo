# GitHub Actions Debugging Playbook

這份 playbook 用在 tachigo 的 PR / CI / auto-ready 流程出問題時。目標是先判斷失敗屬於哪一層，再決定要修 PR metadata、workflow 邏輯，還是真的修產品程式碼。

## 使用時機

- PR 開了，但重型 CI 沒有跑。
- `PR Scope Police` 失敗，但不確定是欄位、scope，還是 diff 大小問題。
- `auto-ready` label 已加，但 draft PR 沒有自動轉成 ready。
- required checks 看起來都過了，但 auto-merge 沒有 armed。
- 修改 `.github/workflows/*.yml` 後，需要確認 workflow script regression 沒壞。

## 先分層

遇到 CI 紅燈時，先把問題分成下面幾類。

| 層級 | 代表訊號 | 先看哪裡 |
| --- | --- | --- |
| PR metadata | title/body/template 欄位不合規 | `docs/pr-scope-policy.md`、`.github/PULL_REQUEST_TEMPLATE.md` |
| Scope gate | 重型 CI 被跳過 | `.github/workflows/ci.yml` 的 `scope-gate` job |
| Scope police | PR 被標記或擋下 | `.github/workflows/pr-scope-police.yml` |
| Workflow regression | workflow script 本身壞掉 | `.github/workflows/ci.test.mjs` |
| Product CI | backend/frontend/dashboard/contract 測試失敗 | 對應 product surface 的 job log |
| Auto-ready | draft PR 沒轉 ready 或 auto-merge 沒 armed | `.github/workflows/auto-ready-pr.yml`、`ci.yml` 的 `auto-ready-after-ci` |

## 常見情境

### 重型 CI 沒有跑

先確認這是不是預期行為。

1. 如果 PR 只改 `docs/`、template、repo metadata，重型 product CI 會被跳過。
2. 如果 PR metadata 不合格，`scope-gate` 會把重型 CI 關掉。
3. 如果 frontend PR 依賴尚未 merge 的 backend contract，重型 CI 會被 dependency gate 擋住。

本地先跑：

```bash
make pr-meta-check TITLE="[chore] Example title" BODY_FILE=/tmp/pr-body.md
```

### PR Scope Police 失敗

先對照 `docs/pr-scope-policy.md`，不要直接改 workflow。

常見原因：

- PR title 沒有 `[backend]`、`[frontend]`、`[contract]`、`[discussion]`、`[release]`、`[infra]` 或 `[chore]` prefix。
- PR body 缺 `Source of truth`、`Depends on PR` 或 `本 PR 明確不做`。
- 同一個 PR 同時碰 backend / frontend / contracts surface。
- 檔案數或 diff 行數超過限制。
- frontend PR 依賴未落地 backend contract，但 PR body 沒標清楚 stacked / blocked 狀態。

### Auto-ready 沒有轉 ready

先確認 PR 是否符合 auto-ready 條件：

- PR 是 draft。
- 有 `auto-ready` label。
- base branch 是 `develop` 或 `main`。
- head branch 來自同一個 repo。
- 不是 Dependabot PR。
- required check snapshot 中列出的 checks 都已經是 success、neutral 或 skipped。

`develop` 的 required check snapshot 目前在 `.github/workflows/auto-ready-pr.yml`：

- `Scope gate`
- `Frontend build`
- `Dashboard build`
- `Contracts build`
- `Backend CI (gate)`

如果 GitHub UI 上的 check 名稱、app id 或 workflow job 名稱改了，要同步更新 workflow regression tests。

### 修改 workflow 後

凡是改到 `.github/workflows/ci.yml`、`.github/workflows/pr-scope-police.yml`、`.github/workflows/auto-ready-pr.yml` 或 related workflow script，至少跑：

```bash
node --test .github/workflows/ci.test.mjs
```

如果改到 backend CI cache wiring，也跑：

```bash
bash infra/scripts/check-backend-ci-cache.sh
```

如果改到 PR 開啟或 metadata preflight script，也跑：

```bash
bash infra/scripts/pr-open.test.sh
```

## 不要做的事

- 不要為了讓 docs / metadata PR 轉綠，把 backend 或 frontend 修補混進同一個 PR。
- 不要用 `scope-exception` 當一般逃生門；只有真的不可拆時才用。
- 不要在不理解 `scope-gate` 輸出前直接改 product code。
- 不要把影片或外部工具示範當成 tachigo 的 implementation source of truth。
- 不要在 workflow debug PR 順手調整產品功能、schema、API contract 或 UI。

## 建議處理順序

1. 讀失敗 job 的 summary 和 notice，判斷是哪一層。
2. 若是 metadata / scope 問題，先修 PR body 或拆 PR。
3. 若是 workflow script 問題，跑 `.github/workflows/ci.test.mjs` 復現。
4. 若是 product CI 問題，只在該 product surface 的獨立 PR 中修。
5. 若發現需要新機制，先開 issue 或 discussion，不混入當前修補 PR。

