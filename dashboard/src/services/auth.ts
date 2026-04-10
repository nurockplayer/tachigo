import { isAxiosError } from 'axios'
import client, { setAuthToken, clearAuthToken } from '@/services/api'

// 記憶體儲存，不 export
let accessToken: string | null = null
let currentUser: Record<string, unknown> | null = null

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
  currentUser = data.data.user
  localStorage.setItem('refresh_token', data.data.tokens.refresh_token)
  localStorage.setItem('current_user', JSON.stringify(data.data.user))
  setAuthToken(accessToken)
}

export async function logout(): Promise<void> {
  const refreshToken = localStorage.getItem('refresh_token')
  if (refreshToken) {
    await client.post('/api/v1/auth/logout', { refresh_token: refreshToken }).catch(() => {})
  }
  accessToken = null
  currentUser = null
  localStorage.removeItem('refresh_token')
  localStorage.removeItem('current_user')
  clearAuthToken()
}

export function isAuthenticated(): boolean {
  return accessToken !== null
}

export function getCurrentUserRole(): string | null {
  if (currentUser && typeof currentUser.role === 'string') {
    return currentUser.role
  }
  const stored = localStorage.getItem('current_user')
  if (!stored) return null
  try {
    const parsed = JSON.parse(stored) as Record<string, unknown>
    currentUser = parsed
    return typeof parsed.role === 'string' ? parsed.role : null
  } catch {
    return null
  }
}

export { isAxiosError }
