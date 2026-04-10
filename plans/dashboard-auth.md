# 任務：Dashboard 登入與 JWT 管理

## 專案背景

這是 tachigo 專案的 Dashboard 前端，位於 `dashboard/` 資料夾。
技術棧：React 19 + TypeScript + Vite + pnpm + Tailwind CSS v4 + React Router v7。

issue #33（Dashboard 骨架）已完成，本次任務是 issue #34：實作登入功能與 JWT 管理。

---

## 開始前準備

1. 從 `develop` 拉新 branch：
   ```bash
   cd C:\Users\higgus\tachigo
   git checkout develop
   git pull
   git checkout -b feat/dashboard-auth
   ```

2. 確認 `dashboard/.env` 存在（複製範本）：
   ```bash
   cp dashboard/.env.example dashboard/.env
   ```
   確認內容包含：
   ```
   VITE_API_URL=http://localhost:8080
   ```

---

## 後端 API 規格

**POST /api/v1/auth/login**

Request body：
```json
{ "email": "string", "password": "string" }
```

Response（200）：
```json
{
  "data": {
    "user": { ... },
    "tokens": {
      "access_token": "string",
      "refresh_token": "string"
    }
  }
}
```

Error（401）：帳密錯誤

**POST /api/v1/auth/logout**

Request body：
```json
{ "refresh_token": "string" }
```

Response（200）：成功，無 body

---

## Token 儲存策略

- `access_token` → module-level 變數（記憶體），頁面重整後消失
- `refresh_token` → `localStorage`（key: `refresh_token`）

---

## 要修改 / 建立的檔案

### 1. 新增 `dashboard/src/services/auth.ts`

```ts
import { isAxiosError } from 'axios'
import client, { setAuthToken, clearAuthToken } from '@/services/api'

// 記憶體儲存，不 export
let accessToken: string | null = null

interface LoginResponse {
  data: {
    user: Record<string, unknown>
    tokens: {
      access_token: string
      refresh_token: string
    }
  }
}

export async function login(email: string, password: string): Promise<void> {
  const { data } = await client.post<LoginResponse>('/api/v1/auth/login', {
    email,
    password,
  })
  accessToken = data.data.tokens.access_token
  localStorage.setItem('refresh_token', data.data.tokens.refresh_token)
  setAuthToken(accessToken)
}

export async function logout(): Promise<void> {
  const refreshToken = localStorage.getItem('refresh_token')
  if (refreshToken) {
    // 通知後端使 refresh_token 失效，失敗不阻斷登出流程
    await client.post('/api/v1/auth/logout', { refresh_token: refreshToken }).catch(() => {})
  }
  accessToken = null
  localStorage.removeItem('refresh_token')
  clearAuthToken()
}

export function isAuthenticated(): boolean {
  return accessToken !== null
}

export { isAxiosError }
```

---

### 2. 修改 `dashboard/src/pages/LoginPage.tsx`

```tsx
import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router'
import { isAxiosError } from 'axios'
import { login } from '@/services/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export default function LoginPage() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setIsLoading(true)
    setError('')
    try {
      await login(email, password)
      navigate('/')
    } catch (err) {
      if (isAxiosError(err) && err.response?.status === 401) {
        setError('帳號或密碼錯誤')
      } else {
        setError('連線失敗，請稍後再試')
      }
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="w-full max-w-sm rounded-lg border border-border bg-background p-8 shadow-sm">
        <h1 className="mb-6 text-2xl font-bold text-foreground">登入</h1>
        {error && (
          <p className="mb-4 text-sm text-destructive">{error}</p>
        )}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1">
            <Label htmlFor="email">Email</Label>
            <Input
              id="email"
              type="email"
              placeholder="admin@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="password">密碼</Label>
            <Input
              id="password"
              type="password"
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>
          <Button type="submit" className="w-full" disabled={isLoading}>
            {isLoading ? '登入中...' : '登入'}
          </Button>
        </form>
      </div>
    </div>
  )
}
```

---

### 3. 修改 `dashboard/src/components/ProtectedRoute.tsx`

```tsx
import { Navigate, Outlet } from 'react-router'
import { isAuthenticated } from '@/services/auth'

export default function ProtectedRoute() {
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }

  return <Outlet />
}
```

---

### 4. 修改 `dashboard/src/components/Layout.tsx`

在 Header 加入登出按鈕，並加入 `isLoggingOut` loading 狀態：

- import `logout` from `@/services/auth`
- import `useNavigate` from `react-router`
- import `useState` from `react`
- 新增 `isLoggingOut` state
- 新增 `handleLogout`：設定 `isLoggingOut(true)`，await `logout()`，navigate 到 `/login`
- Header 改為 `flex items-center justify-between`
- 右側加入登出按鈕，`disabled={isLoggingOut}`：

```tsx
const [isLoggingOut, setIsLoggingOut] = useState(false)

async function handleLogout() {
  setIsLoggingOut(true)
  await logout()
  navigate('/login')
}

// Header 右側
<Button variant="ghost" size="sm" onClick={handleLogout} disabled={isLoggingOut}>
  {isLoggingOut ? '登出中...' : '登出'}
</Button>
```

### 5. 新增 `dashboard/src/vite-env.d.ts`

補上自訂環境變數的型別定義，避免 `import.meta.env.VITE_API_URL` 被推斷為 `string | undefined`：

```ts
/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
```

---

## 完成條件（請逐一驗證）

- [ ] 用正確帳密登入，成功後導向 `/`（總覽頁）
- [ ] 用錯誤帳密登入，顯示「帳號或密碼錯誤」紅色訊息
- [ ] 網路無法連線時，顯示「連線失敗，請稍後再試」
- [ ] 登入按鈕在送出期間顯示「登入中...」並無法點擊
- [ ] 未登入直接訪問 `/`、`/streamers` 等，自動導向 `/login`
- [ ] 登入後重整頁面，因 access_token 在記憶體中消失，導回登入頁（符合規格）
- [ ] 點 Header 的登出按鈕，清除 token，導回登入頁
- [ ] 登出按鈕在 await 期間顯示「登出中...」並無法點擊
- [ ] 登入後開啟 DevTools → Application → localStorage，確認 key 為 `refresh_token`，value 為 JWT 字串（非 `[object Object]`）
- [ ] `pnpm lint` 通過
- [ ] `pnpm build` 通過

---

## Commit 與 PR

```
feat: implement dashboard login and JWT management refs #34

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
```

PR：`feat/dashboard-auth` → `develop`，PR body 加上 `Closes #34`

---

## 注意事項

1. 全程使用 pnpm，不使用 npm
2. 路徑 alias `@/` 對應 `src/`，使用 `@/` 不要用相對路徑
3. `isAuthenticated()` 只檢查記憶體中的 access_token
4. 不需要實作 token refresh（那是另一個 issue 的工作）
5. `logout()` 是 async，呼叫時記得 `await`
6. `isAxiosError` 用 named import：`import { isAxiosError } from 'axios'`，不要用 `axios.isAxiosError`
