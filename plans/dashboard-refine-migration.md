# Dashboard — 遷移至 Refine 框架

狀態：已完成

## 背景

Dashboard 未來會持續新增 CRUD 頁面（實況主管理、交易記錄、空投管理等），
在程式碼量還少的現在導入 Refine，可以減少未來每個頁面的重複樣板程式碼。

Refine 負責：authProvider（自動導航）、dataProvider（API 抓取）
TanStack Table / shadcn/ui 繼續負責：表格邏輯、視覺元件

## 相容性確認

- `@refinedev/react-router` peerDependency：`react-router ^7.0.2` ✅
- `@refinedev/core` 最新版：`5.0.12`
- React 19 ✅

## 套件安裝

```bash
cd dashboard
pnpm add @refinedev/core @refinedev/react-router
```

## 架構決策

### 保留不動
- `src/services/api.ts` — axios client 繼續使用
- 所有 shadcn/ui 元件
- `LoginPage.tsx`、`DashboardPage.tsx` 等頁面元件

### 需要改寫

| 檔案 | 變更說明 |
|---|---|
| `src/App.tsx` | 包上 `<Refine>`，路由改用 `<RefineRoutes>` |
| `src/services/auth.ts` | 移植成 `authProvider` 物件 |
| `src/components/ProtectedRoute.tsx` | 改用 `<Authenticated>` 取代 |
| `src/services/dataProvider.ts` | 新增，包裝 axios client |

## 實作 Checklist

### 1. 安裝套件
- [x] `pnpm add @refinedev/core @refinedev/react-router`

### 2. 建立 authProvider（`src/services/authProvider.ts`）

從現有 `auth.ts` 移植，需實作以下方法：

```ts
import { AuthProvider } from '@refinedev/core'

export const authProvider: AuthProvider = {
  login: async ({ email, password }) => { ... },   // 對應現有 login()
  logout: async () => { ... },                      // 對應現有 logout()
  check: async () => { ... },                       // 對應現有 isAuthenticated()
  getPermissions: async () => { ... },              // 回傳 role（Streamer/Agency/Admin）
  onError: async (error) => { ... },
}
```

`getPermissions` 需要從 JWT payload 解出 role，是新增邏輯。

### 3. 建立 dataProvider（`src/services/dataProvider.ts`）

```ts
import { DataProvider } from '@refinedev/core'
import client from './api'

export const dataProvider: DataProvider = {
  getList: async ({ resource, pagination, filters }) => { ... },
  getOne: async ({ resource, id }) => { ... },
  // create / update / deleteOne 之後再補
}
```

### 4. 改寫 App.tsx

```tsx
import { Refine } from '@refinedev/core'
import { BrowserRouter, Routes, Route } from 'react-router'
import routerProvider from '@refinedev/react-router'
import { authProvider } from '@/services/authProvider'
import { dataProvider } from '@/services/dataProvider'

export default function App() {
  return (
    <BrowserRouter>
      <Refine
        authProvider={authProvider}
        dataProvider={dataProvider}
        routerProvider={routerProvider}
      >
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<Authenticated><Layout /></Authenticated>}>
            <Route index element={<DashboardPage />} />
            <Route path="streamers" element={<StreamersPage />} />
            <Route path="streamers/:streamerId" element={<StreamerDetailPage />} />
            <Route path="transactions" element={<TransactionsPage />} />
            <Route path="settings" element={<SettingsPage />} />
          </Route>
        </Routes>
      </Refine>
    </BrowserRouter>
  )
}
```

### 5. 移除 ProtectedRoute.tsx
- [x] `<Authenticated>` 取代 `<ProtectedRoute>`，完成後刪除舊檔案

### 6. 驗證
- [x] 登入流程正常（LoginPage → Dashboard）
- [x] 未登入直接進 `/` 會跳回 `/login`
- [x] logout 後清除 token 並導向 `/login`
- [x] `getPermissions` 能正確回傳 role

## Streamer 角色導向（與 #69 銜接）

`authProvider.getPermissions()` 回傳 role 後，可以在 `<Authenticated>` 的 `fallback` 或 `StreamersPage` 內用 `useGetIdentity()` / `usePermissions()` hook 判斷：
- `role === 'streamer'` → redirect 到 `/streamers/:self_id`
- 其他 → 顯示列表

## 驗證方式

```bash
pnpm dev
# 1. 未登入 → 自動跳 /login
# 2. 登入後 → 進 /
# 3. logout → 跳回 /login
# 4. Streamer 帳號登入 → 跳 /streamers/:id（#69 實作後驗證）
```

## 參考
- Refine 官方文件：https://refine.dev/docs
- `@refinedev/react-router` 整合：https://refine.dev/docs/routing/integrations/react-router
