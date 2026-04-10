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

## 輸出格式

回報結果時保持精簡：
- 只列出關鍵變更（檔案名稱 + 一行說明），不貼完整 diff
- 測試結果只報 pass/fail 數量與失敗原因，不貼完整 log
- 遇到錯誤：先給出診斷與建議修法，再問是否繼續

## PR Review 最小化規則

目標：降低 Codex 在 PR review 的 token 使用量，同時維持 blocker 導向的審查品質。

### Review policy

- 預設先做輕量 review：先看 PR 描述、changed files、checks、未解 review thread，再決定是否深入讀碼
- 優先找 blocker、regression、scope violation、缺失的必要測試；不要先做全面 code walkthrough
- 若前一輪 blocker 已解除，優先更新 merge 判斷，不重做整份 review

### Scope control

- 只審查 PR changed files 與直接相關的最小必要上下文
- 不主動延伸到未變更檔案，除非：
  - 這次改動明確依賴該檔案行為
  - review comment 指向跨檔案整合風險
  - 需要驗證是否真的有 regression
- 對 scope 外議題，直接標示為 follow-up issue / wrong PR，不展開長篇分析

### Output constraints

- findings 優先，摘要次之
- 單次 review 以 3 個 finding 為上限；若沒有 blocker，直接明講 `no blocker found`
- 每個 finding 只寫：
  - 檔案 / 位置
  - 風險一句
  - 為什麼構成 blocker 或 non-blocker 一句
- 不重述 PR 全貌，不貼大段 diff，不列與 merge 無關的觀察清單

### Avoid unnecessary explanation

- 不解釋顯而易見的程式碼流程
- 不為了展現覆蓋率而列出已檢查但無問題的檔案
- 不重複前一輪已成立的結論，除非新 commit 改變判斷
- 對 minor / nit / future work，除非使用者明問，否則一句帶過即可

### Escalate deeper review when

- PR 涉及 auth、權限、金流、私鑰、contract interaction、schema migration、刪資料風險
- CI fail、review thread 指向實際行為回歸、或 PR 描述與 diff 明顯不一致
- changed files 雖少，但跨 service boundary、環境設定、部署流程或 release 流程
- 使用者明確要求 full review / deep dive / second pass
