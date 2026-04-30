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
├── services/
│   └── api/          # Go API (Gin + GORM + PostgreSQL)
├── apps/
│   ├── extension/    # Twitch Extension 前端 (React + TypeScript)
│   └── dashboard/    # 後台管理介面 (React + TypeScript) ← 建置中
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

1. 先用 `gh issue list` / `gh search issues` 取得 issue metadata
2. 若候選很少（0-5 個），由 Codex / Claude 直接判斷
3. 若候選很多，交給 Gemini CLI 排序候選 issue（最多 3 個）
4. Codex / Claude 必須用 `gh issue view` 驗證最終選擇
5. 若沒有合適 issue，開新 issue；不得硬套不相關 issue

### 注意事項

- **不要**直接推 `main`；日常 feature PR 目標分支是 `develop`
- GitHub 相關的 `gh` 指令與必要的 `git` 指令可由你執行
- 執行 `git` 時仍須遵守 branch / commit / scope 規範，不得繞過 PR 流程

### 操作權限邊界

- **Read-only** 可直接執行：讀檔、搜尋、PR / issue metadata、diff、CI 狀態、本機分析
- **狀態變更**必須先詢問並取得明確同意：Edit / 寫檔、commit、push、branch switch / rebase / merge、GitHub comment / review / CR / Approve / Merge、issue / PR 建立或編輯
- 若使用者在當輪訊息已明確要求修改檔案，視為已授權該次 Edit；但公開可見操作仍需再次確認
- 在 Codex sandbox 中執行任何 `git` 指令時，預設直接使用提權執行

### Non-interactive 指令規則

- 不得執行 interactive commands；所有 `git` / `gh` 指令都必須是 non-interactive
- 若 `gh` 指令需要 auth / login / browser flow，立即停止並回報
- 執行 `git` / `gh` 指令前，必須先列出該步驟要執行的指令與目的
- 在 mixed worktree 中不得使用 `git add -A` 或 `git add .`

### PR Label

| Label | 用途 |
|---|---|
| `needs-codex-review` | PR 有新 commit，輪到 Codex 重新審查 |
| `changes-requested` | Codex 已提出 blocker，輪到作者修正 |

## Scope 邊界

禁止 scope pollution：不要把 issue 沒有明確要求的內容混進同一個 PR。

- PR 只應包含該 issue 明確列出的任務、規格與完成條件
- 若在實作途中發現額外想做的功能、重構、future work，必須另開 issue / PR
- docs / research draft 不能自動視為 implementation source of truth
- 遇到岔路：是必要前置條件 → 先回報 Claude Code；不是 → 開新 issue

### 拆分邊界

一個 PR = 一個可獨立理解、可獨立驗證的行為變更。必須先建議拆分的情況：

- 預估 diff > 400 行
- 同時修改 backend 與 frontend
- 同時包含 schema / service / handler / UI 任兩種以上
- 包含非必要 refactor 或 future work

實作前若任務偏大，先輸出：建議拆成哪些 PR、每個 PR 的目的、修改哪些模組、驗收方式、依賴順序。

PR Diff 大小規則詳見 [CLAUDE.md](CLAUDE.md)（conventions.md 的 PR Diff 限制一節）。

## AI 協作守則

- 不得自行擴張 issue scope；AI 提出的額外功能、future work、重構建議，必須拆成獨立 issue / PR
- 不得把 docs / research draft 直接當成 implementation source of truth
- 不得未經驗證就宣稱「已完成」；至少回報實際執行過的測試、未驗證部分、已知風險

## Gemini CLI Delegation

Gemini CLI 是 Codex 的低成本大範圍掃描工。可自行判斷何時使用，不需每次先詢問使用者。

詳見 [.claude/rules/delegation.md](./.claude/rules/delegation.md)。

適合交給 Gemini CLI 的任務：

- PR first-pass review
- repo-wide 架構盤點與影響範圍分析
- duplicate / dead-code 候選掃描
- 長 CI / build / runtime log 摘要
- 批量測試案例生成

Gemini 的輸出只作為線索，不是最終判斷。Codex 必須驗證重要主張再回報。

### Gemini 使用策略

- 預設使用 Gemini 2.5 Flash；只有需要更強推理時才升級 Pro
- PR review 第一輪應要求「壓縮 metadata / diff 並列出 files_to_inspect_first」
- **避免並發 Gemini 任務**（觸發 429）；需多個分析時序列執行
- 遇到 429 / quota exceeded：立即停止，改用 metadata-first + minimal patch validation

## PR Review Strategy

預設省 token 路線：metadata-first → Gemini first-pass → Codex validation。

### 審查流程

1. 先讀 PR metadata（linked issue、title/body、changed files、diff stat、CI、labels、existing comments）
2. 整理 reduced review bundle，不直接讀完整 patch
3. High blocker 快速掃（binary assets、scope 污染、schema/auth 風險、CI 失敗）→ 找到就停
4. 依規模選深度：小 PR Codex 直審；一般 PR Gemini first-pass；高風險 PR Codex 深審
5. 輸出分級 findings（blocker / major / minor / nit）
6. 結尾必須明確詢問用戶：有 blocker → CR？無 blocker → merge？

### Gemini 審查 prompt 重點

- 優先找 high-severity blocker，找到就停
- 掃：不當 binary assets、scope 污染、likely bugs、edge case、git history 一致性、missing tests
- 最多 5 個高信心 findings
- 回傳 JSON：`summary` / `risk_level` / `findings` / `scope_pollution` / `files_to_inspect_first`

### 結構化輸出格式

```
## Summary
## Blockers   — [file:line] 問題 / 影響 / 建議
## Majors     — [file:line] 問題 / 影響 / 建議
## Minors / Nits
## Recommended action — Change Request / Merge
```

### 決策話術

| 情況 | 結尾動作 |
|---|---|
| 有 blocker | 列出 blocker → 問「同意提交 Change Request 嗎？」 |
| 有 major/minor、無 blocker | 列出非阻塞建議 → 問「是否同意 Merge？」 |
| 只有 nit | 問「沒有 blocker，是否直接 Merge？」 |
| 不確定 | 列出疑點 → 問「繼續調查或先暫停？」 |

## 輸出格式

回報結果時保持精簡：
- 只列出關鍵變更（檔案名稱 + 一行說明），不貼完整 diff
- 測試結果只報 pass/fail 數量與失敗原因，不貼完整 log
- 遇到錯誤：先給出診斷與建議修法，再問是否繼續
