import { beforeEach, describe, expect, it, vi } from 'vitest'

const postMock = vi.fn()
const setAuthTokenMock = vi.fn()
const clearAuthTokenMock = vi.fn()

vi.mock('@/services/api', () => ({
  default: {
    post: (...args: unknown[]) => postMock(...args),
  },
  setAuthToken: (...args: unknown[]) => setAuthTokenMock(...args),
  clearAuthToken: (...args: unknown[]) => clearAuthTokenMock(...args),
}))

describe('auth service session storage', () => {
  beforeEach(() => {
    vi.resetModules()
    postMock.mockReset()
    setAuthTokenMock.mockReset()
    clearAuthTokenMock.mockReset()
    localStorage.clear()
  })

  it('login only persists the minimum session fields', async () => {
    postMock.mockResolvedValue({
      data: {
        data: {
          user: {
            id: 'user-123',
            role: 'streamer',
            email: 'streamer@example.com',
            username: 'alice',
            avatar_url: 'https://example.com/avatar.png',
          },
          tokens: {
            access_token: 'access-token',
            refresh_token: 'refresh-token',
          },
        },
      },
    })

    const { login } = await import('@/services/auth')

    await login('streamer@example.com', 'password123')

    expect(JSON.parse(localStorage.getItem('current_user') ?? 'null')).toEqual({
      id: 'user-123',
      role: 'streamer',
    })
    expect(localStorage.getItem('refresh_token')).toBeNull()
    expect(setAuthTokenMock).toHaveBeenCalledWith('access-token')
  })

  it('logout clears session without requiring a persisted refresh token', async () => {
    postMock.mockResolvedValue({})
    localStorage.setItem('current_user', JSON.stringify({ id: 'user-123', role: 'streamer' }))

    const { logout } = await import('@/services/auth')

    await logout()

    expect(postMock).toHaveBeenCalledWith('/api/v1/auth/logout')
    expect(localStorage.getItem('current_user')).toBeNull()
    expect(localStorage.getItem('refresh_token')).toBeNull()
    expect(clearAuthTokenMock).toHaveBeenCalledTimes(1)
  })
})
