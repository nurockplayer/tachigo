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

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
```

- 實作過程中的 commit 用 `refs #號碼`
- PR 的最後一個 commit 或 PR 描述用 `closes #號碼`

Type：`feat` / `fix` / `docs` / `chore` / `refactor` / `test`

### 注意事項

- **不要** 直接推 `main`，PR 目標分支是 `develop`
- `git commit` / `git push` / `git checkout -b` 由 Claude Code 執行（sandbox 對 `.git` 無寫入權限）
- `gh` 指令（issue、PR、API）由你執行

## 輸出格式

回報結果時保持精簡：
- 只列出關鍵變更（檔案名稱 + 一行說明），不貼完整 diff
- 測試結果只報 pass/fail 數量與失敗原因，不貼完整 log
- 遇到錯誤：先給出診斷與建議修法，再問是否繼續
