import { isAxiosError } from 'axios'
import type { AxiosRequestConfig } from 'axios'
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

interface RefreshResponse {
  data: {
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

export function getAccessToken(): string | null {
  return accessToken
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
    localStorage.removeItem('refresh_token')
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
      }).then(() => client({ ...retryConfig, _retry: true } as RetryConfig))
    }

    retryConfig._retry = true
    isRefreshing = true

    try {
      await restoreSession()
      const queued = pendingRequests.splice(0)
      queued.forEach(({ resolve }) => resolve(undefined))
      return client(retryConfig as AxiosRequestConfig)
    } catch (refreshError) {
      accessToken = null
      clearAuthToken()
      const queued = pendingRequests.splice(0)
      queued.forEach(({ reject }) => reject(refreshError))
      return Promise.reject(error)
    } finally {
      isRefreshing = false
    }
  },
)
