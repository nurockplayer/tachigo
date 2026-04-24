# Dashboard Auth Cookie Migration (Phase 2 + 3) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 完成 issue #217 剩餘三個 checklist：移除 dashboard `localStorage` refresh token、新增 401 interceptor + promise deduplication、補上測試、更新 auth docs。

**Architecture:**
後端（PR #220 已 merge）在 login/refresh/logout 時以 httpOnly cookie 管理 refresh token，並保留 body fallback。前端只需移除主動讀寫 localStorage 的邏輯，讓 `withCredentials: true` 自動攜帶 cookie，並補上 401 interceptor 以便在 access token 過期時自動靜默刷新。`accessToken` 的單一來源從 `auth.ts` 移至 `api.ts`（透過 `hasAuthToken()` export），避免 interceptor 更新 header 後 `isAuthenticated()` 讀到過期值。頁面重整時透過 `restoreSession()` 在 React mount 前呼叫 `/auth/refresh`，以 cookie 靜默還原 session。

**Tech Stack:** React, TypeScript, Axios 1.x, Vitest + jsdom, axios-mock-adapter（新增 devDep）

---

## 安全背景（寫計劃前的評估摘要）

### 後端已確認
| 面向 | 狀態 |
|---|---|
| httpOnly cookie | ✅ PR #220 |
| SameSite=Lax（預設）/ SameSite=None（跨域 prod） | ✅ PR #220 |
| Path=/api/v1/auth | ✅ PR #220 |
| Secure 依 APP_ENV | ✅ PR #220 |
| CORS AllowCredentials: true + 非 wildcard origin | ✅ middleware/cors.go |
| Token rotation（每次 refresh 刪舊建新） | ✅ auth_service.go:168-170 |
| Body fallback（過渡期） | ✅ PR #220 |

### 前端必做（本計劃範圍）
- `withCredentials: true`：browser 才會自動帶 cookie
- 401 dedupe：rotation 下若 N 個並發請求同時過期，只能 refresh 一次，否則第 2-N 個會因 refresh token 已輪換而 401 並踢出用戶
- 移除 localStorage：消除 XSS 竊取 refresh token 的攻擊面
- 刷新 endpoint 不進入 retry loop：`/api/v1/auth/refresh` 本身若 401 不應再觸發 interceptor

### 前端不需要做
- CSRF token：SameSite=Lax 已保護 POST 路徑（auth 端點均為 POST）
- 自行管理 refresh token 值：httpOnly cookie 對 JS 不可見，backend 全權管理

---

## 檔案異動總覽

| 檔案 | 動作 | 說明 |
|---|---|---|
| `dashboard/src/services/api.ts` | Modify | 加 `withCredentials`、`hasAuthToken()`、401 interceptor |
| `dashboard/src/services/auth.ts` | Modify | 移除 localStorage、改用 `hasAuthToken()`、加 `refresh()` / `restoreSession()` |
| `dashboard/src/main.tsx` | Modify | bootstrap：mount 前呼叫 `restoreSession()` |
| `dashboard/src/services/__tests__/auth.test.ts` | Create | auth service 新行為測試 |
| `dashboard/src/services/__tests__/api.interceptor.test.ts` | Create | 401 dedupe interceptor 測試 |
| `docs/auth-architecture.md` | Modify | 標記 Phase 1 + Phase 2 完成，補 migration notes |

---

## Task 1：重構 `api.ts` — 加 `hasAuthToken()`、`withCredentials`、401 interceptor

### 為什麼要先改 api.ts

`auth.ts` 的 `isAuthenticated()` 目前讀 auth.ts 自有的 `accessToken` 變數，但 401 interceptor 在 api.ts 裡更新 header 時不會同步到這個變數。解法是把 token 的存在性查詢移到 api.ts（`hasAuthToken()`），讓 auth.ts delegate 給它。此外，interceptor 呼叫 `/auth/refresh` 時需要避免無窮迴圈（refresh 本身若 401 不能再 retry）。

**Files:**
- Modify: `dashboard/src/services/api.ts`
- Create: `dashboard/src/services/__tests__/api.interceptor.test.ts`

### 安裝 axios-mock-adapter（測試用）

- [ ] **Step 1: 安裝 devDependency**

```bash
cd dashboard
pnpm add -D axios-mock-adapter
```

Expected：`package.json` devDependencies 出現 `"axios-mock-adapter"`。

### 寫失敗的 interceptor 測試

- [ ] **Step 2: 建立測試檔，先讓它失敗**

建立 `dashboard/src/services/__tests__/api.interceptor.test.ts`：

```typescript
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import MockAdapter from 'axios-mock-adapter'
import client, { hasAuthToken, setAuthToken, clearAuthToken } from '@/services/api'

let mock: InstanceType<typeof MockAdapter>

beforeEach(() => {
  mock = new MockAdapter(client)
  clearAuthToken()
})

afterEach(() => {
  mock.restore()
  clearAuthToken()
})

describe('401 interceptor', () => {
  it('access token 過期時自動刷新並重試原始請求', async () => {
    setAuthToken('expired-token')

    mock
      .onGet('/api/v1/some-resource')
      .replyOnce(401)
      .onGet('/api/v1/some-resource')
      .replyOnce(200, { data: 'ok' })
    mock
      .onPost('/api/v1/auth/refresh')
      .replyOnce(200, { data: { tokens: { access_token: 'new-token' } } })

    const result = await client.get('/api/v1/some-resource')

    expect(result.data).toEqual({ data: 'ok' })
    expect(hasAuthToken()).toBe(true)
  })

  it('並發多個 401 時只呼叫一次 /auth/refresh（dedupe）', async () => {
    setAuthToken('expired-token')

    mock
      .onGet('/api/v1/resource-a')
      .replyOnce(401)
      .onGet('/api/v1/resource-a')
      .replyOnce(200, { data: 'a' })
    mock
      .onGet('/api/v1/resource-b')
      .replyOnce(401)
      .onGet('/api/v1/resource-b')
      .replyOnce(200, { data: 'b' })
    mock
      .onPost('/api/v1/auth/refresh')
      .replyOnce(200, { data: { tokens: { access_token: 'new-token' } } })

    const [a, b] = await Promise.all([
      client.get('/api/v1/resource-a'),
      client.get('/api/v1/resource-b'),
    ])

    expect(a.data).toEqual({ data: 'a' })
    expect(b.data).toEqual({ data: 'b' })
    // refresh 只被呼叫一次（mock 只設了一次成功回應）
    expect(mock.history.post.filter(r => r.url === '/api/v1/auth/refresh')).toHaveLength(1)
  })

  it('/auth/refresh 本身 401 時不觸發 retry loop，直接拋出', async () => {
    setAuthToken('expired-token')

    mock.onGet('/api/v1/some-resource').replyOnce(401)
    mock.onPost('/api/v1/auth/refresh').replyOnce(401)

    await expect(client.get('/api/v1/some-resource')).rejects.toMatchObject({
      response: { status: 401 },
    })
    // refresh 只被呼叫一次，沒有遞迴
    expect(mock.history.post.filter(r => r.url === '/api/v1/auth/refresh')).toHaveLength(1)
  })

  it('非 401 錯誤不觸發 refresh', async () => {
    mock.onGet('/api/v1/some-resource').replyOnce(500)

    await expect(client.get('/api/v1/some-resource')).rejects.toMatchObject({
      response: { status: 500 },
    })
    expect(mock.history.post.filter(r => r.url === '/api/v1/auth/refresh')).toHaveLength(0)
  })
})

describe('hasAuthToken', () => {
  it('無 token 時回傳 false', () => {
    expect(hasAuthToken()).toBe(false)
  })

  it('setAuthToken 後回傳 true', () => {
    setAuthToken('some-token')
    expect(hasAuthToken()).toBe(true)
  })

  it('clearAuthToken 後回傳 false', () => {
    setAuthToken('some-token')
    clearAuthToken()
    expect(hasAuthToken()).toBe(false)
  })
})
```

- [ ] **Step 3: 確認測試失敗（`hasAuthToken` 尚不存在）**

```bash
cd dashboard && pnpm test -- api.interceptor
```

Expected：測試 fail，錯誤為 `hasAuthToken is not exported` 或類似。

### 實作 api.ts

- [ ] **Step 4: 更新 `dashboard/src/services/api.ts`**

```typescript
import axios, { type AxiosRequestConfig } from 'axios'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

const client = axios.create({
  baseURL: BASE_URL,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
})

let _accessToken: string | null = null

export function setAuthToken(token: string) {
  _accessToken = token
  client.defaults.headers.common['Authorization'] = `Bearer ${token}`
}

export function clearAuthToken() {
  _accessToken = null
  delete client.defaults.headers.common['Authorization']
}

export function hasAuthToken(): boolean {
  return _accessToken !== null
}

// 正在進行中的 refresh promise（dedupe 用）
let _refreshPromise: Promise<void> | null = null

interface RefreshResponse {
  data: { tokens: { access_token: string } }
}

client.interceptors.response.use(
  response => response,
  async (error) => {
    const originalRequest = error.config as AxiosRequestConfig & { _retry?: boolean }

    const isRefreshEndpoint = (originalRequest.url ?? '').includes('/api/v1/auth/refresh')
    if (
      error.response?.status !== 401 ||
      originalRequest._retry ||
      isRefreshEndpoint
    ) {
      return Promise.reject(error)
    }

    originalRequest._retry = true

    if (!_refreshPromise) {
      _refreshPromise = client
        .post<RefreshResponse>('/api/v1/auth/refresh')
        .then(({ data }) => {
          setAuthToken(data.data.tokens.access_token)
        })
        .catch((refreshError) => {
          clearAuthToken()
          throw refreshError
        })
        .finally(() => {
          _refreshPromise = null
        })
    }

    try {
      await _refreshPromise
    } catch {
      return Promise.reject(error)
    }

    return client(originalRequest)
  },
)

export default client
```

- [ ] **Step 5: 執行 interceptor 測試，確認全部通過**

```bash
cd dashboard && pnpm test -- api.interceptor
```

Expected：全部 PASS。

- [ ] **Step 6: Commit**

```bash
git add dashboard/src/services/api.ts \
        dashboard/src/services/__tests__/api.interceptor.test.ts \
        dashboard/package.json \
        dashboard/pnpm-lock.yaml
git commit -m "feat(dashboard): add withCredentials, hasAuthToken, and 401 refresh interceptor with dedupe

refs #217

Co-Authored-By: Claude Sonnet 4.6 <claude[bot]@anthropic.com>"
```

---

## Task 2：重寫 `auth.ts` — 移除 localStorage，加 `refresh()` / `restoreSession()`

**Files:**
- Modify: `dashboard/src/services/auth.ts`
- Create: `dashboard/src/services/__tests__/auth.test.ts`

### 寫失敗的 auth 測試

- [ ] **Step 1: 建立測試檔**

建立 `dashboard/src/services/__tests__/auth.test.ts`：

```typescript
import { beforeEach, describe, expect, it, vi } from 'vitest'
import MockAdapter from 'axios-mock-adapter'
import client, { hasAuthToken, clearAuthToken } from '@/services/api'
import { login, logout, refresh, restoreSession, isAuthenticated } from '@/services/auth'

let mock: InstanceType<typeof MockAdapter>

beforeEach(() => {
  mock = new MockAdapter(client)
  clearAuthToken()
  localStorage.clear()
})

afterEach(() => {
  mock.restore()
})

describe('login()', () => {
  it('成功時設定 Authorization header', async () => {
    mock.onPost('/api/v1/auth/login').replyOnce(200, {
      data: {
        user: { id: 'u1' },
        tokens: { access_token: 'access-abc', refresh_token: 'refresh-xyz' },
      },
    })

    await login('user@example.com', 'password')

    expect(hasAuthToken()).toBe(true)
    expect(isAuthenticated()).toBe(true)
  })

  it('不將 refresh_token 寫入 localStorage', async () => {
    mock.onPost('/api/v1/auth/login').replyOnce(200, {
      data: {
        user: { id: 'u1' },
        tokens: { access_token: 'access-abc', refresh_token: 'refresh-xyz' },
      },
    })

    await login('user@example.com', 'password')

    expect(localStorage.getItem('refresh_token')).toBeNull()
  })

  it('API 失敗時 throw error', async () => {
    mock.onPost('/api/v1/auth/login').replyOnce(401, { error: 'invalid credentials' })

    await expect(login('user@example.com', 'wrong')).rejects.toThrow()
    expect(isAuthenticated()).toBe(false)
  })
})

describe('logout()', () => {
  it('呼叫 /auth/logout 並清除 Authorization header', async () => {
    // 先 login
    mock.onPost('/api/v1/auth/login').replyOnce(200, {
      data: { user: {}, tokens: { access_token: 'access-abc', refresh_token: 'r' } },
    })
    await login('u@e.com', 'pw')

    mock.onPost('/api/v1/auth/logout').replyOnce(200)

    await logout()

    expect(hasAuthToken()).toBe(false)
    expect(isAuthenticated()).toBe(false)
  })

  it('不送 refresh_token body（由 cookie 處理）', async () => {
    mock.onPost('/api/v1/auth/login').replyOnce(200, {
      data: { user: {}, tokens: { access_token: 'access-abc', refresh_token: 'r' } },
    })
    await login('u@e.com', 'pw')

    mock.onPost('/api/v1/auth/logout').replyOnce(200)

    await logout()

    const logoutCall = mock.history.post.find(r => r.url === '/api/v1/auth/logout')
    expect(logoutCall).toBeDefined()
    // body 應為 null / 空（不含 refresh_token）
    expect(logoutCall?.data ?? null).toBeFalsy()
  })

  it('logout API 失敗時仍清除本機狀態（fire and forget）', async () => {
    mock.onPost('/api/v1/auth/login').replyOnce(200, {
      data: { user: {}, tokens: { access_token: 'access-abc', refresh_token: 'r' } },
    })
    await login('u@e.com', 'pw')

    mock.onPost('/api/v1/auth/logout').replyOnce(500)

    await logout() // 不應拋出

    expect(isAuthenticated()).toBe(false)
  })

  it('不讀取也不清除 localStorage', async () => {
    localStorage.setItem('refresh_token', 'old-token')
    mock.onPost('/api/v1/auth/login').replyOnce(200, {
      data: { user: {}, tokens: { access_token: 'access-abc', refresh_token: 'r' } },
    })
    await login('u@e.com', 'pw')
    mock.onPost('/api/v1/auth/logout').replyOnce(200)

    await logout()

    // localStorage 內容不受 logout 影響（它不是我們的責任了）
    expect(localStorage.getItem('refresh_token')).toBe('old-token')
  })
})

describe('refresh()', () => {
  it('成功時更新 Authorization header', async () => {
    mock.onPost('/api/v1/auth/refresh').replyOnce(200, {
      data: { tokens: { access_token: 'new-access-token' } },
    })

    await refresh()

    expect(hasAuthToken()).toBe(true)
    expect(isAuthenticated()).toBe(true)
  })

  it('失敗時 throw error 且不設 token', async () => {
    mock.onPost('/api/v1/auth/refresh').replyOnce(401)

    await expect(refresh()).rejects.toThrow()
    expect(isAuthenticated()).toBe(false)
  })
})

describe('restoreSession()', () => {
  it('refresh 成功時 isAuthenticated() 變為 true', async () => {
    mock.onPost('/api/v1/auth/refresh').replyOnce(200, {
      data: { tokens: { access_token: 'restored-token' } },
    })

    await restoreSession()

    expect(isAuthenticated()).toBe(true)
  })

  it('refresh 失敗時靜默處理，isAuthenticated() 仍為 false', async () => {
    mock.onPost('/api/v1/auth/refresh').replyOnce(401)

    await expect(restoreSession()).resolves.toBeUndefined()
    expect(isAuthenticated()).toBe(false)
  })
})
```

- [ ] **Step 2: 確認測試失敗（`refresh`, `restoreSession` 尚不存在）**

```bash
cd dashboard && pnpm test -- auth.test
```

Expected：多個 FAIL（import 錯誤 / 行為不符）。

### 實作新版 auth.ts

- [ ] **Step 3: 改寫 `dashboard/src/services/auth.ts`**

```typescript
import { isAxiosError } from 'axios'
import client, { setAuthToken, clearAuthToken, hasAuthToken } from '@/services/api'

interface LoginResponse {
  data: {
    user: Record<string, unknown>
    tokens: { access_token: string }
  }
}

interface RefreshResponse {
  data: { tokens: { access_token: string } }
}

export async function login(email: string, password: string): Promise<void> {
  const { data } = await client.post<LoginResponse>('/api/v1/auth/login', { email, password })
  setAuthToken(data.data.tokens.access_token)
}

export async function refresh(): Promise<void> {
  const { data } = await client.post<RefreshResponse>('/api/v1/auth/refresh')
  setAuthToken(data.data.tokens.access_token)
}

export async function restoreSession(): Promise<void> {
  try {
    await refresh()
  } catch {
    // cookie 不存在或已過期；維持未登入狀態
  }
}

export async function logout(): Promise<void> {
  await client.post('/api/v1/auth/logout').catch(() => {})
  clearAuthToken()
}

export function isAuthenticated(): boolean {
  return hasAuthToken()
}

export { isAxiosError }
```

> **注意：** `LoginResponse` 保留了後端仍回傳的 `refresh_token` 欄位（body fallback 過渡期），但前端不讀取它。若後端未來移除 body fallback，這個 interface 也可以移除 refresh_token。

- [ ] **Step 4: 執行 auth 測試，確認全部通過**

```bash
cd dashboard && pnpm test -- auth.test
```

Expected：全部 PASS。

- [ ] **Step 5: 執行全部測試確認沒有 regression**

```bash
cd dashboard && pnpm test
```

Expected：所有測試 PASS（包含 channels.test）。

- [ ] **Step 6: Commit**

```bash
git add dashboard/src/services/auth.ts \
        dashboard/src/services/__tests__/auth.test.ts
git commit -m "feat(dashboard): migrate auth.ts to cookie-based refresh, remove localStorage

refs #217

Co-Authored-By: Claude Sonnet 4.6 <claude[bot]@anthropic.com>"
```

---

## Task 3：Bootstrap session restore — 更新 `main.tsx`

頁面重整後 `accessToken` 在記憶體中消失，需要在 React mount 前呼叫 `restoreSession()`，透過 httpOnly cookie 靜默還原 session。若 cookie 已過期，React 正常渲染，`ProtectedRoute` 把用戶導向 `/login`。

**Files:**
- Modify: `dashboard/src/main.tsx`

（`main.tsx` 的 bootstrap 邏輯為 side-effectful、依賴 DOM，不易單元測試；由 E2E 或手動驗證涵蓋。）

- [ ] **Step 1: 更新 `dashboard/src/main.tsx`**

```typescript
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { restoreSession } from '@/services/auth'

async function bootstrap() {
  await restoreSession()
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <App />
    </StrictMode>,
  )
}

bootstrap()
```

- [ ] **Step 2: 執行所有測試，確認無 regression**

```bash
cd dashboard && pnpm test
```

Expected：全部 PASS。

- [ ] **Step 3: Commit**

```bash
git add dashboard/src/main.tsx
git commit -m "feat(dashboard): restore session from cookie before React mount

refs #217

Co-Authored-By: Claude Sonnet 4.6 <claude[bot]@anthropic.com>"
```

---

## Task 4：更新 `docs/auth-architecture.md`

**Files:**
- Modify: `docs/auth-architecture.md`

- [ ] **Step 1: 讀取現有文件**

```bash
cat docs/auth-architecture.md
```

- [ ] **Step 2: 在文件適當位置加入以下內容**

在文件頂部或「Migration」段落加入：

```markdown
## Refresh Token Migration Status

| Phase | 說明 | 狀態 |
|---|---|---|
| Phase 1 — Backend contract | login/refresh/logout 設定 httpOnly cookie；refresh/logout 優先讀 cookie，保留 body fallback | ✅ 完成（PR #220） |
| Phase 2 — Dashboard frontend | 移除 localStorage、cookie-based refresh/logout、401 dedupe interceptor、session restore | ✅ 完成（本 PR） |

## Refresh Token Security Baseline

- **儲存方式：** httpOnly cookie（JS 無法讀取，消除 XSS 竊取路徑）
- **Cookie 屬性：** `HttpOnly=true; SameSite=Lax; Path=/api/v1/auth; Secure=<依 APP_ENV>`
- **CSRF 防護：** SameSite=Lax 保護所有 POST auth 端點（不需要額外 CSRF token）
- **Token Rotation：** 每次 refresh 廢棄舊 token，發新 token（後端 `auth_service.go`）
- **401 Dedupe：** Dashboard axios client 以 promise deduplication 確保並發 401 只觸發一次 refresh

## 已知未完成事項（Body Fallback 退場）

後端目前保留 body 傳送 refresh token 的 fallback（PR #220）。退場時間點：
- [ ] 確認所有 dashboard client 已升級後，移除後端 body fallback
- [ ] 確認 extension / mobile 等其他 client 不依賴 body-based refresh
```

- [ ] **Step 3: 執行測試，確認無 regression**

```bash
cd dashboard && pnpm test
```

- [ ] **Step 4: Commit**

```bash
git add docs/auth-architecture.md
git commit -m "docs: update auth-architecture.md with Phase 2 migration status and security baseline

refs #217

Co-Authored-By: Claude Sonnet 4.6 <claude[bot]@anthropic.com>"
```

---

## PR 開法

### 單一 PR（推薦）

四個 commit 合為一個 PR，對應 issue #217 的三個剩餘 checklist：

```
[frontend] dashboard auth cookie migration — Phase 2

closes #217

## 什麼改動
- `api.ts`：加 `withCredentials: true`、`hasAuthToken()`、401 refresh interceptor（promise dedupe）
- `auth.ts`：移除 localStorage、改用 `hasAuthToken()`、加 `refresh()` / `restoreSession()`
- `main.tsx`：mount 前呼叫 `restoreSession()` 還原 session
- 新增 auth.test.ts + api.interceptor.test.ts
- 更新 docs/auth-architecture.md

## 為什麼
PR #220（backend）已建立 cookie contract。本 PR 完成 Phase 2 前端遷移：
消除 localStorage refresh token 的 XSS 攻擊面，補上 401 dedupe（rotation 必要），
並在頁面重整時透過 httpOnly cookie 靜默還原 session。
```

---

## 驗收清單

- [ ] `localStorage.setItem('refresh_token', ...)` 已從 codebase 移除
- [ ] `localStorage.getItem('refresh_token')` 已從 codebase 移除
- [ ] `client` 帶 `withCredentials: true`
- [ ] 401 interceptor：單次 401 → 自動 refresh + retry
- [ ] 401 interceptor：並發 401 → 只呼叫一次 `/auth/refresh`（dedupe）
- [ ] `/auth/refresh` 本身 401 → 不觸發 retry loop
- [ ] `isAuthenticated()` 在 interceptor 更新 token 後仍回傳正確值
- [ ] `restoreSession()` 頁面重整後能透過 cookie 還原 session
- [ ] `restoreSession()` 靜默失敗（cookie 過期 → 導向 login）
- [ ] 所有 vitest 測試通過
- [ ] `docs/auth-architecture.md` 標記 Phase 2 完成並說明 body fallback 退場計劃

---

## 已知邊界

1. **Body fallback 退場**：後端保留 body fallback 期間，攻擊面部分仍存在（舊前端版本可能仍送 body）。退場時機等所有 client 確認升級後，另開 issue 移除。
2. **extension auth contract**：本計劃不修改 extension 的 auth 邏輯（issue #217 明確排除）。
3. **Twitch / Google OAuth cookie**：OAuth callback 的 cookie 已由後端在 PR #220 補上，前端只需處理 password-based login/logout/refresh。
4. **SameSite=None 生產環境**：若 dashboard 與 backend 跨域，後端已有邏輯在 production 設 `SameSite=None; Secure`。確保 `ALLOWED_ORIGINS` env var 在部署時正確設定（非 wildcard）。
