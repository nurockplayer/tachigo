# Autonomous Bootstrap

本文件是 tachigo 的 Hybrid AWP with Explicit Fallback Gate 單一啟動入口。當使用者要求 Autonomous、AWP、Hybrid AWP、Codex autonomous workflow，或要求搭配 `spec-injector` 執行 issue-first work 時，AI agent 先讀本文件，再由本文件展開其餘必讀文件、spec-injector gate、routing plan 與 PR closeout。

人類只需要指定這一份文件，例如：

```text
請讀 docs/ai/autonomous-bootstrap.md，使用 Hybrid AWP with Explicit Fallback Gate + spec-injector 處理 issue #123。
```

本文件只適用於明確啟動 autonomous / Codex workflow 的工作。一般 human PR 不需要填 AWP routing、worker profile、spec gate evidence 或 review-triage refs。

## 0. Hard Boundaries

- 不直接 push `develop` 或 `main`；日常 PR 目標是 `develop`。
- 不覆蓋 dirty worktree；若目前工作區有未提交變更，先回報並改用乾淨 worktree。
- 不在沒有 source issue 的情況下實作；先找既有 issue，沒有才依 repo 規則請求建立 issue 的明確授權。
- 不把 `.spec-injector/`、spec output、private context、task package 或工具暫存內容 commit 進 repo。
- 不因 spec-injector output 擴張 issue scope；guardrail、classifier、auto-discovered reference 都不是擴 scope 授權。
- 不修改遠端 issue / PR / label / review / branch，除非使用者已明確授權該公開狀態變更；授權範圍與執行結果需在回報或 PR evidence 中留痕。

## 1. Mandatory Read Set

讀完本文件後，AI 必須讀下列 canonical 文件，不得只靠提示詞記憶：

- `AGENTS.md`
- `CLAUDE.md`
- `.github/PULL_REQUEST_TEMPLATE.md`
- `docs/ai/codex-autonomous-workflow.md`
- `docs/ai/autonomous-pr-gates.md`
- `docs/pr-scope-policy.md`

若要修改 workflow / Scope Police / PR template，還要讀：

- `.github/workflows/pr-scope-police.yml`
- `.github/workflows/ci.test.mjs`

若 spec-injector tool behavior 本身不符合預期，先回報工具版本落差；不要在 tachigo PR 裡實作 spec-injector runtime。

## 2. Startup Readback

開始任何實作前，AI 必須先回報：

- `cwd`
- current branch
- dirty state
- target base branch, normally `develop`
- source issue URL / number
- duplicate issue / branch / PR readback
- spec-injector availability and gate status

如果任一項無法確認，先回報風險，不要假裝已完成前置作業。

## 3. spec-injector Local Setup

先檢查工具是否可用：

```bash
command -v spec
spec workflow-check --help
```

`spec workflow-check --help` 必須顯示 #242 之後的 flags：

- `--finding-disposition`
- `--threshold-evidence`
- `--pr`

若既有全域 `spec` 沒有這些能力，先用 canonical source 建立 local runner，不要直接改組員的全域環境：

```bash
export SPEC_INJECTOR_DIR="${SPEC_INJECTOR_DIR:-$HOME/dev/spec-injector}"

if [ ! -d "$SPEC_INJECTOR_DIR/.git" ]; then
  git clone https://github.com/Erick52106/spec-injector.git "$SPEC_INJECTOR_DIR"
fi

git -C "$SPEC_INJECTOR_DIR" pull --ff-only
pnpm --dir "$SPEC_INJECTOR_DIR" install --frozen-lockfile
pnpm --dir "$SPEC_INJECTOR_DIR" build
```

接著用 local runner 做能力檢查：

```bash
node "$SPEC_INJECTOR_DIR/dist/cli/index.js" workflow-check --help
```

若 local runner 可用但全域 `spec` 仍是舊版，本次工作優先使用 local runner，例如：

```bash
node "$SPEC_INJECTOR_DIR/dist/cli/index.js" workflow-check --repo . --phase start --issue <issue-number-or-url>
```

若 clone、pull、install、build 或 capability check 失敗，先停止並回報 `tool_gap=spec-injector #242 workflow-check unavailable`。可以用人工 checklist、`spec plan`、`spec validate` 作為暫時 evidence，但不得宣稱已跑完整 #242 workflow-check。

若 repo 缺 spec-injector config，AI 可以提議本機初始化：

```bash
spec init --repo .
```

初始化只允許 local-only。除非 source issue 明確要求並經 human review，否則不得把 `.spec-injector/` 或其輸出納入 commit / PR。

不得使用 `curl | bash`、`wget | sh`、`npx`、`pnpm dlx` 或未經 review 的任意安裝腳本取得 `spec-injector`。若環境政策禁止安裝依賴，保留 `blocked` evidence 並改走 manual checklist，不要偷渡產物。

## 4. Start Gate

如果任務已提供 issue number / URL：

```bash
spec plan <issue-number-or-url> --repo . --dry-run --format prompt --verbose
spec workflow-check --repo . --phase start --issue <issue-number-or-url>
```

若使用 local runner，將第二行改為：

```bash
node "$SPEC_INJECTOR_DIR/dist/cli/index.js" workflow-check --repo . --phase start --issue <issue-number-or-url>
```

如果尚未確定 issue：

1. 先用 GitHub metadata 搜尋既有 issue。
2. 讀候選 issue，確認 source of truth。
3. 若沒有合適 issue，先取得使用者對「建立 source issue」的明確授權；取得授權後再依 repo 規則建立。
4. 再跑上面的 `spec plan` 與 start gate。

`spec workflow-check` 若不可用，記錄 `tool_gap=workflow-check unavailable`，並以 `spec plan` output + manual AWP checklist 作為 start evidence。

## 5. Routing Plan Gate

跑完 start gate 後，先輸出 routing plan，等 routing 清楚後才實作。routing plan 至少包含：

- source issue
- 本 PR 明確不做
- 讀資料 / GitHub metadata / CI readback 的 worker
- 實作 worker
- 驗證 worker
- controller 保留的決策
- planned validation
- `controller_fallback_reason`, only if skipping worker/subagent

預設分工：

| Work type | Default route |
| --- | --- |
| GitHub issue / PR / branch / label / CI / review readback | `ops_spark` or equivalent low-cost worker |
| docs / PR template / workflow regression / narrow tests | GPT-5.4 worker |
| backend / frontend implementation | GPT-5.4 domain worker, with controller review |
| schema / migration / auth / wallet / ledger / money / permission model | GPT-5.5 controller or high-reasoning worker review |
| scope / architecture / review decision / merge gate | GPT-5.5 controller |

0-3 分鐘的單一 read-only trivial check 可以由 controller 做，但必須在 routing plan 或 PR body 寫明 `controller_fallback_reason`。

## 6. Implementation Rules

- 從最新 `develop` 開 branch。
- 保持 PR 對齊單一 source issue；額外想法開 follow-up issue / PR。
- 不把 docs / research draft 當成 implementation source of truth。
- worker 完成後，controller 必須 read back diff / tests / output，不直接相信摘要。
- 不需要的 worker session 要 close；若 stale handle / not found，記錄原因。

## 7. Commit Gate

commit 前執行：

```bash
spec workflow-check --repo . --phase commit --pr-body <path-to-pr-body> \
  --routing-evidence <path-to-routing-evidence.json> \
  --finding-disposition <path-to-finding-disposition.json> \
  --threshold-evidence <path-to-threshold-evidence.json>
git diff --check
```

這些 evidence JSON 是 local-only。PR body 只放 `status + ref`，不得把 full spec output、routing JSON、finding matrix、private context 或 task package commit 進 repo。

若 `spec workflow-check` 不可用，至少檢查：

- PR body 已從 `.github/PULL_REQUEST_TEMPLATE.md` 完整填寫。
- `Source of truth`、`Depends on PR`、`本 PR 明確不做` 已填。
- autonomous PR 已填 `Delegation Execution Log`、`Review conversation closeout`、`Final merge gate`。
- 沒有 staged `.spec-injector/`、spec output、private context 或 task package。
- commit message 包含 `refs #<issue-number>`。

## 8. PR Gate

開 PR 時：

- 未取得使用者對「開 PR / 公開遠端變更」的明確授權前，只能保留 local evidence / readback，不得建立 PR 或修改遠端 PR metadata。
- PR title 使用 repo prefix，例如 `[infra] ...`、`[discussion] ...`。
- PR body 必須從 `.github/PULL_REQUEST_TEMPLATE.md` 填，不使用自由格式。
- Initial PR body 要完整；尚未穩定欄位可填 `pending with reason` 或 `n/a`。
- `spec workflow-check evidence` 只放 `status + ref`；完整三段細節放 PR comment / issue comment / local-only evidence summary。
- normal non-autonomous PR 可填 `n/a`，不需要 AWP worker profile。

## 9. Review Triage Gate

Automated review finding 不能照單全收。每個 finding 先判斷：

- 是否仍對 latest head 成立。
- 是否 duplicate / outdated。
- 是否在 source issue scope 內。
- 是否需要 root-cause / state-model，而不是再補一個局部 patch。
- 是否讓 follow-up patch 超過原 PR diff 約 25-30%，需要拆 PR。

Final closeout 必須把 finding 分成：

- `adopted`
- `partial`
- `rejected`
- `deferred`

並在 PR body 的 `review_triage_ref`、`root_cause_gate_ref`、`finding_disposition_ref` 留 latest-head evidence ref。

## 10. Merge Gate

merge 前執行：

- 未取得使用者對「merge / 合併遠端狀態變更」的明確授權前，只能做 local evidence / readback，不得執行 merge 或替代性遠端合併操作。

```bash
spec workflow-check --repo . --phase merge --pr-body <path-to-pr-body> \
  --head-sha <latest-head-sha> \
  --pr <number-or-url> \
  --routing-evidence <path-to-routing-evidence.json> \
  --finding-disposition <path-to-finding-disposition.json> \
  --threshold-evidence <path-to-threshold-evidence.json>
```

若 `spec workflow-check` 不可用，至少 fresh readback：

- exact latest head SHA
- base branch and mergeability
- CI / required checks
- PR Scope Police
- review threads unresolved count
- CodeRabbit latest-head state
- chatgpt-codex-connector latest-head comment / review / reaction
- PR body final closeout

final closeout 只在狀態穩定後更新一次。ready-to-merge closeout 不可留下裸 `pending` 或 unresolved actionable finding。

## 11. Post-merge Closeout

merge 後：

- 若使用者已明確授權 closeout 公開狀態變更，在 source issue 留 proof comment。
- 若使用者已明確授權 closeout 公開狀態變更，close source issue as completed, unless it is intentionally long-lived tracking / ledger.
- 若本 PR 是 autonomous workflow dogfood，將 threshold calibration 短紀錄留在 #664 ledger issue comment；不要把 metrics 展開進 PR body 或專案檔案。
- 不刪 branch / worktree，除非使用者明確要求 cleanup。

## 12. First Reply Template

AI 讀完本文件後，第一輪回覆應包含：

```text
我已讀 docs/ai/autonomous-bootstrap.md，接下來會讀 mandatory read set、確認 repo 狀態、跑 spec-injector start gate，然後輸出 routing plan。

目前先做 read-only preflight：
- cwd:
- branch:
- dirty state:
- target base:
- source issue:
- duplicate PR / branch:
- spec-injector:
- controller_fallback_reason: <只有不派 worker 時才填>
```

如果任一必要資訊缺失，先補 readback 或回報 blocker，不要開始實作。
