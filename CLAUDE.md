# tachigo — Claude Code Guidelines

## 語言設定

永遠使用台灣正體中文回覆，不得使用日文、韓文或簡體中文。

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

常用範例：

- 不修改未列於本票的 schema / API contract
- 不擴張到其他頁面 / 其他角色 / future scope
- 不把 placeholder、research 或 draft 內容視為正式完成
- 不補本票依賴但尚未由上游提供的能力
- 不進行與本票無關的重構

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
   - 等未來有正式部署、freeze window、hotfix/backport 需求時，再升級為 release branch 流程

## Scope 邊界

禁止 scope pollution：不要把 issue 沒有明確要求的內容混進同一個 PR。

### 基本規則

- PR 只應包含該 issue 明確列出的任務、規格與完成條件
- 實作途中若發現額外想做的功能、重構、future work、design exploration，必須另開 issue / PR，不可順手一起提交
- docs / research draft 不能自動視為 implementation source of truth；只有被明確指定的 issue / PR / 文件，才能作為當前實作依據

### 常見禁止情況

- issue 只要求 migration，PR 卻同時加入 service、handler、router、前端串接
- 本輪 MVP 只要求單一畫面，PR 卻順手加入 future panels、bottom nav、完整 design system
- 修 bug 時順便重構整個模組，且未經事前同意
- backend issue 混入 dashboard / tachimint UI 改動，反之亦然

### PR 不得依賴未 merge 的 PR

禁止在一般 feature PR body 中宣告對其他尚未 merge 的 PR 的依賴（例如「依賴：#123 需先 merge」）。

- 若前置 PR 尚未 merge，後續 PR 不應開出來
- 若真有順序依賴，先等前置 PR merge，再從最新 `develop` 拉新 branch 開發
- scope police 會自動偵測 `依賴：#xxx` 或 `depends on #xxx` 語法，若引用的 PR 仍為 open 狀態，該 PR 會被自動關閉

例外：

- 正式 `[release]` 的 `develop -> main` promotion PR 不屬於這條規則的限制對象

### 遇到岔路時怎麼做

- 如果額外內容是必要前置條件：先明確說明為什麼原 issue 缺這一塊，再決定是否調整範圍
- 如果額外內容不是必要前置條件：先記錄成新的 issue / TODO，不要混進目前 PR
- 若 PR 已經超出 issue 範圍，reviewer 可以直接要求拆 PR、縮 scope 或關閉 draft PR

## 細粒度原則

禁止 scope pollution 之外，還要**主動拆細** Issue / Commit / PR。細粒度帶來更好的可審查性、可追蹤性、可回滾性。

### Issue

- 單一職責：一個 issue 解決一個明確問題或實現一個完整功能
- 避免「大雜燴」issue（如「優化所有頁面」、「重構整個模組」）
- 如果做著做著發現要 touch 多個不相關的子任務，先拆成多個 issue
- 細粒度 issue 更容易讓人專注、評估、以及日後追溯

### Commit

- 原子化：每個 commit 應該是獨立的工作單位，可以單獨 revert、cherry-pick、bisect
- 避免「一次性 commit」（如一次改 schema + service + handler + 前端四個層）
- 按邏輯層或步驟分割：schema migration → service 實作 → API 路由 → 前端整合，各為一個 commit
- 細粒度 commit 讓 code review 和問題追蹤更精確

### PR

- 專注一個主題：一個 PR 應該對應一個 issue，不應跨越多個獨立功能
- 保持可控大小：盡量 < 400 lines（除非不可避免的大改）
- 大 PR 難以 review、容易漏漏、合併時風險高
- 細粒度 PR 審查週期短、反饋快、merge 也快

## AI 協作守則

若貢獻內容主要由 AI 產生，必須額外遵守以下規則：

- 不得讓 AI 自行擴張 issue scope；AI 提出的額外功能、future work、重構建議，必須拆成獨立 issue / PR
- 不得把 docs / research draft / brainstorming 內容直接當成 implementation source of truth，除非 repo 已明確指定
- 不得未經驗證就宣稱「已完成」；至少要回報實際執行過的測試、未驗證部分、以及已知風險
- reviewer 應優先檢查 AI 是否偏離 issue、腦補需求、混入未要求的 schema / API / UI 改動，而不是只看程式碼表面是否完整

## Branch 命名

`<type>/<short-description>`

例：`feat/points-service`、`fix/bits-receipt`、`docs/architecture`

## Commit 訊息格式

每個 commit 必須用 `refs #<issue號碼>` 標記相關 issue，方便日後追溯當初的規格與討論。

```
<type>: <short description>

refs #27

Co-Authored-By: Claude Sonnet 4.6 <claude[bot]@anthropic.com>
```

- 實作過程中的 commit 用 `refs #號碼`
- PR 的最後一個 commit 或 PR 描述用 `closes #號碼`（merge 後自動關閉 issue）

Type：`feat` / `fix` / `docs` / `chore` / `refactor` / `test`

---

## Review 流程

有意義的程式碼變更完成後：

1. 自我檢查實作
2. 跑最低限度的測試
3. 執行 `/codex:review`
4. auth / payment / security 邏輯額外執行 `/codex:adversarial-review`
5. 只套用有理由的建議

---

## 專案結構

```
tachigo/
├── backend/          # Go API (Gin + GORM + PostgreSQL)
├── tachimint/        # Twitch Extension 前端 (React + TypeScript)
├── dashboard/        # 後台管理介面 (React + TypeScript) ← 建置中
└── docs/             # 設計文件
```

## 開發指令

```bash
make dev    # 啟動所有服務（hot reload）
make down   # 停止所有服務

# 執行後端測試
docker compose run --no-deps --rm app go test ./...
```

## Claude Code 設定

`.claude/settings.json` 是共享設定，已 commit 進 repo，**請勿直接修改**。

個人設定請放在 `.claude/settings.local.json`（已 gitignore，不會影響其他人）。

## 文件放置規範

| 位置 | 對象 | 內容 |
|---|---|---|
| `README.md` | 所有人 | 開發環境建置（快速上手） |
| `docs/` | 工程師 | 架構設計、API 規格、技術決策 |
| `plans/` | 工程師 | 實作計畫（每個功能開始前先寫） |
| GitHub Wiki | 全體人員 | 產品說明、功能介紹、非技術文件 |

### plans/ 慣例

- 每個功能或修改在開始實作前，先在 `plans/` 建立計畫文件
- 檔名：`<feature-slug>.md`，例如 `watch-points-channel-config.md`
- 計畫文件包含：背景、架構決策、待實作 checklist、驗證方式
- 完成後在文件頂端標注 `狀態：已完成`

## 架構參考

見 [docs/architecture.md](docs/architecture.md)
