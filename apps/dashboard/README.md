# Tachigo Dashboard

Tachigo Dashboard 是 Vite + React + TypeScript 應用，管理介面。

## 啟動方式

```bash
pnpm install
pnpm dev
```

預設 Vite dev server 使用 `5174`。API base URL 使用 `VITE_API_URL`，未設定時回退到 `http://localhost:8080`。

## 目前架構

`src/App.tsx` 使用 React Router v7 `createBrowserRouter`，並以 `ProtectedRoute`（驗證 `isAuthenticated()`）保護需要登入的路由。

認證邏輯集中在 `src/services/auth.ts`，使用 in-memory access token + httpOnly refresh cookie。`src/main.tsx` 在 app 啟動時呼叫 `restoreSession()` 取回 access token；各頁面的 API request path 由各 service 函式自行帶入 `/api/v1` prefix。

## 規劃中：Refine.dev 架構遷移

以下異動正在進行，尚未合併到 develop（追蹤 #456）：

- `src/providers/authProvider.ts`：為 Refine `<Authenticated>` 提供 `check()` 介面；`check()` 內部同樣呼叫 `restoreSession()`，`main.tsx` 的啟動呼叫不變
- `src/providers/dataProvider.ts`：以 `@refinedev/simple-rest` 為 base，實際 request 使用既有 axios client；`/api/v1` prefix 作為 base URL 的一部分（非中介層自動注入）
- `src/App.tsx`：改以 `<Refine>` 包住 React Router，resources 定義 `streamers`、`raffles`、`transactions`、`settings`
- dataProvider base URL 使用 `VITE_API_URL`，未設定時回退到 `http://localhost:8080`，並加上 `/api/v1` 作為路徑前綴

## API response envelope

後端 response 格式如下（非標準 simple-rest）：

```json
{ "data": { "raffles": [] } }
```

或：

```json
{ "data": { "raffle": {} } }
```

## 常用指令

```bash
pnpm test
pnpm lint
pnpm build
```
