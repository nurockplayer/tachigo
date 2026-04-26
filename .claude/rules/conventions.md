# 共用規範

## 語言設定

永遠使用台灣正體中文回覆，不得使用日文、韓文或簡體中文。

## Branch 命名

`<type>/<short-description>`

例：`feat/points-service`、`fix/bits-receipt`、`docs/architecture`

## Commit 訊息格式

每個 commit 必須用 `refs #<issue號碼>` 標記，方便日後追溯規格與討論。

```
<type>: <short description>

refs #27

Co-Authored-By: Claude Sonnet 4.6 <claude[bot]@anthropic.com>
```

- 實作過程中的 commit 用 `refs #號碼`
- PR 的最後一個 commit 或 PR 描述用 `closes #號碼`（merge 後自動關閉 issue）

Type：`feat` / `fix` / `docs` / `chore` / `refactor` / `test`

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

例外：正式 `[release]` 的 `develop -> main` promotion PR 不屬於這條規則的限制對象

### 遇到岔路時怎麼做

- 如果額外內容是必要前置條件：先明確說明為什麼原 issue 缺這一塊，再決定是否調整範圍
- 如果額外內容不是必要前置條件：先記錄成新的 issue / TODO，不要混進目前 PR
- 若 PR 已經超出 issue 範圍，reviewer 可以直接要求拆 PR、縮 scope 或關閉 draft PR

### 例外：Trivial 附帶修

< 10 行、直接關聯當前 PR scope 的小修正（typo、import、config 微調），可 inline 進同一個 PR，**不需另開 issue**。由人類判斷是否符合，AI 不得自行套用此例外。

### 例外：Conflict 解法

以下情況允許直接 `merge develop` 解 conflict，不需 cherry-pick / restack：

- PR 改動 < 50 行，且
- Conflict 是由 develop 的無關改動造成（非 scope 污染引起）

不符合以上條件時，仍須 restack（從最新 develop 開新 branch，cherry-pick 原 commits）。

## 細粒度原則

禁止 scope pollution 之外，還要**主動拆細** Issue / Commit / PR。

### 何時應該拆細

開始實作前，問自己：

- 這個任務涉及 3 個以上不相關的子系統？→ 拆
- PR diff 預估超過 400 行？→ 拆
- 涉及多個 layer（schema + service + API + 前端）任兩種以上？→ 拆
- 修改的文件超過 15 個？→ 拆
- 這個功能可以分階段交付（MVP + 優化）？→ 拆
- issue body 中有明確邊界，但這個 PR 已超出邊界？→ 拆

任何一個條件成立就應該拆。

### PR Diff 限制

| 區間 | 評估 | 處理 |
|---|---|---|
| < 200 行 | ✅ 最佳 | 直接秒審 |
| 200-400 行 | ✅ 很好 | 標準審查 |
| 400-600 行 | ⚠️ 注意 | 建議拆分（不阻擋） |
| 600-1000 行 | ⚠️ 危險 | 需在 PR body 說明為何不拆（不阻擋） |
| 1000+ 行 | ❌ 超限 | 自動擋下（`scope-violation` label） |
| 1001-1500 行 | 例外上限 | generated code / migration / dep bump 可用 `scope-exception` label |
| release PR | — | 不受限制 |

### Claude Code 實作前必檢

在任何實作開始前：

1. 評估「這個任務會產生多大的 PR？」
2. 如果預估 > 400 行，立即詢問：「這個任務預估會改 X 行，建議拆細為 PR1: [A]、PR2: [B]，同意嗎？」
3. 只有獲得明確同意後才開始實作

## AI 協作守則

若貢獻內容主要由 AI 產生，必須額外遵守以下規則：

- 不得讓 AI 自行擴張 issue scope；AI 提出的額外功能、future work、重構建議，必須拆成獨立 issue / PR
- 不得把 docs / research draft / brainstorming 內容直接當成 implementation source of truth，除非 repo 已明確指定
- 不得未經驗證就宣稱「已完成」；至少要回報實際執行過的測試、未驗證部分、以及已知風險
- reviewer 應優先檢查 AI 是否偏離 issue、腦補需求、混入未要求的 schema / API / UI 改動，而不是只看程式碼表面是否完整

## 操作權限邊界

- **Read-only 操作**可直接執行（無需詢問）：讀檔、搜尋、`gh pr view` / `gh issue view` 等查詢、`gh api` 讀取、diff 掃描、本機分析
- **變動操作**必須事先詢問並取得明確同意：Edit / 寫檔、commit、push、branch switch / rebase / merge、GitHub comment / review / CR / Approve / Merge、issue / PR 建立或編輯
