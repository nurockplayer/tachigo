import type { AuthProvider } from '@refinedev/core'
import { isAxiosError } from 'axios'
import { getAccessToken, isAuthenticated, login, logout, restoreSession } from '@/services/auth'

type LoginVariables = {
  email?: string
  password?: string
}

function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const [, payload] = token.split('.')
    if (!payload) {
      return null
    }

    const normalized = payload.replace(/-/g, '+').replace(/_/g, '/')
    const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
    const json = atob(padded)

    return JSON.parse(json) as Record<string, unknown>
  } catch {
    return null
  }
}

export const authProvider: AuthProvider = {
  login: async ({ email, password }: LoginVariables = {}) => {
    try {
      await login(email ?? '', password ?? '')

      return {
        success: true,
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error : new Error('Login failed'),
      }
    }
  },
  logout: async () => {
    await logout()

    return {
      success: true,
    }
  },
  // Blocker 3 fix: 頁面重整後嘗試用 refresh_token 重建 session，而非直接跳 /login
  check: async () => {
    if (isAuthenticated()) {
      return { authenticated: true }
    }

    const restored = await restoreSession()
    if (restored) {
      return { authenticated: true }
    }

    return {
      authenticated: false,
      redirectTo: '/login',
    }
  },
  // Blocker 1 fix: 解 access_token（帶有 role claims），而非 opaque refresh_token
  getPermissions: async () => {
    const token = getAccessToken()
    if (!token) return null

    const payload = decodeJwtPayload(token)
    return payload?.role ?? null
  },
  onError: async (error) => {
    if (isAxiosError(error) && error.response?.status === 401) {
      return {
        logout: true,
      }
    }

    return {
      error: error instanceof Error ? error : new Error('Request failed'),
    }
  },
}
