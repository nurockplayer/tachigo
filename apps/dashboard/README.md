# Tachigo Dashboard

Tachigo Dashboard 是 Vite + React + TypeScript 應用，管理介面以 Refine.dev 作為資料、認證與路由整合層，畫面仍使用專案既有 Tailwind / shadcn-style 元件。

## 啟動方式

```bash
pnpm install
pnpm dev
```

預設 Vite dev server 使用 `5174`。API base URL 依序讀取：

1. `VITE_TACHIGO_API_URL`
2. `VITE_API_URL`
3. `http://localhost:8080`

所有 API request 會再加上 `/api/v1` prefix。

## Refine 架構

`src/App.tsx` 以 `<Refine>` 包住 React Router v7 `createBrowserRouter`，並傳入：

- `src/providers/authProvider.ts`：橋接既有 `services/auth.ts`，登入、登出、啟動時 session restore、JWT role / identity 解析都集中在這裡。
- `src/providers/dataProvider.ts`：以 `@refinedev/simple-rest` 為 base，實際 request 使用既有 axios client，因此會沿用 in-memory access token、httpOnly refresh cookie 與 401 refresh interceptor。
- Refine resources：`streamers`、`raffles`、`transactions`、`settings`。首頁 `DashboardPage` 保持自寫頁面，不走 resource。

受保護路由使用 Refine `<Authenticated>`，未登入會導向 `/login`。`authProvider.check()` 是 async，會呼叫 `restoreSession()` 讓 app 啟動時可以用 refresh cookie 換回 access token。

## API response envelope

後端 response 不是標準 simple-rest 格式，常見格式如下：

```json
{ "data": { "raffles": [] } }
```

或：

```json
{ "data": { "raffle": {} } }
```

`dataProvider` 會把 envelope 轉成 Refine hooks 需要的 `{ data, total }` 或 `{ data }`，讓 `useList` / `useOne` 可直接在頁面使用。

## 常用指令

```bash
pnpm test
pnpm lint
pnpm build
```
