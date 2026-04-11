import { isAxiosError } from 'axios'
import type { AxiosRequestConfig } from 'axios'
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

interface RefreshResponse {
  data: {
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

export function getAccessToken(): string | null {
  return accessToken
}

export function getCurrentUserRole(): string | null {
  return currentUserSession?.role ?? readStoredCurrentUserSession()?.role ?? null
}

export function getCurrentUserId(): string | null {
  return currentUserSession?.id ?? readStoredCurrentUserSession()?.id ?? null
}

export async function restoreSession(): Promise<void> {
  const refreshToken = localStorage.getItem('refresh_token')
  if (!refreshToken) throw new Error('no refresh token')

  try {
    const { data } = await client.post<RefreshResponse>('/api/v1/auth/refresh', {
      refresh_token: refreshToken,
    })
    accessToken = data.data.tokens.access_token
    localStorage.setItem('refresh_token', data.data.tokens.refresh_token)
    setAuthToken(accessToken)
  } catch (error) {
    if (isAxiosError(error) && error.response?.status === 401) {
      accessToken = null
      currentUserSession = null
      localStorage.removeItem('refresh_token')
      localStorage.removeItem('current_user')
      clearAuthToken()
    }
    throw error
  }
}

let isRefreshing = false
const pendingRequests: Array<{
  resolve: (value: unknown) => void
  reject: (reason: unknown) => void
}> = []

client.interceptors.response.use(
  undefined,
  async (error: unknown) => {
    if (!isAxiosError(error)) return Promise.reject(error)

    const config = error.config
    if (!config) return Promise.reject(error)

    type RetryConfig = AxiosRequestConfig & { _retry?: boolean }
    const retryConfig = config as RetryConfig
    const isRefreshEndpoint = config.url?.includes('/auth/refresh')

    if (error.response?.status !== 401 || retryConfig._retry || isRefreshEndpoint) {
      return Promise.reject(error)
    }

    if (isRefreshing) {
      return new Promise<unknown>((resolve, reject) => {
        pendingRequests.push({ resolve, reject })
      }).then(() => {
        const newToken = getAccessToken()
        const updatedConfig = { ...retryConfig, _retry: true } as RetryConfig
        if (newToken) {
          updatedConfig.headers = {
            ...(retryConfig.headers as Record<string, string>),
            Authorization: `Bearer ${newToken}`,
          }
        }
        return client(updatedConfig as AxiosRequestConfig)
      })
    }

    retryConfig._retry = true
    isRefreshing = true

    try {
      await restoreSession()
      const queued = pendingRequests.splice(0)
      queued.forEach(({ resolve }) => resolve(undefined))

      const newToken = getAccessToken()
      if (newToken) {
        retryConfig.headers = {
          ...(retryConfig.headers as Record<string, string>),
          Authorization: `Bearer ${newToken}`,
        }
      }

      return client(retryConfig as AxiosRequestConfig)
    } catch (refreshError) {
      const isDefiniteAuthFailure =
        isAxiosError(refreshError) && refreshError.response?.status === 401
      if (isDefiniteAuthFailure) {
        accessToken = null
        currentUserSession = null
        localStorage.removeItem('refresh_token')
        localStorage.removeItem('current_user')
        clearAuthToken()
      }
      const queued = pendingRequests.splice(0)
      queued.forEach(({ reject }) => reject(refreshError))
      return Promise.reject(isDefiniteAuthFailure ? error : refreshError)
    } finally {
      isRefreshing = false
    }
  },
)

export { isAxiosError }
