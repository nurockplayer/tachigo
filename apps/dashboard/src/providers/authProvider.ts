import type { AuthProvider } from '@refinedev/core'
import { getAuthToken } from '@/services/api'
import {
  getUserRole,
  isAuthenticated,
  login,
  logout,
  restoreSession,
} from '@/services/auth'

type JwtPayload = {
  sub?: string
  user_id?: string
  email?: string
  name?: string
  role?: string
}

function decodeAccessToken(): JwtPayload | null {
  const accessToken = getAuthToken()
  if (!accessToken) return null

  try {
    const b64 = accessToken.split('.')[1]
    if (!b64) return null
    const normalized = b64.replace(/-/g, '+').replace(/_/g, '/')
    const padded = normalized + '='.repeat((4 - (normalized.length % 4)) % 4)
    return JSON.parse(atob(padded)) as JwtPayload
  } catch {
    return null
  }
}

export const authProvider: AuthProvider = {
  login: async ({ email, password }: { email: string; password: string }) => {
    await login(String(email), String(password))

    return {
      success: true,
      redirectTo: '/',
    }
  },

  logout: async () => {
    await logout()

    return {
      success: true,
      redirectTo: '/login',
    }
  },

  check: async () => {
    await restoreSession()

    if (isAuthenticated()) {
      return { authenticated: true }
    }

    return {
      authenticated: false,
      redirectTo: '/login',
    }
  },

  getPermissions: async () => getUserRole(),

  getIdentity: async () => {
    const payload = decodeAccessToken()
    if (!payload) return null

    return {
      id: payload.user_id ?? payload.sub ?? '',
      name: payload.name ?? payload.email ?? payload.sub ?? 'Dashboard user',
      email: payload.email,
      role: payload.role,
    }
  },

  onError: async (error: unknown) => {
    const status = (error as { status?: number; response?: { status?: number } }).status
      ?? (error as { response?: { status?: number } }).response?.status

    if (status === 401 || status === 403) {
      return { logout: true }
    }

    return { error: error as Error }
  },
}
