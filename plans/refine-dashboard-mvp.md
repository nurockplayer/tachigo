# Refine Dashboard MVP 導入計畫

## 狀態

提案中

## 目標

在不推翻既有 `dashboard/` 專案的前提下，導入 `Refine.dev` 作為 MVP 階段的 dashboard framework，快速完成：

- 登入後台基礎流程
- 受保護路由
- 基本 resource 管理頁
- 後端 REST API 串接

---

## 專案背景

目前 repo 結構：

- `backend/`：Go + Gin + GORM
- `tachimint/`：Twitch Extension 前端
- `dashboard/`：React 後台骨架

目前 `dashboard/` 已經有：

- React Router 路由骨架
- 基本 Layout
- Login UI
- axios client
- 基本 auth service

但尚未有：

- 可持久化的登入狀態
- 資料抓取管理慣例
- resource-based CRUD 頁面
- 完整 table/form/query workflow

---

## 技術決策

### 採用

- `@refinedev/core`
- `@refinedev/react-router`
- `@refinedev/simple-rest` 或自訂 data provider
- `@refinedev/react-hook-form`
- `react-hook-form`
- `zod`
- `@hookform/resolvers`

### 暫不強制導入

- `@tanstack/react-table`
  - 先使用 Refine 內建較快的 list flow
  - 等真的需要更高彈性再補
- 全面重做 UI
  - 先保留既有 layout 與頁面容器

### UI 策略

- 保留既有 `shadcn/ui` 元件
- `Refine` 解決框架層與 CRUD 慣例
- 視覺層維持目前專案風格，不追求一次重做

---

## 預計範圍

### Phase 1: Foundation

- 安裝 Refine 相關套件
- 建立 `Refine` 根層設定
- 接上 React Router
- 建立 `authProvider`
- 建立 `dataProvider`

### Phase 2: Auth 與 Session

- 重構 [`dashboard/src/services/auth.ts`](/Users/tachikoma/Documents/Web3/tachigo/dashboard/src/services/auth.ts)
- 支援：
  - access token 持久化策略
  - refresh token 使用策略
  - app 啟動時還原登入狀態
- 將 `ProtectedRoute` 遷移為 Refine auth flow

### Phase 3: Resources

第一波 resources：

- `streamers`
- `transactions`
- `settings`

每個 resource 至少包含：

- list
- edit 或 create
- 對應 route

### Phase 4: 保留式整合

- 盡量保留現有 [`dashboard/src/components/Layout.tsx`](/Users/tachikoma/Documents/Web3/tachigo/dashboard/src/components/Layout.tsx)
- 將現有 nav 與 Refine resource route 對齊
- 保留 [`dashboard/src/pages/DashboardPage.tsx`](/Users/tachikoma/Documents/Web3/tachigo/dashboard/src/pages/DashboardPage.tsx) 作為自寫總覽頁

### Phase 5: 文件與驗證

- 更新 `dashboard/README.md`
- 補上 dashboard 啟動與架構說明
- 驗證基本登入、跳轉、resource 頁面流程

---

## 建議檔案結構

```text
dashboard/src/
├── main.tsx
├── App.tsx
├── components/
│   ├── Layout.tsx
│   └── ui/
├── pages/
│   ├── DashboardPage.tsx
│   ├── LoginPage.tsx
│   ├── streamers/
│   │   ├── list.tsx
│   │   ├── create.tsx
│   │   └── edit.tsx
│   ├── transactions/
│   │   └── list.tsx
│   └── settings/
│       └── edit.tsx
├── providers/
│   ├── authProvider.ts
│   └── dataProvider.ts
├── services/
│   ├── api.ts
│   └── auth.ts
└── types/
```

---

## 第一波頁面策略

### `Dashboard`

- 保持自寫
- 放 KPI、摘要資訊、後續報表卡片

### `Streamers`

- 優先導入為 Refine resource
- MVP 目標：
  - 列表
  - 建立
  - 編輯

### `Transactions`

- 優先導入為 Refine resource
- MVP 目標：
  - 列表
  - 篩選條件可先簡化

### `Settings`

- 可作為單一 edit/view 頁處理
- MVP 先做最小設定集合

---

## 主要風險

### 1. Backend API 尚未完全齊備

目前有些 dashboard / admin / agency 路由仍回傳 `501 not implemented`：

- [`backend/internal/router/router.go`](/Users/tachikoma/Documents/Web3/tachigo/backend/internal/router/router.go)

影響：

- 前端 resource 可先搭骨架
- 真正串接前需先補齊 API 或用 mock/placeholder 資料

### 2. 現有 auth 設計不足以支援穩定 dashboard 體驗

目前 token 僅存在記憶體，重新整理會掉登入狀態。

影響：

- 在導入 Refine 前，應先一起整理 auth flow

### 3. Settings 頁不一定是典型 CRUD

若未來 `Settings` 變成複合設定頁，可能需要保留較多自寫邏輯。

---

## 驗收標準

- [ ] `dashboard/` 可正常啟動
- [ ] 登入後可進入受保護頁面
- [ ] 重新整理後登入狀態不會立即失效
- [ ] `streamers` resource 可顯示 list 頁
- [ ] `transactions` resource 可顯示 list 頁
- [ ] `settings` 頁可作為 Refine route 或整合式頁面進入
- [ ] Layout 與側邊欄仍正常運作
- [ ] `dashboard/README.md` 已更新

---

## 建議執行順序

1. 安裝 Refine 相關套件
2. 建立 `authProvider` / `dataProvider`
3. 修正 token 持久化與 refresh flow
4. 將路由接入 Refine
5. 實作 `streamers`
6. 實作 `transactions`
7. 整理 `settings`
8. 更新 README 與開發文件

---

## 備註

本計畫的核心原則是：

**優先提升 MVP 交付速度，而不是追求最高自由度。**

若未來 dashboard 開始出現大量高客製頁面，再逐步將特定頁面從 Refine 慣例中抽離即可。
