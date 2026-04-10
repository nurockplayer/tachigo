import { isAxiosError } from 'axios'
import client, { clearAuthToken, setAuthToken } from '@/services/api'

let accessToken: string | null = null
let currentUserSession: CurrentUserSession | null = null

interface CurrentUserSession {
  id: string
  role: string
}

interface LoginResponse {
  data: {
    user: Record<string, unknown>
    tokens: {
      access_token: string
      refresh_token: string
    }
  }
}

function toCurrentUserSession(user: Record<string, unknown>): CurrentUserSession | null {
  if (typeof user.id !== 'string' || typeof user.role !== 'string') {
    return null
  }

  return {
    id: user.id,
    role: user.role,
  }
}

function readStoredCurrentUserSession(): CurrentUserSession | null {
  const stored = localStorage.getItem('current_user')
  if (!stored) return null

  try {
    const parsed = JSON.parse(stored) as Record<string, unknown>
    const session = toCurrentUserSession(parsed)
    currentUserSession = session
    return session
  } catch {
    return null
  }
}

export async function login(email: string, password: string): Promise<void> {
  const { data } = await client.post<LoginResponse>('/api/v1/auth/login', {
    email,
    password,
  })

  const session = toCurrentUserSession(data.data.user)

  accessToken = data.data.tokens.access_token
  currentUserSession = session
  localStorage.setItem('refresh_token', data.data.tokens.refresh_token)

  if (session) {
    localStorage.setItem('current_user', JSON.stringify(session))
  } else {
    localStorage.removeItem('current_user')
  }

  setAuthToken(accessToken)
}

export async function logout(): Promise<void> {
  const refreshToken = localStorage.getItem('refresh_token')
  if (refreshToken) {
    await client.post('/api/v1/auth/logout', { refresh_token: refreshToken }).catch(() => {})
  }

  accessToken = null
  currentUserSession = null
  localStorage.removeItem('refresh_token')
  localStorage.removeItem('current_user')
  clearAuthToken()
}

export function isAuthenticated(): boolean {
  return accessToken !== null
}

export function getCurrentUserRole(): string | null {
  return currentUserSession?.role ?? readStoredCurrentUserSession()?.role ?? null
}

export function getCurrentUserId(): string | null {
  return currentUserSession?.id ?? readStoredCurrentUserSession()?.id ?? null
}

export { isAxiosError }
