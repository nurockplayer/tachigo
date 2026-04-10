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
