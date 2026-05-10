# Tachigo Dashboard

`apps/dashboard` 是給 Streamer / Agency / Admin 使用的後台管理介面，負責查看營運概況、管理 streamers、設定頻道獎勵、檢視交易紀錄與操作 raffle 流程。

## 快速啟動

需求：

- Node.js / pnpm：以 [`package.json`](package.json) 的 `packageManager` 為準
- 後端 API：預設連到 `http://localhost:8080`

```bash
cd apps/dashboard
pnpm install
pnpm dev
```

Vite dev server 預設在：

```text
http://localhost:5174
```

常用指令：

```bash
pnpm dev
pnpm build
pnpm test
pnpm lint
pnpm preview
```

也可以從 repo root 用 Docker Compose 啟動完整 stack：

```bash
make dev
```

## 連接後端

建立本機 env：

```bash
cp apps/dashboard/.env.example apps/dashboard/.env
```

目前支援的 API base URL 變數：

| 變數 | 用途 |
| --- | --- |
| `VITE_TACHIGO_API_URL` | Tachigo API origin，例如 `http://localhost:8080` |
| `VITE_API_URL` | 舊本機 env 的 fallback key；新設定請優先使用 `VITE_TACHIGO_API_URL` |

Dashboard API client 會自行帶上 `/api/v1` path prefix。登入後的 request 使用 in-memory access token，refresh flow 透過後端 httpOnly cookie。

## 主要模組

目前 router / resources 由 [`src/App.tsx`](src/App.tsx) 定義：

| 路由 | 用途 |
| --- | --- |
| `/login` | Dashboard auth |
| `/` | 營運概覽 dashboard |
| `/streamers` | Streamer list |
| `/streamers/:streamerId` | Streamer detail / stats |
| `/raffles` | Raffle list |
| `/raffles/:raffleId` | Raffle detail / draw management |
| `/transactions` | Viewer points transaction history |
| `/settings` | Reward / channel settings entry |

核心實作位置：

| 路徑 | 說明 |
| --- | --- |
| `src/pages/` | Route-level pages |
| `src/components/` | Layout、route guard 與共用 UI |
| `src/services/` | Axios client、auth、channels、raffles API calls |
| `src/providers/` | Refine authProvider / dataProvider |
| `src/test/` | Refine test wrapper |

## Refine / API response

Dashboard 目前使用 Refine + React Router。`dataProvider` 以 `@refinedev/simple-rest` 為基底，再配合既有 Axios client 對接 tachigo API。

後端 response envelope 不是標準 simple-rest 格式，常見形狀如下：

```json
{ "data": { "raffles": [] } }
```

或：

```json
{ "data": { "raffle": {} } }
```

新增 resource 時，請同步確認 `src/providers/dataProvider.ts` 的 resource path mapping 與 unwrap logic。

## 測試與 build

```bash
cd apps/dashboard
pnpm test
pnpm lint
pnpm build
```

PR 若修改頁面或 dataProvider，建議至少跑相關 Vitest，加上 `pnpm build` 確認 TypeScript 與 Vite build 都通過。
