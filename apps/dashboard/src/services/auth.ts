import { isAxiosError } from 'axios'
import client, { clearAuthToken, getAuthToken, hasAuthToken, setAuthToken } from '@/services/api'

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
  await client.post('/api/v1/auth/logout')
  clearAuthToken()
}

export function isAuthenticated(): boolean {
  return hasAuthToken()
}

export function getUserRole(): string | null {
  const accessToken = getAuthToken()
  if (!accessToken) return null
  try {
    const b64 = accessToken.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')
    const padded = b64 + '='.repeat((4 - (b64.length % 4)) % 4)
    const payload = JSON.parse(atob(padded)) as Record<string, unknown>
    return typeof payload.role === 'string' ? payload.role : null
  } catch {
    return null
  }
}

export { isAxiosError }
