# Dashboard 技術選型評估

## 背景

目前 `tachigo` 已有一個獨立的 `dashboard/` 前端專案，技術基礎如下：

- React 19 + TypeScript + Vite
- React Router
- Tailwind CSS v4
- 手動放入的 `shadcn/ui` 基礎元件
- 後端為 Go + Gin + GORM，提供 REST API

本文件的目的是評估：

- 目前同事已經做的 dashboard 骨架是否適合延續
- 針對 MVP 階段，是否適合導入 `Refine.dev`
- 若導入，應採取什麼範圍與方式

---

## 目前程式狀態摘要

### 已有基礎

- [`dashboard/src/App.tsx`](/Users/tachikoma/Documents/Web3/tachigo/dashboard/src/App.tsx)
  - 已建立登入頁、受保護路由、Layout 與幾個基礎頁面
- [`dashboard/src/components/Layout.tsx`](/Users/tachikoma/Documents/Web3/tachigo/dashboard/src/components/Layout.tsx)
  - 已有基本側邊欄與頁面容器
- [`dashboard/src/pages/LoginPage.tsx`](/Users/tachikoma/Documents/Web3/tachigo/dashboard/src/pages/LoginPage.tsx)
  - 已完成登入 UI，並串接登入 API
- [`dashboard/src/services/auth.ts`](/Users/tachikoma/Documents/Web3/tachigo/dashboard/src/services/auth.ts)
  - 已有 login/logout 邏輯與 token 設定
- [`backend/internal/router/router.go`](/Users/tachikoma/Documents/Web3/tachigo/backend/internal/router/router.go)
  - 後端已提供 auth 與部分 dashboard/API 路由骨架

### 尚未成形的部分

- `Streamers`、`Transactions`、`Settings` 頁面目前仍是 placeholder
- 尚未看到完整 CRUD 頁面、table、form、query cache、access control 抽象
- 尚未導入 `Refine`、`React Query`、`TanStack Table` 等主要資料層工具

### 目前技術風險

- [`dashboard/src/services/auth.ts`](/Users/tachikoma/Documents/Web3/tachigo/dashboard/src/services/auth.ts)
  - `accessToken` 僅保存在記憶體
  - 重新整理頁面後，`isAuthenticated()` 會回傳 false
- [`dashboard/src/components/ProtectedRoute.tsx`](/Users/tachikoma/Documents/Web3/tachigo/dashboard/src/components/ProtectedRoute.tsx)
  - 現在的 route guard 只適合非常早期原型
- [`dashboard/README.md`](/Users/tachikoma/Documents/Web3/tachigo/dashboard/README.md)
  - 仍是 Vite 預設內容，尚未反映專案實際開發方式

---

## 選型結論

### 結論摘要

在目前條件下：

- 不追求深度客製化
- 目前還在 MVP
- 希望盡可能有現成能力快速跑起來

建議採用：

**`Refine.dev` 作為 dashboard framework，保留既有 React 專案與部分 UI 骨架。**

### 為什麼不是 Go Admin

- 目前已經有獨立 `dashboard/` React 專案
- 現有 backend 是 API-first 架構，不是 server-rendered admin 架構
- 若改用 Go Admin，會與現有前後端分離方向衝突
- UI 自由度與前端生態整合性都較差

### 為什麼暫時不優先選純 `shadcn/ui + TanStack Table`

- 這條路最自由，但也要自己補最多骨架
- MVP 階段會花很多時間處理：
  - CRUD 頁面結構
  - 資料抓取與 mutation 狀態
  - 權限與 route convention
  - 表格與表單樣板
- 對目前需求來說，這些多半不是核心產品差異

### 為什麼 `Refine` 最適合現在

- 比較符合「先跑起來」的目標
- 能快速建立 resource-based CRUD 頁面
- 可接既有 REST API，不需要改變 backend 技術棧
- 保留 React 生態整合能力，未來仍可局部客製
- 目前 dashboard 還在骨架階段，導入成本仍低

---

## 對目前同事改動的判斷

目前同事的改動不構成導入 `Refine` 的阻礙，原因如下：

- 已完成的多半是骨架，不是大量綁死的商業流程頁面
- `Layout`、頁面檔名、登入頁都可以保留或輕量調整
- 真正需要重構的主要是 auth integration 與 resource 結構

換句話說：

**現在轉向 `Refine` 的成本，明顯低於未來頁面都寫滿之後再轉。**

---

## 建議採用方式

不建議做法：

- 直接推翻整個 `dashboard/`
- 完全照搬 Refine 預設範本，不保留現有 layout

建議做法：

- 保留既有 `dashboard/` Vite 專案
- 導入 `Refine` 作為 app framework
- 保留現有 `Layout` 視覺骨架
- 把 auth 改寫為 `authProvider`
- 把 `Streamers`、`Transactions`、`Settings` 轉為第一波 resources

---

## MVP 階段的頁面適配度

以下頁面很適合優先走 Refine：

- `Streamers`
  - 列表、建立、編輯、狀態管理
- `Transactions`
  - 列表、篩選、查詢
- `Settings`
  - 設定表單
- 未來的 `Agencies`
  - 標準 CRUD 型頁面

以下頁面可先保留自寫或晚一點再整合：

- `Dashboard` 總覽頁
  - 若以 KPI 卡片、摘要資訊為主，可先自寫
- 未來若有複雜營運流程頁
  - 例如多步驟操作、特殊批次任務、複合圖表頁

---

## 推薦決策

### 建議採納

1. 將 `Refine.dev` 視為 MVP 階段 dashboard 主框架
2. 保留既有 React 專案結構，不另起新專案
3. 先用 `Refine` 解決 auth、resource、list/edit/create 的基本問題
4. 自訂程度先控制在必要範圍

### 不建議現在投入的方向

- 大量手刻 table/form infra
- 為了未來可能需求而過早最佳化 dashboard 架構
- 在還沒有完整 CRUD 頁面前，就自行打造完整設計系統

---

## 下一步

建議接著執行：

1. 建立 Refine MVP 導入計畫
2. 決定第一波 resource：
   - `streamers`
   - `transactions`
   - `settings`
3. 重構 auth flow：
   - token 持久化
   - refresh 機制
   - `authProvider`
4. 再進行第一波頁面實作

對應的實作規劃見：

- [`plans/refine-dashboard-mvp.md`](/Users/tachikoma/Documents/Web3/tachigo/plans/refine-dashboard-mvp.md)
