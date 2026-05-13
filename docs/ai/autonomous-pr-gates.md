# Autonomous PR Gates

本文件是 tachigo 的 Autonomous Worker Profiles v2 evidence discipline。它補足 `AGENTS.md`、`docs/ai/codex-autonomous-workflow.md`、`.github/PULL_REQUEST_TEMPLATE.md` 與 `PR Scope Police` 之間的共同語彙。

## 成本感知分派

- `ops_spark` / Spark：GitHub issue / PR metadata、CI / check readback、duplicate PR / branch 檢查、review-thread / CodeRabbit / connector 狀態讀回。
- GPT-5.4 worker：窄範圍 docs、template、workflow、tests、一般實作；需在 bounded write set 內完成。
- GPT-5.5 controller：scope、架構、review decision、final merge gate。controller 不應長期承擔 routine GitHub / CI / readback。

`Delegation Execution Log` 的 spawn directive 必須可讀出 `profile=`, `model=`, `reasoning=`, `controller_fallback=`。若 `controller_fallback=allowed`，同一行必須有非空 `fallback_reason=` 或 `controller_fallback_reason=`。`ops_spark` 類任務不得使用高階 controller model，除非同一行留下 fallback reason。

## Delegation Threshold

0-3 分鐘、單一 read-only trivial check 可由 controller 直接做，但必須記錄 `controller_fallback_reason`。以下任務預設委派給 Spark / 低成本 worker：

- GitHub issue / PR / branch duplicate readback。
- CI / Scope Police / review-thread / CodeRabbit / connector readback。
- PR body、issue checklist、metadata-only repair 的資料蒐集。

涉及 schema、auth、wallet、金流、points ledger、merge decision、review finding 採納與否時，controller 保留 final decision。

## PR Initial Body And Final Closeout

PR body 一開始就要完整合規，包含 Source of truth、Depends on PR、本 PR 明確不做、Delegation Execution Log、Review conversation closeout、Final merge gate。開 PR 時尚未穩定的欄位可填 `pending with reason` 或 `n/a`，但 ready-to-merge closeout 不可留下裸 `pending`。

final closeout 只在 merge 前狀態穩定後更新一次，避免每個 CI tick 都修改 PR body。final closeout 至少讀回 latest head SHA、CI / Scope Police、review threads、CodeRabbit、chatgpt-codex-connector、每個 finding 的採納或不採納狀態。

## Review Fallback Policy

- CodeRabbit skipped / rate limit：不得對同一 head 重複要求 review；改由 controller self-review，並留下 rate-limit / skipped 證據與驗證結果。
- chatgpt-codex-connector timeout：先讀 comment / review / reaction；reaction-only 可視為 fallback evidence，但需記錄 latest head。
- 所有 review finding 必須修正並回覆/resolve，或留下技術理由後 resolve。

## spec workflow-check Gates

`spec-injector` 對 tachigo 是 local-only 輔助，不是不用此工具者的硬性門檻。

- Start gate：使用者要求或任務適用時，可本機執行 `spec plan <issue> --repo . --dry-run --format prompt --verbose` 或未來 `spec workflow-check --repo . --phase start`，用於產生 bounded context。
- Commit gate：commit 前可本機執行 workflow-check，確認 PR body / non-goals / validation / `.spec-injector/` 未 staged。
- Merge gate：merge 前可本機執行 workflow-check，確認 final closeout、unresolved thread count、latest head SHA 與 spec gate evidence。

不得 commit `.spec-injector/`、spec output、private context、或工具產生的 task package。未使用 spec-injector 的 PR 可在 template 填人工 checklist / `n/a`。

## Scope Police Contract

Autonomous evidence gate 只對 autonomous PR 嚴格啟用。判定方式包含 `Delegation Execution Log` 有正式欄位內容，或 label / branch / body 明確標示 autonomous / codex workflow。

Scope Police sticky comment 應顯示 `Autonomous evidence snapshot`，讓 reviewer 快速看見 delegation log、spawn directives、controller fallback、review closeout、final merge gate 與 pending 是否清乾淨。
