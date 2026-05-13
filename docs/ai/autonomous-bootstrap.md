# Autonomous Bootstrap

這份文件是 tachigo 的 AI-facing autonomous startup entrypoint。當使用者要求「Hybrid AWP with Explicit Fallback Gate + spec-injector」或要求 AI 讀單一啟動文件時，先讀本文件，再展開到 canonical docs。

本文件只負責啟動與路由，不取代既有規範。任何衝突以 `AGENTS.md`、PR template、scope policy 與 GitHub issue / PR metadata 為準。

## Canonical Read Set

啟動後必須讀回以下文件，不能只靠提示詞記憶：

1. `AGENTS.md`
2. `.github/PULL_REQUEST_TEMPLATE.md`
3. `docs/ai/codex-autonomous-workflow.md`
4. `docs/ai/autonomous-pr-gates.md`
5. `docs/pr-scope-policy.md`

若任務已有 source issue 或 PR，也必須讀該 issue / PR 的 body、comments、labels、checks、review state 與 linked context。

## Startup Sequence

1. 確認工作目錄、目前 branch、dirty state、target base branch。
2. 讀 source issue / PR metadata，先確認 scope、non-goals、是否已有 open PR。
3. 稽核目前 open Codex / automation PR、CI、CodeRabbit、review threads 與 blocker；有紅 CI、Change Request 或 quota/backoff 時，先修既有 PR，不開新 implementation PR。
4. 檢查或初始化 local-only spec-injector context；spec-injector 不可變成 repo artifact。
5. 產生 routing plan，經 controller scope 判斷後才開始實作。

## Local-Only Spec-Injector Boundary

`spec-injector` 是本機輔助，不是 repo runtime dependency，也不是不用此工具者的硬門檻。

禁止 commit：

- `.spec-injector/`
- spec output / task package
- private context
- local-only prompt bundle

未取得明確授權前，不得 mutation remote issue、PR、label、review、merge state。一般 readback 可直接做；公開 comment、review、approve、request changes、merge、close issue 皆需符合當輪授權與 repo policy。

## Required Gate Commands

開始前可執行：

```bash
spec validate --repo .
spec plan <issue> --repo . --dry-run --format prompt --verbose
spec workflow-check --repo . --phase start --issue <issue>
```

commit 前可執行：

```bash
spec workflow-check --repo . --phase commit --pr-body <path>
```

merge / ready-to-merge closeout 前可執行：

```bash
spec workflow-check --repo . --phase merge --pr-body <path> --head-sha <sha>
```

若 `spec` CLI 不可用、版本不支援、或 local-only context 不存在，記錄 tool gap 與人工替代 checklist；不得把缺工具誤寫成 gate 通過。

## Routing Plan Required

implementation 前必須先產生 routing plan，至少包含：

- Source issue / PR。
- 風險分類：docs、frontend、backend、workflow、schema、auth、payments、wallet、deploy / CI policy 等。
- 預計變更檔案與 write scope。
- worker profile、model、reasoning、任務邊界。
- controller 保留的決策：scope、architecture、security、merge gate、public review action。
- 驗證計畫：local focused tests、CI / CodeRabbit readback、review-thread closeout。
- follow-up split 判斷：哪些內容明確不做，哪些另開 issue / PR。

只有 `trivial/self-only exception` 可以不分派 worker。使用例外時必須明寫：

- `controller_fallback_reason`
- 為什麼是 0-3 分鐘或 bounded docs-only / metadata-only scope
- 為什麼不需要 parallel worker
- 如何驗證與 closeout

## PR Body Requirements

Codex autonomous PR 必須從 `.github/PULL_REQUEST_TEMPLATE.md` 開始填，不得自由格式撰寫。

PR body 至少要可讀回：

- `Source of truth`
- `Depends on PR`
- `本 PR 明確不做`
- `Delegation Execution Log`
- `Review conversation closeout`
- `Final merge gate`
- `spec workflow-check evidence` 或人工替代 checklist

剛開 PR 時尚未穩定的欄位可填 `pending with reason`。ready-to-merge closeout 不可留下裸 `pending`。

## Review And Merge Boundary

merge 前必須 fresh readback：

- latest head SHA
- PR Scope Police
- required checks / CI
- CodeRabbit latest-head 狀態
- chatgpt-codex-connector comment / review / reaction
- unresolved review threads
- 每個 actionable finding 的 adopted / partial / rejected / deferred disposition

CodeRabbit quota / rate-limit / usage-exceeded / backoff 時，不要 flood 新 PR 或重複要求同一 head review；改做 CI 修復、review finding 修復、issue triage、local verification 或 summary。

不得自動 merge，除非使用者在當輪明確授權且 repo policy 允許。不得自我 approve 同一 automation workstream 的 PR。

## Non-Autonomous PRs

一般非 autonomous PR 不需要使用本 bootstrap，也不需要填 worker profile。若 PR template 的 autonomous 欄位不適用，可填 `n/a` 或人工 checklist；不要為了符合 autonomous gate 而虛構 worker、spec evidence 或 review closeout。
