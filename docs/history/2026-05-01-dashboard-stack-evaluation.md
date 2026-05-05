# Dashboard 技術選型評估

> 用途：記錄 dashboard 技術選型評估與 Refine.dev 採納決策。
> 狀態：歷史決策紀錄，不是 active implementation plan。
> 最後更新：2026-05-01
> 最後校正：2026-05-05（#490 docs root audit）

## 背景

目前 `tachigo` 已有一個獨立的 `apps/dashboard/` 前端專案，技術基礎如下：

- React 19 + TypeScript + Vite
- React Router
- Tailwind CSS v4
- 手動放入的 `shadcn/ui` 基礎元件
- 後端為 Go + Gin + GORM，提供 REST API

本文件記錄 dashboard 的技術選型過程與最終決策。

**決策（2026-05-01）：採用 `Refine.dev` 作為 dashboard 主框架。**

---

## 目前程式狀態摘要

### 已有基礎

- [`apps/dashboard/src/App.tsx`](../../apps/dashboard/src/App.tsx)
  - 已建立登入頁、受保護路由、Layout 與幾個基礎頁面
- [`apps/dashboard/src/components/Layout.tsx`](../../apps/dashboard/src/components/Layout.tsx)
  - 已有基本側邊欄與頁面容器
- [`apps/dashboard/src/pages/LoginPage.tsx`](../../apps/dashboard/src/pages/LoginPage.tsx)
  - 已完成登入 UI，並串接登入 API
- [`apps/dashboard/src/services/auth.ts`](../../apps/dashboard/src/services/auth.ts)
  - 已有 login/logout 邏輯與 token 設定
- [`services/api/internal/router/router.go`](../../services/api/internal/router/router.go)
  - 後端已提供 auth 與部分 dashboard/API 路由骨架

### 尚未成形的部分

- `Streamers`、`Transactions`、`Settings` 頁面目前仍是 placeholder
- 尚未看到完整 CRUD 頁面、table、form、query cache、access control 抽象
- 尚未導入 `Refine`、`React Query`、`TanStack Table` 等主要資料層工具

### 目前技術風險

- [`apps/dashboard/src/services/auth.ts`](../../apps/dashboard/src/services/auth.ts)
  - `accessToken` 僅保存在記憶體；但 `main.tsx` 啟動時會 `await restoreSession()`，透過 httpOnly refresh cookie 還原 token，頁面重整不會立即掉登入
  - Refine `authProvider` 尚未整合，目前 route guard 仍由 `ProtectedRoute` 處理
- [`apps/dashboard/src/components/ProtectedRoute.tsx`](../../apps/dashboard/src/components/ProtectedRoute.tsx)
  - 現在的 route guard 只適合非常早期原型
- [`apps/dashboard/README.md`](../../apps/dashboard/README.md)
  - 仍是 Vite 預設內容，尚未反映專案實際開發方式

---

## 選型結論

### 結論摘要

**已決定（2026-05-01）採用 `Refine.dev` 作為 dashboard framework，保留既有 React 專案與部分 UI 骨架。**

理由：dashboard 會持續成長；Refine 的 resource/provider convention 在頁面規模擴大時帶來一致性；access control、快取、分頁等功能免費獲得；複合頁面仍可在 Refine 專案內自寫，不受限制。

### 為什麼不是 Next.js

- 後端是 Go + Gin 的 API-first 架構，不需要 SSR 或 SSG
- Dashboard 是純後台管理工具，沒有 SEO 需求，server render 的優勢無法轉化為價值
- Next.js 帶來的複雜度（server actions、routing convention、需要 Node server 部署）在此場景沒有對應回報
- 初始骨架已用 Vite 建立，重來成本高

### 為什麼不是 Go Admin

- 目前已經有獨立 `apps/dashboard/` React 專案
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

- 直接推翻整個 `apps/dashboard/`
- 完全照搬 Refine 預設範本，不保留現有 layout

建議做法：

- 保留既有 `apps/dashboard/` Vite 專案
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

## 決策

### 已確認採納

1. `Refine.dev` 作為 MVP 階段 dashboard 主框架
2. 保留既有 React 專案結構，不另起新專案
3. 先用 `Refine` 解決 auth、resource、list/edit/create 的基本問題
4. 自訂程度先控制在必要範圍

### 不投入的方向

- 大量手刻 table/form infra
- 為了未來可能需求而過早最佳化 dashboard 架構
- 在還沒有完整 CRUD 頁面前，就自行打造完整設計系統

---

## 下一步

實作規劃見：

- [`plans/refine-dashboard-mvp.md`](../../plans/refine-dashboard-mvp.md)
