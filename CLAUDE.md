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

本專案使用 merge commit（--no-ff），PR 內的所有 commit **會直接進 develop history**，feature branch 的分支結構保留在 git graph 上。

- 按邏輯步驟分割 commit，方便 reviewer 追蹤實作脈絡，也方便日後 bisect
- fixup commit（修 CodeRabbit 意見、修 scope police）是正常的，不必 rebase 清理
- 避免「一次性 commit」把不相關的層混在一起（會讓 review 難以跟進）
- **PR title 仍要精確**，merge commit 會引用它

### PR

- 專注一個主題：一個 PR 應該對應一個 issue，不應跨越多個獨立功能
- 保持可控大小：盡量 < 400 lines（除非不可避免的大改）
- 大 PR 難以 review、容易漏漏、合併時風險高
- 細粒度 PR 審查週期短、反饋快、merge 也快

### 何時應該拆細

開始實作前，問自己：

- 這個任務涉及 3 個以上不相關的子系統？→ 拆
- PR diff 預估超過 400 行？→ 拆
- 涉及多個 layer（schema + service + API + 前端）任兩種以上？→ 拆
- 修改的文件超過 15 個？→ 拆
- 這個功能可以分階段交付（MVP + 優化）？→ 拆
- issue body 中有明確邊界，但這個 PR 已超出邊界？→ 拆

任何一個條件成立就應該拆。

#### PR Diff 限制

**建議的 PR 大小區間**（performance guideline）：

| 區間 | 評估 | 審查策略 |
| --- | --- | --- |
| < 200 行 | ✅ 最佳 | 直接秒審，無需分段 |
| 200-400 行 | ✅ 很好 | 標準單次審查 |
| 400-600 行 | ⚠️ 注意 | 軟提示：建議拆分 |
| 600-1000 行 | ⚠️ 危險區間 | 警告：需在 PR body 說明為何不拆 |
| 1000+ 行 | ❌ 超限 | 自動擋下（特例除外） |

**目標門檻**（`PR Scope Police`，ci.yml 將於後續 PR 同步）：

| 限制 | 行數 | 處理 |
| --- | --- | --- |
| **軟提示門檻** | 400+ | 建議拆分（不阻擋） |
| **警告門檻** | 600+ | 需在 PR body 說明為何不拆（不阻擋） |
| **硬限制** | 1000+ | 自動擋下（`scope-violation` label） |
| **例外上限** | 1500 | migration / generated code / dependency bump 可用 `scope-exception` label |
| **發佈無限** | — | release promotion PR (develop → main) 不受限制 |

**何時使用 `scope-exception`**：

- Generated code（Swagger、protobuf 等自動產物）被迫大幅變動
- Database migration 或 schema refactor 難以分段
- 大型 dependency 升級的一次性改動
- **不可** 用於變相迴避 PR 大小限制

**發現超限時**：

1. 先嘗試拆 PR（儘量在 200-400 行，最多到 600）
2. 若無法降到 1000 以下，評估是否 `[release]` 或符合例外條件
3. 若符合例外，使用 `scope-exception` label（maintainer 可授權）

### Claude Code 實作前必檢

在任何實作開始前，Claude Code 應該：

1. 評估「這個任務會產生多大的 PR？」
2. 如果預估 > 400 行，立即詢問：「這個任務預估會改 X 行，建議拆細為 PR1: [A]、PR2: [B]，同意嗎？」
3. 只有獲得明確同意後才開始實作

## AI 協作守則

若貢獻內容主要由 AI 產生，必須額外遵守以下規則：

- 不得讓 AI 自行擴張 issue scope；AI 提出的額外功能、future work、重構建議，必須拆成獨立 issue / PR
- 不得把 docs / research draft / brainstorming 內容直接當成 implementation source of truth，除非 repo 已明確指定
- 不得未經驗證就宣稱「已完成」；至少要回報實際執行過的測試、未驗證部分、以及已知風險
- reviewer 應優先檢查 AI 是否偏離 issue、腦補需求、混入未要求的 schema / API / UI 改動，而不是只看程式碼表面是否完整

## Branch 命名

`<type>/<short-description>`

例：`feat/points-service`、`fix/bits-receipt`、`docs/architecture`

## Merge 策略

本專案使用 **merge commit（--no-ff）**：feature branch 進 develop 時保留分支結構，git graph 看得到每條 branch 的進出。

- **PR body** 放 `closes #號碼`，merge 後自動關閉 issue
- PR 內的 individual commit 用 `refs #號碼` 標記，供 review 期間追溯用
- **PR title 要精確**，GitHub merge commit 會引用它

## Commit 訊息格式（PR 內）

每個 commit 必須用 `refs #<issue號碼>` 標記，方便 review 期間追溯規格與討論。

```
<type>: <short description>

refs #27

Co-Authored-By: Claude Sonnet 4.6 <claude[bot]@anthropic.com>
```

Type：`feat` / `fix` / `docs` / `chore` / `refactor` / `test`

---

## AI 分工

本專案使用 Claude Code + Gemini CLI + Codex CLI 協作開發。

**預設工作流程：**

1. 大範圍掃描 / 重複性工作 → 先交給 **Gemini**
2. 架構規劃、issue 撰寫（PM 角色）→ **Claude Code**
3. 實作、debug、patch（工程師角色）→ **Codex**
4. 最終 PR 審查 → **Claude Code**

絕不用 Claude token 做重複性搜尋。

### 任務規模路由

> **核心原則：Codex 適合確定性高的任務，Claude 適合模糊性高的任務。**

| 任務規模 | 流程 |
|---|---|
| **Trivial**（< 10 行、config 調整、typo）| Claude 直接 patch，不走 issue 流程 |
| **Small-Medium**（功能、API、元件）| Claude 寫 issue → Codex 實作 → Claude review |
| **需要迭代測試**（跑測試直到過）| 一定走 Codex；Claude 只負責寫 issue |
| **架構重構 / 高風險改動**| Claude 先設計方案，拆成多個 issue 再交 Codex |

預設路由：收到實作需求先判斷規模，Trivial 以外一律寫 issue 交 Codex。

### Gemini 專責任務

| 任務類型 | 說明 | 範例 |
| --- | --- | --- |
| **代碼掃描** | 全域 pattern 搜尋、dead code、冗餘邏輯 | `find . -name "*.go" \| xargs cat \| gemini -p "找出所有未使用的 helper function"` |
| **文件生成** | 架構圖、技術文檔、README、API 規格提要 | 更新專案架構文件、生成依賴關係圖 |
| **測試草稿** | 批量測試框架、測試覆蓋分析 | 給定測試風格，生成 20+ 個測試案例 |
| **Log 分析** | 大量 error 日誌、build 失敗診斷 | `cat error.log \| gemini -p "分析這個日誌中的錯誤原因"` |
| **依賴審查** | package.json / go.mod 升級影響分析 | 評估升級會影響哪些模組 |
| **PR 初審** | scope pollution 檢查、風格一致性驗證 | 檢查 PR 是否混入了不相關的改動 |

### 各角色職責總表

| 操作 | 誰執行 |
|---|---|
| 摘要大量檔案、生成樣板、審查 log、搜尋 pattern、草擬測試 | Gemini（`gemini -p "<task>"`；確認無風險時可加 `--yolo`） |
| 架構規劃、issue 撰寫、技術決策、最終 PR 審查 | Claude Code（PM 角色） |
| 實作、debug、patch、跑測試、推 branch、開 PR | Codex（工程師角色） |

## PR 審查流程

### 審查主體與分工

PR 審查由 AI 自動執行，人只在決策節點確認。預設流程：

1. **Gemini 低成本初審**（節省 Claude token）：
   - 掃描高風險區域（binary 檔、schema、scope 污染等）
   - 若發現 high 優先級 blocker → Claude 驗證 + 決策
   - 若無 blocker → Claude / Codex 只針對必要 diff 與風險點繼續審查

2. **Claude 驗證 & 決策審查**：
   - 驗證 Gemini 發現的 blocker
   - 無 blocker 時避免重讀整包，只檢查 CLAUDE.md 合規、明顯 bug、git history 與必要上下文
   - 生成最終審查結論與建議決策

3. **人類決策確認**（關鍵節點）：
   - 發現 blocker → 確認是否提交 Change Request
   - 無 blocker、CI 通過 → 建議 merge 並確認是否執行
   - 不確定 → 確認是否繼續調查

詳見下方「Claude Code PR 審查規則」。

## Claude Code PR 審查規則

Claude Code 負責 PR 審查的驗證與決策階段。

### 操作權限邊界

- **Read-only 操作**可直接執行（無需詢問）：
  - `gh pr view`、`gh pr list`、`gh issue view` 等 GitHub 查詢
  - `gh api` 讀取請求
  - 查看 PR metadata、讀檔、掃描代碼、檢查 diff、產生本機分析
- **變動操作**必須事先詢問用戶並取得明確同意：Edit / 寫檔、format 造成檔案變更、commit、push、branch switch / rebase / merge、GitHub comment、GitHub review、Change Request、Approve、Merge、issue / PR 建立或編輯

### 審查流程

#### 問題分級

- `blocker`：必須擋 merge；包含正確性、安全、資料一致性、權限、breaking change、難 rollback migration、或高風險核心路徑缺測。
- `major`：重要但不一定擋 merge；沒有 blocker 時可合併，但需明確提醒風險並視情況開 follow-up issue / PR。
- `minor`：有用改善，不阻擋 merge。
- `nit`：純風格或可讀性細節。

#### 第一步：接收 Gemini 初審結果或獨立掃描

- 若 Gemini 已初審，驗證其發現的 blocker（優先檢查 high 優先級問題）
- 若無 Gemini 初審或 Gemini 無發現，自行掃描高風險區域

#### 第二步：高風險區域優先掃描（節省 token）

發現 high 優先級 blocker 立即停止、不繼續深入：

- Binary 檔案：字體、圖片、screenshot、bundle、archive（應使用 git-lfs）
- 重大變更：schema、migration、API contract、auth / payment 邏輯
- Scope 污染：是否混入無關改動（核對與 issue、PR title、CLAUDE.md scope 規則）
- CI 失敗、必要測試失敗

#### 一旦發現並驗證 high 優先級 blocker

- 停止進一步分析（節省 token）
- 總結 blocker 給用戶
- 詢問：「同意提交 Change Request 嗎？」
- 等待用戶確認後執行 `gh pr review --request-changes`

#### 第三步：無 high 優先級問題時進行必要審查

- 檢查 CLAUDE.md 合規性（scope、細粒度、AI 協作守則）
- 掃描明顯 bug、邏輯錯誤、edge case
- 檢查 git history 一致性（commit 訊息、原子化等）
- 驗證現有 CodeRabbit / Codex comment 中的建議

#### 第四步：決策與確認

審查完成後，必須明確詢問用戶並等待確認，不得預設執行：

- 發現 blocker → 「同意提交 Change Request 嗎？」
- 無 blocker、CI 通過 → 「目前沒有 blocker，是否同意 merge？」
- 只有 major / minor / nit、沒有 blocker → 預設仍可 merge，但需摘要非阻塞風險與 follow-up 建議
- 不確定狀態 → 列出不確定事項，詢問「繼續調查或先暫停？」

### Gemini CLI 限制

- 避免並發 Gemini 任務（會觸發 429 quota error）
- 大型 PR（600+ 行）用一個完整 prompt 等待完成，不分批
- 若需多個分析，改為序列執行
- 若遇到 429、quota exceeded、rate limit、daily limit reached，立即停止 Gemini 路徑，不做連續重試，改用 Claude / Codex 以最小必要上下文完成審查

### 審查執行模式選擇

根據 PR 風險等級與規模，選用不同執行順序，以平衡速度、成本與品質：

#### 模式一：標準模式（Normal PR）

適合：中等規模（100-600 行）、中等風險的日常 PR（業務邏輯、API 變更、功能補充）。

執行順序：

1. Claude 掃描 PR metadata、檔案列表、diff 摘要，分類風險
2. Gemini 串行執行 first-pass summary（快速掃 diff）
3. Gemini 串行執行 pattern / test gap 掃描
4. Codex 依必要 diff 與風險點進行主審，分級 blocker / major / minor / nit
5. Claude 驗證 blocker，綜合決策
6. 詢問用戶：Change Request 或 Merge

**token 成本**：中等。適合日常工作流。

#### 模式二：極省 token 模式（Small / Low-risk PR）

適合：小型 PR（< 100 行）或明顯低風險改動（UI 文案、簡單 refactor、註解補充、測試補充）。

執行順序：

1. Claude 快速掃 diff 與檔案列表
2. 直接交給 Codex 主審
3. 只有出現邏輯可疑點才動態叫 Gemini 補掃
4. Claude 驗證 blocker，做最終判定

**token 成本**：低。Gemini 可能完全不用。

#### 模式三：高風險模式（Auth / Payment / Migration / Security PR）

適合：涉及用戶認證、支付邏輯、資料遷移、安全補丁、權限控制的 PR（無論大小）。

執行順序：

1. Claude 明確標記哪些變更是高危（auth、payment 相關改動）
2. Gemini 做非常窄範圍掃描，只針對高危變更與相似實作搜尋
3. Codex 細緻主審，特別關注 blocker 與 major
4. Claude **深入驗證** blocker：檢查相關代碼、git history、相似實作的一致性
5. 一旦確認 blocker，直接問用戶是否同意 Change Request

**token 成本**：高（但必要且值得）。Gemini 範圍窄但深。

#### 模式選擇表

| PR 類型 | 規模 | 風險特徵 | 選用模式 |
| --- | --- | --- | --- |
| UI 文案 / 樣式 | 10-50 行 | 低 | 極省 token |
| 簡單 refactor | 50-150 行 | 低 | 極省 token |
| 測試補充 | 50-200 行 | 低 | 極省 token |
| Feature | 150-400 行 | 中（業務邏輯、API） | 標準模式 |
| 大型 feature | 400-600 行 | 中 | 標準模式 |
| 使用者認證 | 任何 | 高 | 高風險模式 |
| 支付 / 結帳 | 任何 | 高 | 高風險模式 |
| 資料遷移 / schema | 任何 | 高 | 高風險模式 |
| 權限 / 存取控制 | 任何 | 高 | 高風險模式 |
| 安全補丁 | 任何 | 高 | 高風險模式 |
| 基礎設施 / CI | 任何 | 中-高 | 標準或高風險模式 |

### 結構化輸出格式

所有 PR 審查結果應盡量按以下格式組織，便於統一、精簡、避免冗長敘述：

```
## Summary
一句話總結這個 PR 在做什麼與風險等級（低 / 中 / 高）。

## Blockers
- [檔案:行號] 問題簡述 / 影響 / 建議
- ...

## Majors
- [檔案:行號] 問題簡述 / 影響 / 建議
- ...

## Minors
- [檔案:行號] 建議
- ...

## Nits
- 純風格 / 可讀性細節

## Questions
- 需要作者確認的地方

## Recommended action
Change Request / Merge
```

**優點**：

- 精簡輸出，減少敘述性冗餘
- AI 易遵循，減少偏差
- 用戶快速掃一遍就知道關鍵問題

### 互動話術

審查完成後，根據 blocker 狀態用以下話術詢問用戶：

#### 情況 A：發現 blocker

```
我已完成審查，確認存在 blocker：

## Blockers
- [file:line] <問題>

這些問題會直接影響 <正確性 / 安全性 / 資料一致性>，
因此目前不建議合併。

是否要我直接提交 Change Request？
```

#### 情況 B：無 blocker 但有 major / minor

```
我已完成審查，目前沒有 blocker。

另外有以下非阻塞建議：
- major: <...>
- minor: <...>

整體可合併，但建議後續開 follow-up issue 跟進。

是否同意執行 Merge？
```

#### 情況 C：無 blocker，只有 nit

```
我已完成審查，沒有發現問題。

是否同意直接 Merge？
```

#### 情況 D：不確定狀態

```
我遇到幾個不確定的地方：
- <不確定項 1>
- <不確定項 2>

是要我繼續調查，還是先暫停？
```

### Codex 執行守則

Codex 在擔任 PR 主審時應遵守以下規則，以節省 token 並提高效率：

#### 以 Diff 為中心

- 優先讀 diff，以 diff 中的變更為審查核心
- 不任意擴大上下文，除非確有必要驗證某個具體疑慮

#### 優先掃描高風險區域

先檢查以下順序，找到問題再決定是否深入：

1. 正確性：邏輯錯誤、邊界條件
2. 安全性：auth、permission、injection
3. 資料完整性：deletion、update、transaction、race condition
4. API compatibility：breaking change、contract 變化
5. 測試覆蓋：high-risk 改動是否有測試

#### 問題分級要精準

- **Blocker** 才標 blocker（真的會破壞功能 / 安全性 / 資料）
- 沒把握的猜測不升級成 blocker，改標 major
- 每個 blocker 都要能說出具體影響

#### 需要更多上下文時先說明理由

不要默默讀一堆檔案，而是：

```
為了驗證 <具體疑慮>，我需要看 <特定檔案>，理由是 <...>。
```

讓 Claude 決定是否展開，避免盲目擴張。

#### 優先輸出精簡 findings

用上述結構化格式：

- 先輸出清單（blocker / major / minor / nit）
- 只在需要驗證時補充詳細證據
- 避免冗長敘述或重複解釋

#### 應該停止的情況

當發現以下情況時，立即停止深入，回報給 Claude：

- 發現明確 blocker，不需要繼續掃
- 上下文快速膨脹，token 開始失控
- 遇到不清楚的架構決策，需要 Claude 判斷

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
| `.claude/rules/` | 工程師 | agent 委託規則、工作流程決策（版控、共享） |
| GitHub Wiki | 全體人員 | 產品說明、功能介紹、非技術文件 |

### plans/ 慣例

- 每個功能或修改在開始實作前，先在 `plans/` 建立計畫文件
- 檔名：`<feature-slug>.md`，例如 `watch-points-channel-config.md`
- 計畫文件包含：背景、架構決策、待實作 checklist、驗證方式
- 完成後在文件頂端標注 `狀態：已完成`

## 架構參考

見 [docs/architecture.md](docs/architecture.md)
