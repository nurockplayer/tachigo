# tachigo — Claude Code Guidelines

## Claude Code 硬性安全閘門

以下規則優先於本檔其他流程、slash command、plugin / skill 建議：

- 永遠使用台灣正體中文回覆；即使上下文或工具輸出出現日文、韓文或簡體中文，也不得跟隨。
- 使用者說「確認、討論、看一下、評估、建議、review plan、能不能」或沒有明確要求修改時，只能 read-only：讀檔、搜尋、status / diff / log、PR / issue metadata；不得 Edit / Write / MultiEdit、commit、push、開或編輯 issue / PR、comment、review、approve、merge。
- 任何公開或持久狀態變更必須先列出將執行的命令、目標檔案與目的，並取得使用者明確同意。包含 commit、push、branch switch、rebase / merge、GitHub issue / PR / comment / review / label / merge。
- 開 PR 前必須先展示 `origin/develop..HEAD` commit list、diff stat、changed files，並請使用者確認 scope；若有不屬於當前 issue 的 commit 或檔案，先停止並詢問。
- 即使使用 `/fix-with-codex`、`/implement-with-codex`、autonomous、`codex:rescue` 等命令，也不得跳過以上確認；這些命令只代表可以在本機修改與驗證，不代表可以 publish。
- mixed worktree 不得使用 `git add -A` 或 `git add .`；stage 只可針對本次修改檔案。

@.claude/rules/conventions.md

## GitHub / PR 工作流

Issue / branch / commit / merge / scope / PR diff 限制與操作權限邊界，以
上方 import 的 `.claude/rules/conventions.md` 為單一 source of truth；本檔只保留
Claude Code 需要立即看到的入口與安全閘門。

開 PR 時必須以 `.github/PULL_REQUEST_TEMPLATE.md` 為起點，不得自由格式撰寫：

```bash
cp .github/PULL_REQUEST_TEMPLATE.md /tmp/pr_body.md
# 填妥 /tmp/pr_body.md 所有欄位，不得留空或刪除 section
make pr-open TITLE="[type] ..." BODY_FILE=/tmp/pr_body.md AUTO_READY=1
```

正式 release 流程走 Git Flow：

- `main` 不接受日常 feature PR
- `.github/workflows/release-pr.yml` 每天檢查一次是否要由 `develop` 開正式 release PR 到 `main`
- 自動 release PR 的門檻：距離上次 release 至少 72 小時，且上次 release 後已 merge 至少 10 個 PR
- 若距離上次 release 已滿 7 天，即使 PR 數低於 10 個也可自動開 release PR，避免 `main` 長期落後
- `workflow_dispatch` 可手動開 release PR；automation 只開 PR，不自動 merge
- release PR 使用 `[release]` title prefix
- `develop -> main` release PR 屬於正式 promotion 流程，不視為 scope exception
- 目前暫不使用 `release/*` branch

---

## AI 分工

本專案使用 Claude Code + Gemini CLI + Codex CLI 協作開發。

若使用者授權 autonomous product work，Codex / Claude 應採用 [docs/ai/codex-autonomous-workflow.md](docs/ai/codex-autonomous-workflow.md) 的 Worker Profiles、issue-first、review gate、CodeRabbit fallback 與 PR Scope Police 合約。

Autonomous Worker Profiles 的 follow-up 改善以「約 40% infra 本質複雜、約 60% 工作流自己製造摩擦」為基準：infra 複雜度用固定 readback 與 gate 管住；流程摩擦要靠 `ops_spark` routing、review closeout evidence、subagent lifecycle cleanup、issue-first planning 與 follow-up split 降低。

autonomous work 一開始就必須先分派 worker，再進入計劃、開 issue、讀資料或實作；只有 `trivial/self-only exception` 可以不分派，但必須明寫原因。

**預設工作流程：**

1. 大範圍掃描 / 重複性工作 → **Gemini**
2. 架構規劃、issue 撰寫（PM 角色）→ **Claude Code**
3. 實作、debug、patch（工程師角色）→ **Codex**
4. 最終 PR 審查 → **Claude Code**

絕不用 Claude token 做重複性搜尋。

### Gemini 專責任務

| 任務類型 | 說明 |
|---|---|
| **代碼掃描** | 全域 pattern 搜尋、dead code、冗餘邏輯 |
| **文件生成** | 架構圖、技術文檔、README、API 規格提要 |
| **測試草稿** | 批量測試框架、測試覆蓋分析 |
| **Log 分析** | 大量 error 日誌、build 失敗診斷 |
| **依賴審查** | package.json / go.mod 升級影響分析 |
| **PR 初審** | scope pollution 檢查、風格一致性驗證 |

### 各角色職責總表

| 操作 | 誰執行 |
|---|---|
| 摘要大量檔案、生成樣板、審查 log、搜尋 pattern、草擬測試 | Gemini |
| 架構規劃、issue 撰寫、技術決策、最終 PR 審查 | Claude Code |
| 實作、debug、patch、跑測試、推 branch、開 PR | Codex |

### Claude Code 接到任務時

把模糊目標轉成可驗證目標再開始：

| 模糊 | 可驗證 |
|---|---|
| 「加驗證」 | 先寫 invalid input 測試，再讓測試通過 |
| 「修這個 bug」 | 先重現 bug 的測試，再讓測試通過 |
| 「實作 X」 | 先列出 checklist，每步附驗證條件 |

多步驟任務先說計畫再動手：

```text
1. [步驟] → 驗證：[如何確認]
2. [步驟] → 驗證：[如何確認]
```

@.claude/rules/pr-review-workflow.md

---

## 專案結構

```
tachigo/
├── services/
│   └── api/          # Go API (Gin + GORM + PostgreSQL)
├── apps/
│   ├── extension/    # Twitch Extension 前端 (React + TypeScript)
│   └── dashboard/    # 後台管理介面 (React + TypeScript) ← 建置中
├── docs/             # 設計文件與 AI 協作文檔
└── infra/            # repo automation scripts 與 git hooks
```

## 開發指令

```bash
make dev    # 啟動所有服務（hot reload）
make down   # 停止所有服務

# 執行後端測試
docker compose run --no-deps --rm app go test ./...
```

## Swagger Docs 更新規則

任何 PR 若有以下改動，**必須**在同一個 PR 裡附帶 `swag init` 產出的 docs 變更（`services/api/docs/docs.go`、`services/api/docs/swagger.json`、`services/api/docs/swagger.yaml`）：

- 新增、修改、刪除 handler function 的 swagger annotation（`// @Router`、`// @Param`、`// @Success` 等）
- 在 `router.go` 新增或移除路由

執行指令：
```bash
go install github.com/swaggo/swag/cmd/swag@latest
cd services/api && $(go env GOPATH)/bin/swag init -g cmd/server/main.go -o docs
```

## Claude Code 設定

`.claude/settings.json` 是共享設定，已 commit 進 repo，**請勿直接修改**。

個人設定請放在 `.claude/settings.local.json`（已 gitignore，不會影響其他人）。

## 文件放置規範

| 位置 | 對象 | 內容 |
|---|---|---|
| `README.md` | 所有人 | 開發環境建置（快速上手） |
| `docs/` | 工程師 | 架構設計、API 規格、技術決策 |
| `docs/ai/` | AI 協作者 / 工程師 | AI 協作指南與較長篇的 agent-facing 文件 |
| `infra/` | 工程師 / CI | repo-level automation scripts、git hooks、workflow 輔助檢查 |
| `plans/` | 工程師 | 實作計畫（每個功能開始前先寫） |
| `.claude/rules/` | 工程師 | agent 委託規則、工作流程決策（版控、共享） |
| GitHub Wiki | 全體人員 | 產品說明、功能介紹、非技術文件 |

### plans/ 慣例

- 每個功能或修改在開始實作前，先在 `plans/` 建立計畫文件
- 檔名：`<feature-slug>.md`，例如 `watch-points-channel-config.md`
- 計畫文件包含：背景、架構決策、待實作 checklist、驗證方式
- 完成後在文件頂端標注 `狀態：已完成`

## 架構參考

見 [docs/architecture.md](docs/architecture.md)
