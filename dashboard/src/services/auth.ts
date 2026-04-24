import { isAxiosError } from 'axios'
import client, { clearAuthToken, hasAuthToken, setAuthToken } from '@/services/api'

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
