# Codex Autonomous Workflow

本文件定義 tachigo 的 autonomous product work 規範。目標是讓 Codex 在 issue-first、PR-first、review-gated 的流程下，作為總控 agent 推進產品級工作，同時把可切分的低風險工作交給 worker/subagent。

## 核心原則

- 每個實作或政策變更都要先有 GitHub issue，再開 PR 回 `develop`。
- 不直接 push 到 `develop` 或 `main`。
- 一個 PR 只對齊一張 source issue；額外想法、future work、重構建議另開 issue / PR。
- 總控 agent 負責架構、計劃、scope、風險取捨、最終 review 與 merge decision。
- worker/subagent 負責可切分的探索、文件、測試、一般實作、CI log 摘要、GitHub readback。
- 能用 Spark 或低推理成本 worker 完成的 routine 工作，優先交給低成本 worker。
- schema、migration、auth、wallet signature、points ledger、金流、權限模型、merge decision 必須由總控或高推理 worker 審查。

## Start-of-work Delegation Gate

- 只要是 autonomous work，第一步必須先分派 worker，再進入計劃、開 issue、讀資料、或開始實作。
- 必須先讀 GitHub issue、相關文件、以及現有 metadata，才能決定 worker profile、模型強度與切分方式。
- 必須在 issue body 先寫 `Issue Delegation Plan`，才能把任務交給 worker；不得先做一部分再補 plan。
- 必須在 PR body 先寫 `Delegation Execution Log`，才能把實作結果交回；不得只交 diff 不交執行紀錄。
- 只有 `trivial/self-only exception` 可以不分派 worker；它必須是單一檔案或單一小改動，沒有跨檔案影響，沒有需要並行驗證的切面，而且總控可以獨立完成。
- 只要使用 `trivial/self-only exception`，就必須在 `Issue Delegation Plan` 或 `Delegation Execution Log` 內明講原因與範圍；不得把例外原因藏在備註。
- `scope-exception` 只影響 PR Scope Police，不能拿來豁免開工前分派。

## Issue Delegation Plan

- 每張 autonomous issue 必須包含 `Issue Delegation Plan`。
- `Issue Delegation Plan` 必須明列 source issue、worker profile、預期模型強度、任務切分、驗證方式、以及預期產物。
- `Issue Delegation Plan` 必須用明確欄位寫出誰負責讀資料、誰負責實作、誰負責驗證、誰負責最後 readback。
- 只有 `trivial/self-only exception` 可以寫成不分派 worker；這種情況必須寫明為什麼不需要 worker、為什麼可以由總控單獨完成、以及會用什麼驗證證據收尾。
- 不得把 `Issue Delegation Plan` 寫成空泛目標或大綱；它必須足夠讓 worker 直接開工。

## PR Delegation Execution Log

- 每張 autonomous PR 必須包含 `Delegation Execution Log`。
- `Delegation Execution Log` 必須對照 source issue 的 `Issue Delegation Plan`，寫出實際使用的 worker profile、實際模型強度、實際驗證證據、以及最後自審結果。
- `Delegation Execution Log` 必須說明哪些工作交給 worker，哪些工作由總控保留，不能只寫完成摘要。
- 如果真的用了 `trivial/self-only exception`，`Delegation Execution Log` 必須明講例外原因、例外範圍、以及為什麼不需要 worker。
- 不得把 `Delegation Execution Log` 當成可省略的附註；沒有這一段，就不算 autonomous PR 完成交付。

## 新對話啟動 Checklist

新對話框不要只靠使用者貼一段提示詞來維持一致性。提示詞只負責啟動流程；真正的 source of truth 是 repo 內文件與 GitHub issue / PR metadata。

建議使用者在新對話框貼以下啟動語：

```text
請以 <你的本機 tachigo 專案路徑> 為工作目錄（例如：~/workspace/tachigo）。

開始前請先讀：
- AGENTS.md
- CLAUDE.md
- docs/ai/codex-autonomous-workflow.md
- docs/pr-scope-policy.md

之後所有工作都照 tachigo 的 Autonomous Worker Profiles 執行：
- 先找或開 GitHub issue
- 從 develop 開 branch
- PR 目標一律 develop
- PR body 必須符合 template 與 PR Scope Police
- 可委派 worker/subagent，但總控保留 scope、架構、review、merge decision
- merge 前必須等待 CodeRabbit 與 chatgpt-codex-connector review/readback
- CodeRabbit rate limit 時改由總控 self-review 並留下證據
- chatgpt-codex-connector 沒 comment 但有 reaction 可視為已看過
- 不要直接 push develop/main
- 不要 merge，除非我明確授權
```

貼上前請先把 `<你的本機 tachigo 專案路徑>` 換成目前 clone 的 repo 根目錄，避免 agent 在錯誤目錄讀取文件或執行 GitHub / git 操作。

接到這段啟動語後，總控 agent 的第一輪動作必須是：

1. 讀取上列文件，而不是只依賴提示詞記憶。
2. 回報目前工作目錄、branch、dirty state、target base branch。
3. 搜尋既有 issue，避免重複開票。
4. 若需要新 issue，先建立 source issue，再開始實作。
5. 若主 worktree 已 dirty，改用乾淨 worktree 或停止回報，不覆蓋使用者變更。

## Worker Profiles

| Profile | Preferred model | Reasoning | 用途 |
| --- | --- | --- | --- |
| `controller` | GPT-5.5 | high / xhigh | 總控、架構、計劃、issue/PR scope、最終 review、merge decision |
| `ops_spark` | GPT-5.3-Codex-Spark | low / medium | GitHub issue/PR metadata、label、CI readback、routine terminal、PR body readback |
| `repo_scout` | GPT-5.3-Codex-Spark | medium | 快速掃 codebase、找既有 pattern、列影響範圍與測試缺口 |
| `docs_worker` | GPT-5.3-Codex-Spark | medium | docs、issue body、PR body、規格草稿、驗證摘要 |
| `test_worker` | GPT-5.4-mini / GPT-5.4 | medium / high | 單元測試、fixture 整理、workflow regression、測試缺口補強 |
| `backend_worker` | GPT-5.4 | high | Go service、handler、repository、middleware、API tests |
| `frontend_worker` | GPT-5.4 | medium / high | extension/dashboard UI、React state、loading/error flow、frontend tests |
| `schema_worker` | GPT-5.5 | high / xhigh | DB schema、migration、idempotency、ledger、資料一致性 |
| `review_worker` | GPT-5.4 / GPT-5.5 | high | PR diff review、regression risk、缺測檢查、merge 前風險掃描 |

模型名稱是 preferred profile，不是硬依賴。若當前環境不可用，總控應選擇同級或更保守的替代模型，並保留高風險決策的人工可讀證據。

## Routing Rules

| 場景 | 預設指派 |
| --- | --- |
| GitHub issue/label/PR body/check readback | `ops_spark` |
| 已定 scope 的 issue creation / PR metadata repair | `ops_spark`，總控負責 scope 核准 |
| CI log 初步分析、routine terminal 檢查 | `ops_spark` |
| 大範圍找檔案、讀 code pattern | `repo_scout` |
| 文件、計劃、issue/PR 草稿 | `docs_worker` |
| 小範圍 test 補強 | `test_worker` |
| 一般 backend handler/service 實作 | `backend_worker` |
| schema、migration、ledger、idempotency | `schema_worker` |
| extension/dashboard UI | `frontend_worker` |
| merge 前 review | `review_worker`，總控做 final decision |

## GitHub 操作分工

`ops_spark` 可以處理：

- 查 issue、建立已核准 scope 的 issue、補 issue comment。
- 驗證 issue URL、state、labels、milestone。
- 建 PR、更新 PR body、補缺失標題或 body。
- 查 PR latest head SHA、base/head branch、merge state、status check rollup。
- 查 CodeRabbit / `chatgpt-codex-connector` comment 與 review 狀態。
- 抓 failed check logs 並整理摘要。

總控保留：

- 是否採用某個架構方案。
- 是否拆 issue / PR。
- 是否 merge、merge method 與 `--match-head-commit`。
- conflict、failed check、stale review、review disagreement 的處置。
- issue closeout 與 final evidence。

## Automated Review Gate

Autonomous PR merge 前，總控必須完成 fresh readback：

1. 確認最新 PR head SHA、base branch、mergeability 與 CI/check 狀態。
2. 確認 `PR Scope Police` 通過；若 fail，先縮 scope 或拆 PR，不做完整 review。
3. 確認 CodeRabbit 已產生實際 review。不能只看 status context，因為 skipped review 也可能顯示 success。
4. 若 CodeRabbit 明確 rate limit，同一張 PR 不重複要求 review，改由總控做 self-review 並在 PR 留下替代 review 證據。
5. 確認 `chatgpt-codex-connector` 已 review。若沒有 finding，它可能只在第一則 PR comment 左下角留下 reaction；只有沒有 reaction、也沒有 comment 時，才手動要求 review。
6. 對每個 actionable finding，merge 前只能二選一：修正並重跑相關驗證，或留下不採用的技術佐證 comment。
7. GitHub 允許時，將已處理的 review thread/comment resolve。
8. 若 push 新 commit，merge 前重新讀回 head SHA 與 review/check 狀態。

CodeRabbit 由 `.coderabbit.yaml` 設定 `reviews.auto_review.base_branches: [".*"]`，讓 PR target branch 不限 default branch 都能觸發 auto review。

## PR Scope Police Contract

開 PR 前必須先符合 `.github/workflows/pr-scope-police.yml` 與 [PR Scope Policy](../pr-scope-policy.md)，避免靠 CI 打回才修：

- PR title 必須以 `[backend]`、`[frontend]`、`[contract]`、`[discussion]`、`[release]`、`[infra]`、`[chore]` 之一開頭。
- PR body 必須引用至少一個 issue 或 PR 編號，例如 `#620`。
- PR body 必須包含 `Source of truth:`。
- PR body 必須包含 `Depends on PR: none` 或 `Depends on PR: #123`。
- PR body 必須包含 `本 PR 明確不做`。
- Changed files 不得超過 35。
- Diff lines 超過 600 會警告，超過 1000 會 fail；目前 `scope-exception` 會完整 bypass scope police，因此只能用於 maintainer 已明確接受的大型或例外 PR。
- 同一 PR 不得混 backend、frontend、contract product surface。
- 不得把 `scope-exception` 當作一般擴大 scope 的工具；若需要保留行數上限，必須先改 workflow 讓上限真的生效。

## Standard Autonomous Loop

1. 評估現況與產品級缺口。
2. 總控決定 issue scope，必要時讓 `ops_spark` 建 issue 並讀回 URL/state/labels。
3. 從最新 `origin/develop` 建 scoped branch 或乾淨 worktree。
4. 依任務類型指派 worker。
5. worker 回報變更、驗證與剩餘風險。
6. 總控審查 diff，補必要修正。
7. 跑 relevant validation；高風險改動需 full validation。
8. 用 `.github/PULL_REQUEST_TEMPLATE.md` 或 `make pr-open` 開 PR 到 `develop`，body 必須含 source issue、dependency、non-goals、validation。
9. 等 CI/checks/review 狀態 fresh readback。
10. Merge 前完成 Automated Review Gate。
11. 使用 guarded merge，並以 latest head SHA 防止 stale merge。
12. Merge 後補 issue evidence，確認 issue state / labels / closeout。

## Validation Policy

- Backend Go 變更至少跑相關 `go test`；必要時跑 `docker compose run --no-deps --rm app go test ./...`。
- Frontend 變更至少跑相關 lint/test/build；依 scope 選 extension 或 dashboard。
- Workflow / policy 變更需跑 `node --test .github/workflows/ci.test.mjs`，並做 PR metadata preflight。
- Docs-only 變更至少跑 `git diff --check`；若文件描述 repo command，需確認命令仍存在或明確標示限制。
- 若任何驗證無法執行，PR body 與最終回報都必須寫明原因與剩餘風險。

## Storage Boundaries

| 位置 | 用途 |
| --- | --- |
| `docs/ai/codex-autonomous-workflow.md` | 團隊可見的正式 autonomous policy |
| `AGENTS.md` / `CLAUDE.md` | 短版入口規則與本文件連結 |
| Codex memory | 使用者偏好與歷史決策輔助，不作為唯一規範來源 |
| `.codex/config.toml` / router script | 未來 CLI 自動路由實作；目前不在本文件範圍 |

## Out of Scope

以下需要另開 issue / PR：

- CLI 自動 router 或 long-running daemon。
- `.codex/config.toml` profile 實作。
- 自動 merge daemon 或跨 repo release orchestration。
- 把任何一個 worker profile 當作可跳過 review gate 的理由。
