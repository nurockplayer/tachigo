---
status: proposed
owner: engineering
last_reviewed: 2026-05-13
---

# tachigo Dev Portal 導覽網站設計文件

**日期**：2026-05-14  
**狀態**：proposed  
**目標**：以 `apps/docs` Docusaurus portal 作為呈現層，建立一個漂亮、可維護、能快速理解 `tachigo` / `tachiya` 架構的專案導覽入口。

> **命名說明**：早期討論曾用「Project Atlas」作為內部代號，但 repo 內已有 [atlasgo.io](https://atlasgo.io) 資料庫遷移工具相關檔案（`services/api/atlas.hcl`、`services/api/migrations/atlas.sum`、`docs/atlas-migration-plan.md`）。為避免 onboarding 混淆，實作時的使用者可見名稱、sidebar label、slug 與目錄一律使用 **tachigo Dev Portal** / **Dev Portal**，不要使用 Atlas。

---

## 背景

`tachigo` 已有 repo-first docs：Markdown 在 `docs/` 是 source of truth。Dev Portal 第一版應以 `apps/docs` Docusaurus 作為呈現層；若目標分支尚未包含 `apps/docs`，實作 PR 必須先建立薄 Docusaurus portal，再加上導覽入口。這個基礎適合讓新同事和日常開發者不必先讀完整文件樹，也能快速知道系統分層、domain 邊界、跨 repo 資料流與常見改動入口。

`tachiya` 是另一個獨立 GitHub repo（FastAPI + Saleor 整合），和 `tachigo` 不在同一個 monorepo。Cross-Repo Flows 頁面以外部連結引用 `tachiya` 的 GitHub 路徑即可，**不需要**在 Docusaurus 掛第二個 docs plugin。Dev Portal 第一版應把 `tachigo` 作為主 portal，將 `tachiya` 視為跨 repo 系統的一部分，而不是另開獨立網站。

---

## 使用者與主場景

### 主要使用者

- 新加入同事：需要在短時間內理解 repo 結構、產品 domain、從哪裡開始讀、第一個 PR 該注意什麼。
- 日常開發者：改功能前需要找到相關 service、handler、frontend page、API route、測試與文件。

### 次要使用者

- 架構 reviewer：需要快速確認改動是否跨 domain、是否碰到高風險資料流。
- AI agent：不做獨立聊天入口，但頁面結構、frontmatter、source links 應讓 agent 容易引用。

---

## 核心決策

| 問題 | 決策 |
|---|---|
| 放在哪裡？ | 使用 `apps/docs` Docusaurus 作為 docs portal；若尚未存在，第一版實作需先建立它。不建立獨立導覽 app。 |
| 首頁型態 | 做客製 tachigo Dev Portal 入口，不只是文件列表。 |
| Source of truth | Markdown 仍在 repo 中，PR 可 review；Docusaurus 只負責呈現。 |
| Graphify 角色 | 作為輔助圖譜與影響分析視圖，不作為唯一真相。 |
| AI-ready 程度 | 先用穩定頁面結構、frontmatter、source links 支援 AI；不做 AI chat。 |

---

## 資訊架構

第一版 Dev Portal 應包含以下入口：

| 頁面 | 用途 |
|---|---|
| `Dev Portal Home` | 第一視覺入口，摘要專案、主要系統圖、快速任務入口。 |
| `Start Here` | Onboarding path：first hour、first day、first PR。 |
| `Domain Maps` | 各 domain 的責任、檔案入口、API、測試、相關文件。 |
| `Cross-Repo Flows` | `tachigo` ↔ `tachiya` ↔ Saleor / Twitch / chain 的主要流程。 |
| `Daily Dev Guide` | 「我要改 X」時該看哪些檔案、跑哪些測試、注意哪些 policy。 |
| `Graph Explorer` | 連到或嵌入 graphify 產出的互動圖譜。 |
| `Source Index` | 以 domain 為索引列出 source files、tests、routes、docs。 |

---

## 第一版內容範圍

第一版採 P0 / P1 兩級，P0 必須完整交付，P1 允許以 stub（`_Coming soon_` + 入口連結）佔位。

### Page Priorities

| Priority | Page | Completion requirement |
|---|---|---|
| P0 | Dev Portal Home | 完整交付，作為 `/` 第一入口。 |
| P0 | Start Here | 完整交付 first hour / first day / first PR。 |
| P0 | Domain Maps | 完整交付 P0 domains；P1 domains 可 stub。 |
| P0 | Cross-Repo Flows | 完整交付 P0 flows；P1 flows 可 stub。 |
| P0 | Daily Dev Guide | 完整交付常見改動入口與測試指引。 |
| P0 | Source Index | 完整承接既有 taxonomy、文件狀態說明與 source links。 |
| P0 | Graph Explorer | 完整交付圖譜用途、限制與本機使用說明。 |

### Domain Maps

| Priority | Domain |
|---|---|
| P0 | Points / ledger / watch time |
| P0 | Auth / identity |
| P0 | Extension / sidepanel |
| P1 | Raffle / airdrop |
| P1 | Claim / spend / coupon redemption |
| P1 | Dashboard |
| P1 | Tachiya commerce integration |
| P1 | AI workflow / PR scope policy |

每個 domain 頁應回答：

1. 這個 domain 做什麼？
2. 主要資料流是什麼？
3. 主要程式入口在哪裡？
4. 相關 API route / frontend page / test 在哪裡？
5. 改這個 domain 時最容易踩到什麼？

### Cross-Repo Flows

| Priority | Flow |
|---|---|
| P0 | Twitch viewer watch flow：extension → tachigo API → points ledger |
| P0 | Coupon redemption flow：tachigo points / spend → tachiya FastAPI → Saleor |
| P1 | Streamer / agency management flow：dashboard → tachigo API → channel / agency state |

Cross-Repo Flows 頁連到 `tachiya` 時使用外部 GitHub 連結，不在 Docusaurus 掛第二個 docs plugin。連結格式：

- 已合併到 `tachiya` 預設分支的穩定內容：`https://github.com/nurockplayer/tachiya/blob/master/<path>`
- 尚未合併但必須引用的內容：使用 commit permalink `https://github.com/nurockplayer/tachiya/blob/<commit-sha>/<path>`，避免 branch 漂移。

---

## 視覺設計方向

Dev Portal 應該像「工程產品入口」，不是傳統文件首頁。

### 首頁

首頁第一屏包含：

- `tachigo Dev Portal` 標題與一句清楚定位。
- 兩個主要 CTA：`Start onboarding path`、`Find a feature`。
- 簡化系統圖：Twitch / extension / tachigo API / database / tachiya / Saleor / chain。
- 四個高層卡片：Onboarding、Domain Maps、Daily Dev、Graph Explorer。

### 導覽頁

導覽頁應偏向掃描式資訊：

- 少量大段文字。
- 多用表格、卡片、Mermaid 圖、source links。
- 每個重要概念都能一路點回實際 repo 檔案。

### 風格

- 使用現有 Docusaurus theme 與 `src/css/custom.css` 延伸，不引入新 UI library。
- 視覺應乾淨、專業、資訊密度高。
- 避免做成 marketing landing page；第一屏要直接提供導覽功能。

---

## 技術設計

### Docusaurus 結構

目標結構：

- `apps/docs/docusaurus.config.ts` 使用 classic preset。
- `docs/` 被掛在 `routeBasePath: '/'`。
- root `package.json` 提供 `pnpm build:docs`，對應 `pnpm --filter ./apps/docs build`。
- `docs/index.md` 作為 Dev Portal home，承接 repo-first docs 入口。

建議第一版：

- 建立或保留 `apps/docs` Docusaurus portal。
- 將現有 `docs/index.md` 改造成 Dev Portal home，使用 Markdown / HTML 區塊與 `custom.css` class 做出客製首頁。**原有 taxonomy 分類表與文件狀態說明搬入 `docs/dev-portal/source-index.md`；`docs/index.md` 保留一行連結指回 source-index。**
- 新增 `docs/dev-portal/` 目錄承載結構化導覽內容。
- 更新 `apps/docs/sidebars.ts`，讓 Dev Portal 成為 sidebar 的第一組入口。
- `docs/dev-portal/source-index.md` 必須明確納入 `docs/superpowers/specs/`，定位為「已確認或待確認的設計規格」，避免設計文件散落在 taxonomy 之外。

### Sidebar 結構草稿

```ts
// apps/docs/sidebars.ts（第一版建議結構）
const sidebars: SidebarsConfig = {
  docsSidebar: [
    {
      type: 'doc',
      id: 'index',
      label: 'Dev Portal Home',
    },
    {
      type: 'category',
      label: 'Dev Portal',
      collapsed: false,
      items: [
        'dev-portal/start-here',
        'dev-portal/domain-maps',
        'dev-portal/daily-dev-guide',
        'dev-portal/flows',
        'dev-portal/source-index',
        'dev-portal/graph-explorer',
      ],
    },
    // 以下保留現有結構
    { type: 'category', label: 'Architecture', collapsed: true, items: [ /* 現有 */ ] },
    { type: 'category', label: 'Domains',      collapsed: true, items: [ /* 現有 */ ] },
    // ...
  ],
};
```

Dev Portal category 展開置頂；原有 Architecture / Domains 等 category 預設折疊，避免 sidebar 過長。

### 建議檔案

| 檔案 | 操作 |
|---|---|
| `docs/index.md` | 改造成 tachigo Dev Portal 首頁，保留 repo-first wiki 說明。 |
| `docs/dev-portal/start-here.md` | 新增 onboarding path。 |
| `docs/dev-portal/domain-maps.md` | 新增 domain map index。 |
| `docs/dev-portal/flows.md` | 新增 cross-repo flow index。 |
| `docs/dev-portal/daily-dev-guide.md` | 新增日常開發導航。 |
| `docs/dev-portal/source-index.md` | 新增 source index。 |
| `docs/dev-portal/graph-explorer.md` | 新增 graphify 入口與使用說明。 |
| `apps/docs/package.json` | 若尚未存在，新增 Docusaurus docs app package。 |
| `apps/docs/docusaurus.config.ts` | 若尚未存在，新增 classic preset 設定，將 `../../docs` 掛到 `/`。 |
| `apps/docs/sidebars.ts` | 新增或更新 sidebar，將 Dev Portal 放在頂端。 |
| `apps/docs/src/css/custom.css` | 增加 Dev Portal 首頁與卡片樣式。 |
| root `package.json` | 若尚未存在，新增 `docs:start` 與 `build:docs` scripts。 |

### Graphify 整合

第一版不需要把 graphify 變成 build-time dependency。

做法：

- 在 `graph-explorer.md` 說明 graphify 產物位置與用途。
- 若要在 Docusaurus 中直接展示互動圖，可後續把 `graphify-out/graph.html` 複製到 `apps/docs/static/dev-portal/graph.html`，但第一版不強制。
- `graph.json` 可保留為未來 AI / search / impact view 的資料來源。

---

## 資料與維護流程

Dev Portal 頁面應使用固定 metadata，方便人類與 agent 查找。

建議 frontmatter：

```yaml
---
status: active
owner: engineering
last_reviewed: 2026-05-13
source_of_truth: true
code_areas:
  - services/api
  - apps/dashboard
  - apps/extension
related_repos:
  - tachigo
  - tachiya
---
```

維護規則：

- 新增或大改 domain 時，同 PR 更新對應 Dev Portal 頁。
- 若文件只是 proposal，不能放在 Dev Portal 當作已完成事實。
- Graphify 可定期重建，但 Dev Portal 頁面仍以人工整理過的 domain map 為主。

---

## 錯誤與風險處理

| 風險 | 處理方式 |
|---|---|
| Dev Portal 與程式碼漂移 | 每頁列出 source files；相關 PR 改 domain 時同步更新。 |
| 變成重複文件 | Dev Portal 只做入口與地圖，深入細節連回既有 docs。 |
| Graphify false positive | Graph Explorer 頁明確標示 AST-only 與 inferred edge 的限制。 |
| 首頁太像 marketing | CTA 導向實際導覽任務，不放空泛宣傳。 |
| Scope 過大 | 第一版只完整交付 P0 domain / P0 flow；P1 可 stub，不做搜尋引擎或 AI chat。 |

---

## 驗收標準

第一版完成後應符合：

- 新同事可從 `/` 在 10 分鐘內知道專案由哪些 major systems 組成。
- 開發者能從 Domain Maps 找到 3 個 P0 domain 的 source / tests / docs；P1 domains 可 stub，但必須有清楚入口。
- Cross-Repo Flows 至少包含 2 條 P0 實際資料流，且每條都有 Mermaid 圖與 repo path；P1 flow 可 stub。
- Graph Explorer 頁能清楚說明目前 graphify 圖譜的用途與限制。
- `pnpm build:docs`（root script，對應 `pnpm --filter ./apps/docs build`）通過，無 broken link 錯誤。

---

## 明確不做

- 不建立獨立導覽 app。
- 不新增資料庫、後端 API 或外部服務。
- 不新增 UI library。
- 不做 AI chatbot。
- 不把 graphify edge 當成未驗證的架構真相。
- 不一次重寫所有既有文件。

---

## 後續可擴充

- 將 graphify 產物複製到 Docusaurus static assets，提供可瀏覽互動圖。
- 從 `graph.json` 產出 domain cards 或 impact path。
- 加入搜尋索引與 owner tags。
- 若 Dev Portal 使用頻率高，再評估獨立導覽 app 或更完整互動圖譜。
