import type { AuthProvider } from '@refinedev/core'
import { isAxiosError, isAuthenticated, login, logout } from '@/services/auth'

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
  check: async () => {
    if (isAuthenticated()) {
      return {
        authenticated: true,
      }
    }

    return {
      authenticated: false,
      redirectTo: '/login',
    }
  },
  getPermissions: async () => {
    const refreshToken = localStorage.getItem('refresh_token')
    if (!refreshToken) {
      return null
    }

    const payload = decodeJwtPayload(refreshToken)
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
