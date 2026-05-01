import { describe, expect, it, vi } from 'vitest'
import { authProvider } from '@/providers/authProvider'

vi.mock('@/services/api', () => ({
  getAuthToken: vi.fn(),
}))

vi.mock('@/services/auth', () => ({
  getUserRole: vi.fn(),
  isAuthenticated: vi.fn(),
  login: vi.fn(),
  logout: vi.fn(),
  restoreSession: vi.fn(),
}))

describe('authProvider.onError', () => {
  it('401 時要求 Refine 登出', async () => {
    await expect(authProvider.onError?.({ response: { status: 401 } })).resolves.toEqual({
      logout: true,
    })
  })

  it('403 時保留登入狀態並回傳錯誤給 UI 處理', async () => {
    const error = { response: { status: 403 } }

    await expect(authProvider.onError?.(error)).resolves.toEqual({
      error,
    })
  })
})
