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
- Diff lines 超過 600 會警告，超過 1000 會 fail；`scope-exception` 的上限是 1500。
- 同一 PR 不得混 backend、frontend、contract product surface。
- 只有 maintainer 明確決定後才可使用 `scope-exception`。

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
