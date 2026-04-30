# tachigo — Claude Code Guidelines

@.claude/rules/conventions.md

## 動手前

- 需求有歧義時，列出多種詮釋再問，不要默默選一個
- 不確定的地方直接說出來，不要猜測後實作
- 若有更簡單的解法，主動說出來並等確認

## GitHub Issue 慣例

### 標題前綴

所有 issue 標題必須有前綴：

| 前綴 | 用途 |
|---|---|
| `[backend]` | 後端開發任務（Go） |
| `[frontend]` | 前端開發任務（React / TypeScript） |
| `[discussion]` | 架構決策、設計討論，尚未有結論 |

範例：
- `[backend] PointsService — 雙帳本記帳`
- `[frontend] Extension — 點數餘額顯示`
- `[discussion] Token 經濟設計與 Soulbound 衝突`

### Label

| Label | 用途 |
|---|---|
| `feature` | 開發任務 |
| `discussion` | 討論票（搭配 `[discussion]` 前綴使用） |
| `needs-codex-review` | PR 有新 commit，輪到 Codex 重新審查 |
| `changes-requested` | Codex 已提出 blocker，輪到作者修正 |

### Issue 內容格式

開發任務（`[backend]` / `[frontend]`）需包含：

- **背景** — 這個功能是為了解決什麼問題
- **任務** — 具體要做什麼（用 checklist）
- **介面／規格** — Go interface、API 規格、或 component props
- **參考** — 現有的範本檔案路徑
- **完成條件** — PR merge 前必須達成的條件（checklist）

對於 MVP 邊界、migration / schema、frontend page、docs / design、setup / scaffold 這類容易被擴張範圍的 issue，建議額外補一段 **本票明確不做**，只需列出最常見的外擴方向即可，不必追求完整黑名單。

討論票（`[discussion]`）不需要固定格式，但要列出待決定的問題點。

---

## 開發流程

1. 從 `develop` 拉新的 feature branch：

   ```bash
   git checkout develop
   git pull
   git checkout -b feat/points-service
   ```

2. 開發完成後推上 remote：

   ```bash
   git push -u origin feat/points-service
   ```

3. 在 GitHub 發 PR，日常 feature PR 目標分支：`develop`

4. 正式 release 流程走 Git Flow：

   - `main` 不接受日常 feature PR
   - 每兩週由 `develop` 開一張正式 release PR 到 `main`
   - release PR 使用 `[release]` title prefix
   - `develop -> main` release PR 屬於正式 promotion 流程，不視為 scope exception
   - 目前暫不使用 `release/*` branch

## Merge 策略

本專案使用 **merge commit（--no-ff）**：feature branch 進 develop 時保留分支結構。

- **PR body** 放 `closes #號碼`，merge 後自動關閉 issue
- PR 內的 individual commit 用 `refs #號碼` 標記
- **PR title 要精確**，GitHub merge commit 會引用它

## PR Diff 限制

| 限制 | 行數 | 處理 |
| --- | --- | --- |
| **軟提示門檻** | 400+ | 建議拆分（不阻擋） |
| **警告門檻** | 600+ | 需在 PR body 說明為何不拆（不阻擋） |
| **硬限制** | 1000+ | 自動擋下（`scope-violation` label） |
| **例外上限** | 1500 | migration / generated code / dependency bump 可用 `scope-exception` label |
| **發佈無限** | — | release promotion PR (develop → main) 不受限制 |

---

## AI 分工

本專案使用 Claude Code + Gemini CLI + Codex CLI 協作開發。

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
