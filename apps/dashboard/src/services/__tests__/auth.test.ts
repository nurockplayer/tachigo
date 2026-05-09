import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import MockAdapter from 'axios-mock-adapter'
import client, { clearAuthToken, hasAuthToken } from '@/services/api'
import { getUserRole, isAuthenticated, login, logout, refresh, restoreSession } from '@/services/auth'

let mock: InstanceType<typeof MockAdapter>

beforeEach(() => {
  mock = new MockAdapter(client)
  clearAuthToken()
  localStorage.clear()
})

afterEach(() => {
  mock.restore()
})

describe('login()', () => {
  it('成功時設定 Authorization header，isAuthenticated() 為 true', async () => {
    mock.onPost('/api/v1/auth/login').replyOnce(200, {
      data: { user: { id: 'u1' }, tokens: { access_token: 'access-abc' } },
    })

    await login('user@example.com', 'password')

    expect(hasAuthToken()).toBe(true)
    expect(isAuthenticated()).toBe(true)
  })

  it('不將 refresh_token 寫入 localStorage', async () => {
    mock.onPost('/api/v1/auth/login').replyOnce(200, {
      data: { user: { id: 'u1' }, tokens: { access_token: 'access-abc' } },
    })

    await login('user@example.com', 'password')

    expect(localStorage.getItem('refresh_token')).toBeNull()
  })

  it('API 失敗時 throw，isAuthenticated() 仍為 false', async () => {
    mock.onPost('/api/v1/auth/login').replyOnce(401)

    await expect(login('user@example.com', 'wrong')).rejects.toBeDefined()
    expect(isAuthenticated()).toBe(false)
  })
})

describe('logout()', () => {
  beforeEach(async () => {
    mock.onPost('/api/v1/auth/login').replyOnce(200, {
      data: { user: {}, tokens: { access_token: 'access-abc' } },
    })
    await login('u@e.com', 'pw')
  })

  it('清除 Authorization header，isAuthenticated() 為 false', async () => {
    mock.onPost('/api/v1/auth/logout').replyOnce(200)

    await logout()

    expect(hasAuthToken()).toBe(false)
    expect(isAuthenticated()).toBe(false)
  })

  it('不送 refresh_token body（由 cookie 處理）', async () => {
    mock.onPost('/api/v1/auth/logout').replyOnce(200)

    await logout()

    const call = mock.history.post.find(r => r.url === '/api/v1/auth/logout')
    expect(call).toBeDefined()
    expect(call?.data ?? null).toBeFalsy()
  })

  it('API 失敗時 reject，不清除本機狀態（cookie 可能未清，不應假裝登出）', async () => {
    mock.onPost('/api/v1/auth/logout').replyOnce(500)

    await expect(logout()).rejects.toBeDefined()
    expect(isAuthenticated()).toBe(true)
  })

  it('不讀取也不清除 localStorage', async () => {
    localStorage.setItem('refresh_token', 'old-token')
    mock.onPost('/api/v1/auth/logout').replyOnce(200)

    await logout()

    expect(localStorage.getItem('refresh_token')).toBe('old-token')
  })
})

describe('refresh()', () => {
  it('成功時更新 Authorization header，isAuthenticated() 為 true', async () => {
    mock.onPost('/api/v1/auth/refresh').replyOnce(200, {
      data: { tokens: { access_token: 'new-token' } },
    })

    await refresh()

    expect(hasAuthToken()).toBe(true)
    expect(isAuthenticated()).toBe(true)
  })

  it('失敗時 throw，isAuthenticated() 仍為 false', async () => {
    mock.onPost('/api/v1/auth/refresh').replyOnce(401)

    await expect(refresh()).rejects.toBeDefined()
    expect(isAuthenticated()).toBe(false)
  })
})

describe('restoreSession()', () => {
  it('cookie 有效時 isAuthenticated() 為 true', async () => {
    mock.onPost('/api/v1/auth/refresh').replyOnce(200, {
      data: { tokens: { access_token: 'restored-token' } },
    })

    await restoreSession()

    expect(isAuthenticated()).toBe(true)
  })

  it('cookie 過期或不存在時靜默 resolve，isAuthenticated() 仍為 false', async () => {
    mock.onPost('/api/v1/auth/refresh').replyOnce(401)

    await expect(restoreSession()).resolves.toBeUndefined()
    expect(isAuthenticated()).toBe(false)
  })
})

describe('getUserRole()', () => {
  it('從 access token payload 解析 role', async () => {
    const payload = btoa(JSON.stringify({ role: 'agency' }))
      .replace(/\+/g, '-')
      .replace(/\//g, '_')
      .replace(/=+$/g, '')

    mock.onPost('/api/v1/auth/login').replyOnce(200, {
      data: {
        user: { id: 'u1' },
        tokens: { access_token: `header.${payload}.signature` },
      },
    })

    await login('user@example.com', 'password')

    expect(getUserRole()).toBe('agency')
  })
})
