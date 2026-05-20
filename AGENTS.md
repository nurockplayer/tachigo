# tachigo — Codex Agent Guidelines

## 語言設定

永遠使用台灣正體中文回覆，不得使用日文、韓文或簡體中文。

## RTK 使用規則

`rtk` 只用於 `git` 指令，降低 git 輸出 token。

範例：

```bash
rtk git --no-optional-locks status
rtk git --no-optional-locks diff -- AGENTS.md
```

非 `git` 指令不要加 `rtk`，例如 `rg`、`sed`、`make`、`go test`、`docker compose` 都直接執行。

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
- **開 PR 時必須以 `.github/PULL_REQUEST_TEMPLATE.md` 為起點**，不得自由格式撰寫：

  ```bash
  cp .github/PULL_REQUEST_TEMPLATE.md /tmp/pr_body.md
  # 填妥 /tmp/pr_body.md 所有欄位，不得留空或刪除 section
  make pr-open TITLE="[type] ..." BODY_FILE=/tmp/pr_body.md AUTO_READY=1
  ```

  Codex task PR 預設使用 `AUTO_READY=1`：PR 會以 draft 建立並加上
  `auto-ready` label，等 required checks 通過後再由 workflow 自動轉成
  Ready for review。非 Codex task 或長期 WIP draft 不應加 `auto-ready`。

### 操作權限邊界

- **Read-only** 可直接執行：讀檔、搜尋、PR / issue metadata、diff、CI 狀態、本機分析
- **狀態變更**必須先詢問並取得明確同意：Edit / 寫檔、commit、push、branch switch / rebase / merge、GitHub comment / review / CR / Approve / Merge、issue / PR 建立或編輯
- 若使用者在當輪訊息已明確要求修改檔案，視為已授權該次 Edit；但公開可見操作仍需再次確認
- 在 Codex sandbox 中執行 `git` 指令時，若目前環境支援且允許提權，必須第一時間使用提權執行；若政策禁止提權，照目前權限執行並如實回報限制
- 若 `git` 指令因未使用可用提權而失敗，優先視為 Codex 執行權限使用錯誤；不得歸因為 DNS、GitHub 網路或遠端服務問題

### Non-interactive 指令規則

- 不得執行 interactive commands；所有 `git` / `gh` 指令都必須是 non-interactive
- 若 `gh` 指令需要 auth / login / browser flow，立即停止並回報
- 執行 `git` / `gh` 指令前，必須先列出該步驟要執行的指令與目的
- 在 mixed worktree 中不得使用 `git add -A` 或 `git add .`

### PR Label

| Label | 用途 |
|---|---|
| `awaiting-review` | PR 等待 reviewer 審查或複查 |
| `changes-requested` | Reviewer 已提出 blocker，輪到作者修正 |
| `auto-ready` | Codex task draft PR 的 opt-in label；required checks 全綠後由 workflow 自動轉 ready |

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
- 供應鏈安全規則見 [docs/ai/supply-chain-security.md](docs/ai/supply-chain-security.md)：AI agent 不得自行新增依賴，不得執行 `npx` / `pnpm dlx` / `npm exec` / `curl | bash` / `wget | sh`，`package.json` 與 lockfile 改動必須在 PR 中說明並接受人工 review

### Autonomous Worker Profiles

使用者提到 Autonomous、AWP、Hybrid AWP、Codex autonomous workflow，或要求搭配 `spec-injector` 做 autonomous issue-first work 時，第一步先讀 [docs/ai/autonomous-bootstrap.md](docs/ai/autonomous-bootstrap.md)。這份文件是單一啟動入口；讀完後再依它展開必要文件、spec-injector start / commit / merge gate、routing plan 與 closeout。

當使用者授權 autonomous product work 時，Codex 作為總控 agent，負責架構、計劃、scope、最終 review、guarded merge 與 closeout。

可切分的探索、文件、測試、一般實作、GitHub readback、CI log 分析，可以依任務風險委派給 worker/subagent。routine GitHub / terminal / repo 探索優先使用 Spark 或較低推理成本的 worker；schema、migration、auth、wallet signature、points ledger、金流與權限模型必須由總控或高推理 worker 審查。

Autonomous workflow 的改善假設是「約 40% infra 本質複雜、約 60% 工作流自己製造摩擦」；總控應把可消除的 60% 交給 routing、closeout、worker lifecycle 與 follow-up split 規則處理，不得把 routine readback、comment/resolve、pre-commit/post-push checklist 長期留在總控身上。

autonomous work 一開始就必須先分派 worker，再進入計劃、開 issue、讀資料或實作；只有 `trivial/self-only exception` 可以不分派，但必須明寫原因。0-3 分鐘的單一 read-only trivial check 可由 controller 直接做，但必須記錄 `controller_fallback_reason`。

完整 worker profile、路由規則、review gate 與 PR Scope Police 合約見 [docs/ai/codex-autonomous-workflow.md](docs/ai/codex-autonomous-workflow.md)，Autonomous evidence gate、`spec workflow-check` start / commit / merge gate 與 local-only spec-injector 邊界見 [docs/ai/autonomous-pr-gates.md](docs/ai/autonomous-pr-gates.md)。

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
3. High blocker 快速掃，找到就停。重點類型：
   - CI 失敗
   - 危險 DB migration
   - auth bypass / wallet signature 驗證錯誤
   - replay attack 風險（nonce / expiration 缺失）
   - chain id 未驗證 / signer ownership 未確認
   - API breaking change
   - 無關 scope creep（scope 污染）
   - secrets 被 commit
   - binary / generated 垃圾檔
4. 依規模選深度：
   - 小 PR（diff < 150 行，且無 migration / auth / payment / wallet / API contract 風險）：Codex 可直審
   - 一般 PR：Gemini first-pass
   - 高風險 PR：Codex 深審，不依賴 Gemini
5. 輸出分級 findings（blocker / major / minor / nit）
6. 結尾必須明確詢問用戶：有 blocker → CR？無 blocker → merge？

DB migration 審查與語言別審查重點是上述 high blocker 快速掃的細化，只用來判斷既有風險類型是否存在；不得擴張成新的功能需求、重構要求或超出 issue scope 的 policy surface。

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

> **GitHub 自動 review（非 conversation 模式）**：有 blocker 時結語使用「Recommend changes requested」；無 blocker 時結語使用「Safe to merge after human approval」。這裡只規範 review 結語格式，不代表可以跳過公開 GitHub 操作前的使用者授權確認。

### DB Migration 審查

凡 PR 包含 migration 檔案，必須額外確認：

- 是否為 additive（新增欄位 / 表格），還是破壞性變更（drop / rename / type narrowing）
- NOT NULL 欄位是否有 backfill 或 default 值
- 大表 rewrite 是否評估 lock duration
- 有無 rollback 路徑
- 部署順序：schema → app 相容 → backfill → 切換讀寫 → cleanup

#### rewards / balances 相關表格額外注意

- double credit 風險（idempotency key 缺失）
- 數值精度（decimal precision）
- race condition（缺 row-level lock 或 transaction）

### 語言別審查重點

#### Go backend PR

- context 是否正確傳遞（未忽略 cancellation）
- goroutine 是否有 leak 風險
- transaction 是否正確 commit / rollback
- nil error 是否被遮蔽
- timeout 是否設定

#### Frontend PR（dashboard / extension）

- loading / error state 是否處理
- auth token 是否有洩漏風險（log、URL param）
- API 回傳型別與前端假設是否吻合
- 是否有未處理的 race condition（double submit、stale closure）

## 輸出格式

回報結果時保持精簡：
- 只列出關鍵變更（檔案名稱 + 一行說明），不貼完整 diff
- 測試結果只報 pass/fail 數量與失敗原因，不貼完整 log
- 遇到錯誤：先給出診斷與建議修法，再問是否繼續
