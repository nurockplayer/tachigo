# Autonomous Threshold Policy

本文件收斂 #664 autonomous threshold calibration ledger。#664 已累積足夠 dogfood datapoints；本 policy landing 後，停止對一般 autonomous PR 追加 routine threshold comment。只有出現新的 P0 workflow regression，例如錯誤 merge gate、遺漏 blocking review finding、或 Scope Police / spec-injector gate 明顯失效，才重新在 #664 或新 follow-up issue 留紀錄。

## Calibrated Rule

- Start gate 仍不可省略。Autonomous work 必須先讀 source issue、repo instructions、spec-injector start context，並做 routing decision；只有 routing decision 已判定符合 `trivial/self-only exception` 時，controller 才能不派 worker。
- 小型、單一 surface 的 docs / tests / controller-direct 修補可以走 `trivial/self-only exception`，但必須留下明確 `controller_fallback_reason`。理由要說明為什麼不需要 worker，而不是只寫 `docs only` 或 `small task`。
- Routine GitHub / CI / review / PR body readback 預設交給 `ops_spark` 或同級低成本 worker。包含 issue / PR metadata、status checks、CodeRabbit / connector 狀態、review-thread count、PR body 回填驗證與 post-push head SHA readback。
- implementation slice 只要不是 trivial/self-only，預設交給 bounded worker；worker scope 要有明確檔案範圍、驗證方式與不得擴 scope 的限制。
- controller 保留 final review、merge、scope、review finding disposition、scope-exception、schema / migration / auth / wallet / ledger / 金流 / 權限模型等高風險決策。

## Evidence Shape

- Routing evidence 必須包含 `routing_mode`、worker profile 或 `controller_fallback=true`、`controller_fallback_reason`、`delegation_outcome`、以及驗證/讀回 evidence ref。
- 若 `controller_fallback=allowed`，spawn directive 或 PR log 同一段必須補 `fallback_reason=<category>: <reason>; impact=<decision-or-blocker>; evidence=<url|sha|command>`。
- `fallback_reason` category 限定為 `trivial_self_only`、`worker_unavailable`、`tool_unavailable`、`rate_limited`、`conflicting_evidence`、`schema_or_data_risk`、`review_or_merge_decision`。
- 若 `ops_spark` 不可用，先改用同級低成本 profile，例如 `repo_scout` 或 `docs_worker`。只有低成本 worker 都不可用或任務已進入 controller-only decision，才由 controller 補位，且必須明寫原因。
- `threshold_ledger_ref` 可填 `not_needed`、`#664 policy applied`，或在 P0 workflow regression 時填實際 #664 / follow-up URL。
- PR body 只保留 `status + ref`。不要貼完整 threshold matrix、spawn metrics、CI rerun count、review-thread count 或個人工作量敘述。
- 不把 #664 metrics 加進 PR template、Scope Police、workflow schema、telemetry、daemon、dashboard 或任何 product runtime。

## spec-injector Boundary

- `spec plan <issue> --repo . --dry-run --format prompt --verbose` 與 `spec workflow-check` 是 local-only workflow evidence。
- tachigo PR body / issue comment 只保存 status/ref 摘要；完整 prompt、task package、`.spec-injector/` output、private context 與 generated artifact 不進 repo。
- `spec-injector` output 是 guardrail 與 context，不授權擴張 source issue scope。
