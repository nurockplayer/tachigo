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
- 近期 autonomous / infra 工作的成本拆解以「約 40% infra 本質複雜、約 60% 工作流自己製造摩擦」作為改善假設；不可省的 40% 用文件與 gate 固定，應消除的 60% 用 routing、closeout、lifecycle 與 follow-up split 降低。

## Start-of-work Delegation Gate

- 只要是 autonomous work，第一步必須先分派 worker，再進入計劃、開 issue、讀資料、或開始實作。
- 必須先讀 GitHub issue、相關文件、以及現有 metadata，才能決定 worker profile、模型強度與切分方式。
- 必須在 issue body 先寫 `Issue Delegation Plan`，才能把任務交給 worker；不得先做一部分再補 plan。
- 必須在 PR body 先寫 `Delegation Execution Log`，才能把實作結果交回；不得只交 diff 不交執行紀錄。
- 只有 `trivial/self-only exception` 可以不分派 worker；它必須是單一檔案或單一小改動，沒有跨檔案影響，沒有需要並行驗證的切面，而且總控可以獨立完成。
- 只要使用 `trivial/self-only exception`，就必須在 `Issue Delegation Plan` 或 `Delegation Execution Log` 內明講原因與範圍；不得把例外原因藏在備註。
- `scope-exception` 只影響 PR Scope Police，不能拿來豁免開工前分派。

### Autonomous PR 判定

- `Autonomous PR` 只有在以下任一條件成立時才成立：
  - PR 本身有 `codex` / `codex-automation` / `auto-ready` label；或
  - `Delegation Execution Log` 區塊有實質填寫內容，且至少有一個正式欄位有內容（`Source issue delegation plan` / `Actual worker profile(s)` / `Model strength` / `Verification evidence` / `Self-review / exception reason`），且不是 `n/a`、`none`、`無`、`不適用`。
- Autonomous PR 還須在 `Worker session closeout` 欄位填寫有意義內容（不得空白、`n/a`、`none`、`無`、`不適用`）。
- 自由備註（例如 `- pending follow-up`）不會單獨判定為有實質內容。
- 僅有 `## Delegation Execution Log` 標題而欄位空白，不會啟動 autonomous gate。
- 非 autonomous PR 不需要填寫 worker profile，也不需遵守 `Delegation Execution Log`/`Issue Delegation Plan` 的實作義務；只要是一般 PR，照一般 PR template 完成即可。

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

### Review Finding 修正規則（硬規則）

- 一旦接到 actionable review finding，總控不得先自行修正再補派 worker；先要立刻派出適合的 worker 先實作修正，再要求總控做 self-review 與 readback。
- `trivial/self-only exception` 仍可適用，但必須先在原本的 `Issue Delegation Plan / Delegation Execution Log` 記明是例外，不能等修完再補理由。

### Worker Lifecycle Closeout

- worker/subagent 回報完成後，總控必須讀回結果、判斷是否還需要追加任務。
- `Worker session closeout` 請直接寫入 `PR Delegation Execution Log`，至少要包含「已讀回 worker 結果並 close」或等效實質敘述，不可留空白或填 `n/a`、`none`、`無`、`不適用`。
- 若暫時不需要該 worker，總控必須立刻 close worker session，避免後續派工被 thread limit 卡住。
- 如果工具端回報 agent id `not found`，代表目前可操作 handle 已被回收或不在本次 active context；總控需要在回報中說明這是 stale handle，不得誤認為 worker 還在執行。
- `Delegation Execution Log` 應記錄 worker closeout 狀態；若 worker 無法關閉，需記錄原因與是否仍有 active handle。

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

## Cost Model 與摩擦預算

Autonomous Worker Profiles 的優化目標不是把所有工作都丟給低模型，而是把不可避免的 infra 複雜度與可消除的工作流摩擦分開處理。

| 類型 | 估計佔比 | 例子 | 處理方式 |
| --- | --- | --- | --- |
| infra 本質複雜 | 約 40% | GitHub API / review thread 狀態、CI check rollup、rate limit、跨 repo metadata、不同模型額度 | 接受其存在，用固定 readback 欄位與驗證命令降低不確定性 |
| 工作流自己製造摩擦 | 約 60% | 忘記先派 worker、總控自己做 routine readback、worker 完成後未 close、PR 後期無限加碼、review finding 沒有 comment/resolve 證據 | 用 routing map、lifecycle checklist、review closeout checklist、follow-up split policy 消除 |

每張 autonomous PR 的 `Delegation Execution Log` 應說明這次是否遇到 60% 類型的流程摩擦，以及已如何避免它重演。若只是 infra 本質複雜，應留下讀回證據；若是流程摩擦，優先修流程或另開 follow-up issue。

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

### ops_spark Routing Hardening

以下工作預設必須交給 `ops_spark` 或同級低成本替代 worker；若沒有委派，PR body 必須寫明 `trivial/self-only exception`：

- GitHub issue / label / milestone / duplicate readback。
- PR body、title、source issue、follow-up split、closeout comment 的 metadata repair。
- `gh pr checks`、status check rollup、CI log 初步摘要。
- CodeRabbit / `chatgpt-codex-connector` review/comment/reaction 狀態讀回。
- review thread list、resolved/unresolved count、thread URL 蒐集。
- pre-commit checklist、post-push head SHA / branch / PR URL readback。

以下工作不得只靠 `ops_spark` 做最終判斷：

- schema、migration、auth、wallet signature、points ledger、金流、權限模型。
- 是否接受 review finding 的技術取捨。
- 是否 merge、是否使用 `scope-exception`、是否把內容拆成 follow-up。
- 任何可能改變 runtime behavior 的修正。

若 `ops_spark` 額度、工具或模型不可用，總控可以改用同級低成本替代 worker，並在 `Delegation Execution Log` 寫明替代理由。若低成本 worker 都不可用，總控可以完成必要收斂工作，但必須把「worker unavailable」列為 closeout evidence，不得假裝已正常委派。

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

### Review Closeout Evidence Matrix

每個 automated review finding 在 merge 前都必須落入下列其中一種狀態：

| 狀態 | 必要證據 | 可否 merge |
| --- | --- | --- |
| 已修正 | commit SHA、thread/comment URL、相關驗證、resolved readback | 可以 |
| 不採用 | 技術理由 comment、剩餘風險、thread/comment URL、resolved readback | 可以 |
| rate limit fallback | CodeRabbit rate limit / skipped 證據、總控 self-review comment、驗證結果 | 可以，但不得重複要求同一張 PR 的 CodeRabbit review |
| connector reaction-only | `chatgpt-codex-connector` 對 latest head 的 reaction 或明確 review/comment readback | 可以 |
| 未回應 / 未 resolve | trigger comment 後仍無 latest-head readback，或 unresolved actionable thread 存在 | 不可以，除非使用者明確要求停止等待並留下風險證據 |

closeout comment 至少要列出 latest head SHA、CI/check 結論、unresolved thread count、CodeRabbit 狀態、`chatgpt-codex-connector` 狀態，以及每條 finding 的採納/不採納結果。

## Subagent Lifecycle 與 Thread-limit Cleanup

每個 worker 都要有四段生命週期：

1. `spawn`：記錄 profile、model、reasoning、任務範圍、不得修改的檔案或 GitHub metadata。
2. `readback`：總控讀回 worker 結果，不直接相信摘要中的結論。
3. `close`：不再需要 worker 時立即 close session，避免 thread-limit 擋住下一個必要派工。
4. `verify`：若 close 失敗，記錄 agent id、錯誤訊息與 fallback；若回報 `not found`，視為 stale handle，不視為 active worker。

spawn 前若已知 worker 額度或 thread limit 不足，先選低成本替代 worker；若替代 worker 也不可用，才使用總控 fallback。總控 fallback 必須寫入 PR body 的 `Worker session closeout` 或 `Self-review / exception reason`，避免把資源限制誤記成正常委派。

## Issue-first 與 Follow-up Split Policy

- autonomous work 從讀 issue、讀 repo、查 PR metadata 的第一步就套用 Worker Profiles；不得等到 PR 階段才補派 worker。
- issue body 應先列 `Issue Delegation Plan`，包含資料抓取、實作、驗證、GitHub readback 的切分。
- PR 後期只修 blocking review finding、CI failure、scope police failure、merge conflict；新的優化、文件補強、流程 polish 必須拆成 follow-up issue。
- follow-up issue 要在 PR body 或 closeout comment 留 URL，並明確標示「不阻塞目前 PR merge」。
- 如果一張 PR 反覆因新想法加碼，總控應停止擴張，將剩餘優化移出當前 PR。

## PR Template 與 Policy-test Hardening

- PR template 的示例不得是可被 workflow parser 誤判為正式欄位的可執行指令；示例若可能觸發 gate，必須放在 HTML comment 或改成非 executable wording。
- `.github/workflows/ci.test.mjs` 必須覆蓋 template、autonomous detection、placeholder、worker closeout、scope budget 與 `scope-exception` 不 bypass autonomous gate 的 regression。
- policy test 只鎖住可機器檢查的契約；更大範圍的 router / daemon / CLI profile 自動化必須另開 issue，不放進同一張 governance PR。
- PR diff 接近 600 行時應優先壓縮或拆分；超過 1000 行時不得靠 `scope-exception` 當常態解法。

## PR Scope Police Contract

開 PR 前必須先符合 `.github/workflows/pr-scope-police.yml` 與 [PR Scope Policy](../pr-scope-policy.md)，避免靠 CI 打回才修：

- PR title 必須以 `[backend]`、`[frontend]`、`[contract]`、`[discussion]`、`[release]`、`[infra]`、`[chore]` 之一開頭。
- PR body 必須引用至少一個 issue 或 PR 編號，例如 `#620`。
- PR body 必須包含 `Source of truth:`。
- PR body 必須包含 `Depends on PR: none` 或 `Depends on PR: #123`。
- PR body 必須包含 `本 PR 明確不做`。
- 非純 docs/template/metadata PR 必須勾選 `Backend contract already in develop` 的 yes/no；只要改到 `.github/workflows/**` 或其他 CI policy 檔，就不算 metadata-only，仍必填。
- Changed files 不得超過 35。
- Diff lines 超過 600 會警告，超過 1000 會 fail；`scope-exception` 只 bypass 一般 scope / size / surface gate，不會 bypass autonomous delegation gate。
- 同一 PR 不得混 backend、frontend、contract product surface。
- 不得把 `scope-exception` 當作一般擴大 scope 的工具；也不得用它豁免 `Delegation Execution Log`、worker profile 或 trivial/self-only exception reason。

## Standard Autonomous Loop

1. 評估現況與產品級缺口。
2. 總控決定 issue scope，必要時讓 `ops_spark` 建 issue 並讀回 URL/state/labels。
3. 從最新 `origin/develop` 建 scoped branch 或乾淨 worktree。
4. 依任務類型指派 worker。
5. worker 回報變更、驗證與剩餘風險。
6. 總控讀回 worker 結果；若不需追加任務，立即 close worker session。
7. 總控審查 diff，補必要修正。
8. 跑 relevant validation；高風險改動需 full validation。
9. 用 `.github/PULL_REQUEST_TEMPLATE.md` 或 `make pr-open` 開 PR 到 `develop`，body 必須含 source issue、dependency、non-goals、validation。
10. 等 CI/checks/review 狀態 fresh readback。
11. Merge 前完成 Automated Review Gate。
12. 使用 guarded merge，並以 latest head SHA 防止 stale merge。
13. Merge 後補 issue evidence，確認 issue state / labels / closeout。

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
