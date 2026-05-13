## 什麼改動
<!-- 簡單說明做了什麼 -->

## 為什麼
<!-- 背景、需求，或關聯的 issue（e.g. closes #123） -->

## Release Context
<!-- 若這是正式 release PR，請填：`develop -> main`；否則填 `n/a` -->
- Release type：<!-- `develop -> main` 或 `n/a` -->

## Workflow Type
<!-- Codex task PR 請勾選；非 Codex task 填 n/a -->
- [ ] Small / Medium
- [ ] Test-Driven / Debug Loop
- [ ] Architecture / High Risk

## Scope 對齊
<!-- 一般 feature PR 若超過約 35 個檔案、diff 過大、同時改多個 product surface，或依賴尚未 merge 的 backend contract，會被 PR Scope Police 自動擋下。正式 `develop -> main` release PR 請使用 `[release]` title prefix，並在上方 Release Context 填 `develop -> main`。 -->
- Source of truth：<!-- issue / PR / 文件，例如 #115 -->
- Depends on PR：<!-- `none` 或 `#123` -->
- Backend contract already in develop:
  <!-- 必填：只要不是純 docs/template/metadata PR 都要勾。改到 `.github/workflows/**` 時也不是 metadata-only，仍需勾選。 -->
  - [ ] yes
  - [ ] no
- If no, this PR is:
  - [ ] stacked on dependency branch
  - [ ] intentionally blocked until dependency merges
- 本 PR 是否完全在 source of truth 範圍內？
  - [ ] 是
  - [ ] 否，已另開 issue / PR 處理超出部分
- 本 PR 明確不做：
  - <!-- 例：不做 future panels / 不做 router / 不做 dashboard UI -->

## Delegation Execution Log（非 Codex autonomous PR 可略過）
<!-- 一般 PR 不需要 worker profile。若是 Codex autonomous PR，請對照 source issue 的 delegation plan，完整填寫實際執行。 -->
- Source issue delegation plan：
  - <!-- 例：#620 的 Issue Delegation Plan -->
- Actual worker profile(s)：
  - <!-- 例：controller / docs_worker / ops_spark -->
- Model strength：
  - <!-- 例：controller = high；docs_worker = medium -->
- Spawn directive(s)：
  - <!-- 例：profile=ops_spark model=gpt-5.3-codex-spark reasoning=medium controller_fallback=denied fallback_reason=n/a -->
- Verification evidence：
  - <!-- 例：git diff --check；node --test .github/workflows/ci.test.mjs -->
- Self-review / exception reason：
  - <!-- 例：已完成 self-review；或 trivial/self-only exception reason -->
- Worker session closeout：
  - <!-- 例：已讀回 worker 結果，不需追加任務的 worker session 已 close；stale handle / not found 已說明 -->
- Workflow friction / follow-up split：
  - <!-- autonomous PR 請說明本次約 40% infra 複雜 / 約 60% 工作流摩擦中，哪些已由 worker routing / closeout / follow-up issue 收斂；threshold calibration 只留 #664 ledger comment URL 或 n/a，非 autonomous PR 可填 n/a -->

## Review conversation closeout
<!-- autonomous PR 必填；非 autonomous PR 可填 n/a。所有 review finding 必須修正並回覆/resolve，或留下技術理由後 resolve。 -->
- Unresolved threads：
  - <!-- 例：0；若非 0，列 blocker 與下一步 -->
- CodeRabbit：
  - <!-- 例：reviewed latest head；rate limit fallback；n/a -->
- chatgpt-codex-connector：
  - <!-- 例：commented latest head；reaction-only fallback；n/a -->
- review_triage_ref：
  - <!-- autonomous PR：latest-head review triage evidence ref（issue / PR comment / spec output ref）；非 autonomous PR 可填 n/a -->
- root_cause_gate_ref：
  - <!-- autonomous PR：同概念第二次 finding 時的 root-cause / state-model evidence ref；若尚未觸發可填 latest-head rationale ref；非 autonomous PR 可填 n/a -->
- finding_disposition_ref：
  - <!-- autonomous PR：adopted / partial / rejected / deferred disposition evidence ref；非 autonomous PR 可填 n/a -->
- Final reviewer action：
  - <!-- 例：all findings fixed/resolved；adopted / partial / rejected / deferred recorded at latest head；n/a -->

## Final merge gate
<!-- autonomous PR merge 前更新一次即可；PR initial body 請先填 pending/n/a，final closeout 只在狀態穩定後更新一次。 -->
- Ready-to-merge decision：
  - <!-- 例：ready / blocked by CI / n/a；autonomous ready-to-merge closeout 不可留下裸 pending -->
- Latest head SHA：
  - <!-- 例：abc1234；開 PR 前可填 n/a -->
- Required checks：
  - <!-- 例：PR Scope Police pass；CI pass；開 PR 前可填 pending with reason -->
- spec workflow-check evidence：
  - <!-- status + ref only：使用 spec-injector 者填 local-only gate status 與 evidence ref；未使用者填 n/a 或人工 checklist；完整三段細節留在 Evidence URL 或 review/issue comment，不塞 PR body -->
- Evidence URL：
  - <!-- 例：final closeout comment / CI run / n/a -->

## PR 拆分檢查
<!-- 一個 PR = 一個可獨立理解、可獨立驗證的行為變更。若超過 400 行、同時改 backend/frontend、或同時包含 schema/service/handler/UI 任兩種以上，請優先拆 PR。 -->
- 這個 PR 的單一句子目的：
  - <!-- 例：新增 points ledger migration -->
- Approx changed lines：<!-- 例：~250；若超過 400 行請說明為什麼不拆 -->
- 本 PR 是否可獨立 review，不需要理解未合併的其他 PR？
  - [ ] 是
  - [ ] 否，原因：
- 本 PR 是否同時包含以下多個層級？
  - [ ] migration / schema
  - [ ] backend domain service
  - [ ] API handler / router
    - [ ] 已執行 `swag init` 並將 `services/api/docs/` 變更一起 commit
  - [ ] frontend integration
  - [ ] tests
  - [ ] docs
  - [ ] refactor / cleanup
- 若勾選兩項以上，為什麼這些變更需要放在同一個 PR？
  - <!-- 若無請填 n/a -->
- 若已拆分，相關 PR：
  - <!-- 例：#123 migration, #124 service；若無請填 n/a -->

## Acceptance Criteria
<!-- 對應 source issue 的 AC，逐條勾選；非 Codex task 填 n/a -->
- [ ] AC 1
- [ ] AC 2
- [ ] AC 3

## 超出範圍內容
<!-- 若有額外重構、future work、design exploration、順手修正，請寫在這裡；若無請填「無」 -->

## 測試方式
- [ ] 本地測試過
- [ ] 有寫 / 更新測試

**驗證結果**
<!-- 貼上關鍵指令的執行結果或摘要；若無測試填 n/a -->
```
（貼上驗證輸出）
```

## 備註
<!-- 其他需要 reviewer 注意的事情（可留空） -->

## Notes for Claude Code Review
<!-- Codex 填寫：有哪些地方需要 Claude 特別注意、不確定的假設、已知風險 -->
- 
