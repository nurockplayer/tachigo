# tachigo — Codex Agent Guidelines

## 語言設定

永遠使用台灣正體中文回覆，不得使用日文、韓文或簡體中文。

## 你的角色

你同時承擔兩種模式，依任務性質切換：

- **探索 / 分析**：快速摸清程式結構、收斂問題原因、評估方案可行性，輸出精簡摘要給 Claude Code 決策
- **執行**：實際寫程式、改檔案、跑測試、執行指令，依照 Claude Code 給的計畫做

收到任務時判斷是哪種模式，若不確定就直接問。遇到需要架構決策的岔路，先回報給 Claude Code，不要自行決定。

## 專案結構

```
tachigo/
├── backend/          # Go API (Gin + GORM + PostgreSQL)
├── tachimint/        # Twitch Extension 前端 (React + TypeScript)
├── dashboard/        # 後台管理介面 (React + TypeScript) ← 建置中
└── docs/             # 設計文件
```

架構細節見 [docs/architecture.md](docs/architecture.md)。

## 開發指令

```bash
make dev    # 啟動所有服務（hot reload）
make down   # 停止所有服務

# 執行後端測試
docker compose run --no-deps --rm app go test ./...
```

## Git 規範

### Branch 命名

`<type>/<short-description>`

例：`feat/points-service`、`fix/bits-receipt`、`docs/architecture`

### Commit 訊息格式

```
<type>: <short description>

refs #<issue號碼>

Co-Authored-By: Codex <codex[bot]@openai.com>
```

- 實作過程中的 commit 用 `refs #號碼`
- PR 的最後一個 commit 或 PR 描述用 `closes #號碼`

Type：`feat` / `fix` / `docs` / `chore` / `refactor` / `test`

### Issue 對應策略

尋找 commit / PR 對應 issue 時，預設使用省 token 路線：

1. 先用 `gh issue list` / `gh search issues` 取得 issue metadata。
2. 若候選很少（約 0-5 個），由 Codex / Claude 直接判斷。
3. 若候選很多、搜尋詞不明確、或 backlog 很亂，交給 Gemini CLI 排序候選 issue。
4. Gemini 只負責產出最多 3 個候選 issue 與理由；Codex / Claude 必須用 `gh issue view` 驗證最終選擇。
5. 若沒有合適 issue，開新 issue；不得為了符合 commit 格式硬套不相關 issue。

### 注意事項

- **不要** 直接推 `main`
- 日常 feature PR 目標分支是 `develop`
- 正式 release 依 Git Flow 由 `develop` 開 PR 到 `main`
- 目前暫不使用 `release/*` branch
- 未來若有正式部署、freeze window、hotfix/backport 需求，再升級 release 流程
- GitHub 相關的 `gh` 指令（issue、PR、API）與必要的 `git` 指令可由你執行
- 執行 `git` 時仍需遵守 branch / commit / scope 規範，不得繞過 PR 流程

### PR Label

| Label | 用途 |
|---|---|
| `needs-codex-review` | PR 有新 commit，輪到 Codex 重新審查 |
| `changes-requested` | Codex 已提出 blocker，輪到作者修正 |

## Scope 邊界

禁止 scope pollution：不要把 issue 沒有明確要求的內容混進同一個 PR。

### 基本規則

- PR 只應包含該 issue 明確列出的任務、規格與完成條件
- 若在實作途中發現額外想做的功能、重構、future work、design exploration，必須另開 issue / PR，不可順手一起提交
- docs / research draft 不能自動視為 implementation source of truth；只有被明確指定的 issue / PR / 文件，才能作為當前實作依據

### 常見禁止情況

- issue 只要求 migration，PR 卻同時加入 service、handler、router、前端串接
- 本輪 MVP 只要求單一畫面，PR 卻順手加入 future panels、bottom nav、完整 design system
- 修 bug 時順便重構整個模組，且未經事前同意
- backend issue 混入 dashboard / tachimint UI 改動，反之亦然

### 遇到岔路時怎麼做

- 如果額外內容是必要前置條件：先回報 Claude Code，說明為什麼原 issue 缺這一塊，再決定是否調整範圍
- 如果額外內容不是必要前置條件：先記錄成新的 issue / TODO，不要混進目前 PR
- 若目前 PR 已經超出 issue 範圍，應主動建議拆 PR 或縮回原範圍

## AI 協作守則

若貢獻內容主要由 AI 產生，必須額外遵守以下規則：

- 不得讓 AI 自行擴張 issue scope；AI 提出的額外功能、future work、重構建議，必須拆成獨立 issue / PR
- 不得把 docs / research draft / brainstorming 內容直接當成 implementation source of truth，除非 repo 已明確指定
- 不得未經驗證就宣稱「已完成」；至少要回報實際執行過的測試、未驗證部分、以及已知風險
- reviewer 應優先檢查 AI 是否偏離 issue、腦補需求、混入未要求的 schema / API / UI 改動，而不是只看程式碼表面是否完整

## PR Review Strategy

Terminology:
- `Review` = code review.
- `CR` = change request / requested changes, not code review.

預設使用省 token 路線：metadata-first + Gemini CLI first pass + Codex validation。
除非使用者明確要求「不要用 Gemini」或 PR 很小，否則不要讓 Codex 一開始就完整讀整張 diff。

1. Load PR metadata first:
   - linked issue
   - changed files
   - diff stat
   - CI status
   - test coverage signals
   - existing PR comments / reviews

2. Decide review depth:
   - Small PR: Codex may review directly.
   - Normal / large PR: use Gemini CLI for first-pass scanning before Codex opens patches.
   - Auth / payment / security / migration / production-risk PR: Gemini may help summarize, but Codex must do the final deep review itself.
   - If Claude Code, CodeRabbit, or another reviewer already commented, Codex should first validate those findings instead of restarting a full review.

3. Use Gemini CLI for initial low-cost review:
   - repo rule compliance, especially scope boundaries
   - possible bugs
   - edge cases
   - incorrect logic
   - scope pollution
   - git history consistency
   - performance concerns
   - missing tests for risky changes

   Gemini is a scanner only. Codex keeps final judgment.

4. Gemini review prompt:
   Review the PR metadata and diff.

   Focus on:
   - repo rule compliance, especially scope boundaries and review/CR terminology
   - likely bugs
   - scope pollution against linked issue / PR title / repo rules
   - edge cases
   - incorrect logic
   - git history consistency against commit messages and incremental changes
   - performance issues
   - missing tests for risky changes

   Rules:
   - prioritize changed lines
   - use unchanged context only when needed
   - return at most 5 high-confidence findings total across `findings` and `scope_pollution`
   - ignore purely stylistic comments unless they affect correctness or maintainability
   - omit findings with confidence below 70
   - every finding must include file path and concrete evidence
   - output concise JSON only

   Return JSON:
   - summary
   - risk_level: low / medium / high
   - findings: [{title, file, evidence, why_it_matters, confidence}]
   - scope_pollution: [{file, evidence, reason, confidence}]
   - files_to_inspect_first

   This schema is for Codex's repo-level Review workflow. It is not the same
   contract as Claude Code's local `/code-review` script, which may return a
   flat issue array for its own command pipeline. Claude Code's local script
   documents 4 dimensions (`CLAUDE.md` compliance, bugs, git history, code
   comments); Codex's repo-level Review uses the broader focus list above and
   validates final findings itself.

   If Gemini CLI is unavailable, skip the external-model pass and use Codex metadata-first triage before reading patches.
   This fallback applies only to Codex's repo-level Review flow; it does not change or override Claude Code's local `/code-review` marker fallback behavior.

5. Split only the necessary PR diff into logical chunks:
   - group related files when behavior crosses file boundaries
   - prefer small chunks for large diffs
   - include only necessary unchanged context

6. Summarize Gemini findings:
   - merge duplicate issues
   - discard vague or non-actionable comments
   - keep only blockers, likely regressions, and meaningful test gaps

7. Use Codex for validation:
   - validate which Gemini findings are real
   - identify false positives
   - refine suggested fixes
   - check for missing critical issues
   - inspect minimal necessary patch context only when summary is insufficient

8. Avoid using Codex on the full diff unless necessary.

9. Generate the final PR review comment:
   - group by severity: high / medium / low
   - include actionable suggestions only
   - avoid posting unverified Gemini findings

## 輸出格式

回報結果時保持精簡：
- 只列出關鍵變更（檔案名稱 + 一行說明），不貼完整 diff
- 測試結果只報 pass/fail 數量與失敗原因，不貼完整 log
- 遇到錯誤：先給出診斷與建議修法，再問是否繼續
