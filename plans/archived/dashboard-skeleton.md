# Dashboard Frontend Workplan

## 目標
在 repo 根目錄建立 `dashboard/`，完成一個可啟動、可路由、可進入受保護頁面的後台管理介面前端骨架，並補上本機開發與 Docker 開發配置。

## 專案背景
- 專案名稱：`tachigo`
- 產品定位：Twitch 直播 + Web3 獎勵平台
- 既有結構：
  - `backend/`：Go + Gin API
  - `tachimint/`：Twitch 擴充套件，使用 React 19 + TypeScript + Vite + pnpm
  - `dashboard/`：本次新增的後台管理介面

## 範圍
本次只做前端骨架與本機開發配置，不做實際登入 API、資料串接或業務邏輯。

## 技術決策
- React 19
- TypeScript
- Vite
- pnpm `10.33.0`
- Tailwind CSS v4（CSS-based config）
- React Router v7 Library Mode（`createBrowserRouter`）
- shadcn/ui 元件原始碼手動放入 `src/components/ui/`

## 交付物
- `dashboard/` 完整前端骨架
- `docker-compose.override.yml`
- 可通過 `pnpm lint`
- 可通過 `pnpm build`
- 可在 `http://localhost:5174` 啟動開發環境

## 預計檔案結構

```text
dashboard/
├── index.html
├── package.json
├── vite.config.ts
├── tsconfig.json
├── tsconfig.app.json
├── tsconfig.node.json
├── postcss.config.js
├── components.json
├── .env.example
├── Dockerfile
└── src/
    ├── main.tsx
    ├── index.css
    ├── App.tsx
    ├── lib/
    │   └── utils.ts
    ├── services/
    │   └── api.ts
    ├── components/
    │   ├── Layout.tsx
    │   ├── ProtectedRoute.tsx
    │   └── ui/
    │       ├── button.tsx
    │       ├── input.tsx
    │       └── label.tsx
    └── pages/
        ├── LoginPage.tsx
        ├── DashboardPage.tsx
        ├── StreamersPage.tsx
        ├── TransactionsPage.tsx
        └── SettingsPage.tsx
```

## 執行階段

### Phase 1: 對齊既有前端設定
- 參考 `tachimint/package.json` 對齊 React、TypeScript、Vite 與 pnpm 版本
- 複用 `tachimint` 的 tsconfig references 結構
- 從既有前端專案借用 axios client 寫法，抽出 dashboard 需要的最小版本

### Phase 2: 建立 dashboard 專案骨架
- 建立 `dashboard/` 目錄與必要設定檔
- 配置 `vite.config.ts` 的 `@` alias、`host: true`、`port: 5174`
- 配置 Tailwind v4 與 PostCSS
- 建立 `.env.example` 與 Dockerfile

### Phase 3: 建立基礎 UI 與路由
- 建立 `main.tsx`、`App.tsx`、`index.css`
- 使用 `createBrowserRouter` + `RouterProvider` 定義路由
- 建立 `ProtectedRoute.tsx`，以 localStorage `token` 判斷登入狀態
- 建立 `Layout.tsx`，包含 Sidebar、Header、`<Outlet />`
- 建立 5 個頁面元件：
  - `LoginPage`
  - `DashboardPage`
  - `StreamersPage`
  - `TransactionsPage`
  - `SettingsPage`

### Phase 4: 建立基礎元件與共用工具
- 建立 `src/lib/utils.ts` 的 `cn()` helper
- 手動加入 shadcn/ui 的 `button.tsx`、`input.tsx`、`label.tsx`
- 確保元件寫法與 Tailwind v4 相容

### Phase 5: 開發環境與容器整合
- 撰寫 `docker-compose.override.yml`
- 讓 dashboard 可透過 `docker compose up dashboard` 啟動
- 確認 volume 掛載能支援 Vite 開發

### Phase 6: 驗證與收尾
- 驗證路由切換與保護路由行為
- 驗證 active NavLink 樣式
- 執行 lint 與 build
- 修正 TypeScript、路由或樣式問題直到通過

## 路由設計
- `/login`：公開頁面，顯示登入表單 UI
- `/`：受保護頁面入口
- `/streamers`：受保護頁面
- `/transactions`：受保護頁面
- `/settings`：受保護頁面

受保護路徑結構：
- `/` → `ProtectedRoute` → `Layout` → 子頁面

## 具體實作要求

### `package.json`
- `name` 設為 `dashboard`
- 新增依賴：
  - `react-router@^7`
  - `tailwindcss@^4`
  - `@tailwindcss/postcss`
  - `clsx`
  - `tailwind-merge`
  - `class-variance-authority`
  - `lucide-react`
  - `@radix-ui/react-slot`
- 新增 devDependencies：
  - `autoprefixer`
  - `postcss`
- 不包含 `@types/twitch-ext`

### `src/services/api.ts`
- 依 `VITE_API_URL` 設定 base URL
- 保留 `client`
- 保留 `setAuthToken()`
- 不包含 Twitch Extension 專屬方法

### `src/components/Layout.tsx`
- 左側固定寬度 Sidebar
- 使用 `NavLink` 呈現四個選單
- active 狀態要有明顯樣式差異
- 右側 Header 顯示當前頁面名稱

### `src/pages/LoginPage.tsx`
- 置中卡片式登入表單
- 包含 email、password、登入按鈕
- 不做 API 串接

## 驗收清單
- [ ] `pnpm dev` 可啟動且首頁可開啟
- [ ] `/login` 可顯示登入表單
- [ ] 未登入訪問 `/` 會導向 `/login`
- [ ] 手動設定 `localStorage.token` 後可進入 `/`
- [ ] Sidebar 的 active 樣式正確
- [ ] `/streamers`、`/transactions`、`/settings` 可正常切換
- [ ] `pnpm lint` 通過
- [ ] `pnpm build` 通過
- [ ] `docker compose up dashboard` 可啟動並連到 5174 port

## 風險與注意事項
- `tachimint` 若有客製 tsconfig 或 lint 規則，dashboard 需對齊避免版本衝突
- shadcn/ui 採手動複製，需注意 import 路徑與 Tailwind v4 相容性
- React Router v7 必須使用 Library Mode，不可混入 Framework Mode 寫法
- Tailwind v4 不使用 `tailwind.config.js`
- 全程使用 `pnpm`，不要混用 `npm`
- 目前登入流程只做 UI 與 route guard，避免提前引入不必要 API 依賴

## Definition of Done
當以下條件同時成立，即視為完成：
- dashboard 專案骨架存在且檔案結構完整
- 本機啟動、路由、保護頁面、側邊欄切換都正常
- lint 與 build 均通過
- Docker 開發配置可運作
