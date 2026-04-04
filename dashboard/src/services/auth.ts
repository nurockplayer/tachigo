import client, { setAuthToken, clearAuthToken } from '@/services/api'

// 記憶體儲存（快取），不 export
let accessToken: string | null = null

const ACCESS_TOKEN_KEY = 'access_token'
const REFRESH_TOKEN_KEY = 'refresh_token'

interface LoginResponse {
  data: {
    user: Record<string, unknown>
    tokens: {
      access_token: string
      refresh_token: string
    }
  }
}

interface RefreshResponse {
  data: {
    tokens: {
      access_token: string
      refresh_token: string
    }
  }
}

function persistTokens(access: string, refresh: string) {
  accessToken = access
  localStorage.setItem(ACCESS_TOKEN_KEY, access)
  localStorage.setItem(REFRESH_TOKEN_KEY, refresh)
  setAuthToken(access)
}

function clearTokens() {
  accessToken = null
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  clearAuthToken()
}

export async function login(email: string, password: string): Promise<void> {
  const { data } = await client.post<LoginResponse>('/api/v1/auth/login', {
    email,
    password,
  })
  persistTokens(data.data.tokens.access_token, data.data.tokens.refresh_token)
}

export async function logout(): Promise<void> {
  const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY)
  if (refreshToken) {
    await client.post('/api/v1/auth/logout', { refresh_token: refreshToken }).catch(() => {})
  }
  clearTokens()
}

/**
 * 嘗試用 localStorage 的 refresh_token 重建 session。
 * 頁面重整後由 authProvider.check() 呼叫。
 * 成功回傳 true，無 token 或 token 已失效回傳 false。
 */
export async function restoreSession(): Promise<boolean> {
  // 優先用記憶體快取（同 tab 內的正常使用路徑）
  if (accessToken) return true

  const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY)
  if (!refreshToken) return false

  try {
    const { data } = await client.post<RefreshResponse>('/api/v1/auth/refresh', {
      refresh_token: refreshToken,
    })
    persistTokens(data.data.tokens.access_token, data.data.tokens.refresh_token)
    return true
  } catch {
    clearTokens()
    return false
  }
}

export function isAuthenticated(): boolean {
  return accessToken !== null
}

/** 給 authProvider.getPermissions() 讀取，解 JWT claims 用 */
export function getAccessToken(): string | null {
  return accessToken
}

