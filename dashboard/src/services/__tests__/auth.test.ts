import { beforeEach, describe, expect, it } from 'vitest'
import AxiosMockAdapter from 'axios-mock-adapter'
import client from '@/services/api'
import {
  getAccessToken,
  getCurrentUserId,
  getCurrentUserRole,
  isAuthenticated,
  login,
  logout,
  restoreSession,
} from '@/services/auth'

const mock = new AxiosMockAdapter(client)

const TOKENS = {
  access_token: 'new-access',
  refresh_token: 'new-refresh',
}

beforeEach(async () => {
  mock.reset()
  localStorage.clear()
  await logout()
})

describe('login session storage', () => {
  it('login persists only the minimum user session fields plus refresh token', async () => {
    mock.onPost('/api/v1/auth/login').reply(200, {
      data: {
        user: {
          id: 'user-123',
          role: 'streamer',
          email: 'streamer@example.com',
          username: 'alice',
        },
        tokens: {
          access_token: 'access-token',
          refresh_token: 'refresh-token',
        },
      },
    })

    await login('streamer@example.com', 'password123')

    expect(JSON.parse(localStorage.getItem('current_user') ?? 'null')).toEqual({
      id: 'user-123',
      role: 'streamer',
    })
    expect(localStorage.getItem('refresh_token')).toBe('refresh-token')
    expect(getCurrentUserRole()).toBe('streamer')
    expect(getCurrentUserId()).toBe('user-123')
  })
})

describe('restoreSession', () => {
  it('refresh 成功時更新 access token 並更新 refresh_token', async () => {
    localStorage.setItem('refresh_token', 'old-refresh')
    mock.onPost('/api/v1/auth/refresh').reply(200, {
      data: { tokens: TOKENS },
    })

    await restoreSession()

    expect(getAccessToken()).toBe('new-access')
    expect(localStorage.getItem('refresh_token')).toBe('new-refresh')
  })

  it('refresh 回 5xx 時不清除 refresh_token', async () => {
    localStorage.setItem('refresh_token', 'valid-refresh')
    mock.onPost('/api/v1/auth/refresh').reply(500)

    await expect(restoreSession()).rejects.toThrow()
    expect(localStorage.getItem('refresh_token')).toBe('valid-refresh')
  })

  it('refresh 回 401 時清除 refresh_token', async () => {
    localStorage.setItem('refresh_token', 'bad-refresh')
    localStorage.setItem('current_user', JSON.stringify({ id: 'user-123', role: 'streamer' }))
    mock.onPost('/api/v1/auth/refresh').reply(401)

    await expect(restoreSession()).rejects.toThrow()
    expect(localStorage.getItem('refresh_token')).toBeNull()
    expect(localStorage.getItem('current_user')).toBeNull()
    expect(getCurrentUserRole()).toBeNull()
    expect(getCurrentUserId()).toBeNull()
  })

  it('沒有 refresh_token 時直接 throw', async () => {
    await expect(restoreSession()).rejects.toThrow('no refresh token')
  })
})

describe('401 interceptor', () => {
  it('API 回 401 時自動 refresh 並重試原始 request', async () => {
    localStorage.setItem('refresh_token', 'valid-refresh')

    mock
      .onGet('/api/v1/streamers')
      .replyOnce(401)
      .onGet('/api/v1/streamers')
      .replyOnce(200, { data: [] })

    mock.onPost('/api/v1/auth/refresh').reply(200, {
      data: { tokens: TOKENS },
    })

    const res = await client.get('/api/v1/streamers')
    expect(res.status).toBe(200)
    expect(getAccessToken()).toBe('new-access')
  })

  it('refresh 成功後 retry 的 Authorization header 是新 token', async () => {
    localStorage.setItem('refresh_token', 'valid-refresh')

    mock
      .onGet('/api/v1/streamers')
      .replyOnce(401)
      .onGet('/api/v1/streamers')
      .replyOnce(200, { data: [] })

    mock.onPost('/api/v1/auth/refresh').reply(200, {
      data: { tokens: TOKENS },
    })

    const res = await client.get('/api/v1/streamers')
    expect(res.status).toBe(200)

    const retryRequest = mock.history.get[1]
    expect(retryRequest.headers?.Authorization).toBe('Bearer new-access')
  })

  it('refresh 失敗時不重試，向上拋出原始 401 error', async () => {
    localStorage.setItem('refresh_token', 'bad-refresh')
    mock.onGet('/api/v1/streamers').reply(401)
    mock.onPost('/api/v1/auth/refresh').reply(401)

    await expect(client.get('/api/v1/streamers')).rejects.toMatchObject({
      response: { status: 401 },
    })
    expect(localStorage.getItem('refresh_token')).toBeNull()
  })

  it('refresh endpoint 本身 401 不觸發 interceptor retry', async () => {
    localStorage.setItem('refresh_token', 'bad-refresh')
    mock.onPost('/api/v1/auth/refresh').reply(401)

    await expect(restoreSession()).rejects.toThrow()
    expect(mock.history.post.filter((r) => r.url?.includes('/auth/refresh')).length).toBe(1)
  })

  it('queued request retry 時帶新 token（不帶舊的過期 token）', async () => {
    localStorage.setItem('refresh_token', 'valid-refresh')

    mock
      .onGet('/api/v1/streamers')
      .replyOnce(401)
      .onGet('/api/v1/streamers')
      .replyOnce(200, { data: [] })
    mock
      .onGet('/api/v1/channels')
      .replyOnce(401)
      .onGet('/api/v1/channels')
      .replyOnce(200, { data: [] })
    mock.onPost('/api/v1/auth/refresh').replyOnce(200, {
      data: { tokens: { access_token: 'refreshed-token', refresh_token: 'new-refresh' } },
    })

    await Promise.all([client.get('/api/v1/streamers'), client.get('/api/v1/channels')])

    const channelRetry = mock.history.get.find(
      (r) => r.url?.includes('/api/v1/channels') && r.headers?.Authorization,
    )
    expect(channelRetry?.headers?.Authorization).toBe('Bearer refreshed-token')
  })
})

describe('auth basics', () => {
  it('logout clears current_user and refresh_token and notifies backend when token exists', async () => {
    localStorage.setItem('refresh_token', 'stored-refresh')
    localStorage.setItem('current_user', JSON.stringify({ id: 'user-123', role: 'streamer' }))
    mock.onPost('/api/v1/auth/logout').reply(200, {})

    await logout()

    expect(mock.history.post[0]?.data).toBe(JSON.stringify({ refresh_token: 'stored-refresh' }))
    expect(localStorage.getItem('current_user')).toBeNull()
    expect(localStorage.getItem('refresh_token')).toBeNull()
    expect(isAuthenticated()).toBe(false)
    expect(getAccessToken()).toBeNull()
  })

  it('login 後 isAuthenticated 為 true，logout 後回到 false', async () => {
    mock.onPost('/api/v1/auth/login').reply(200, {
      data: {
        user: {
          id: 'admin-1',
          role: 'admin',
        },
        tokens: {
          access_token: 'login-access',
          refresh_token: 'login-refresh',
        },
      },
    })

    await login('admin@example.com', 'password')
    expect(isAuthenticated()).toBe(true)

    mock.onPost('/api/v1/auth/logout').reply(200, {})
    await logout()
    expect(isAuthenticated()).toBe(false)
    expect(getAccessToken()).toBeNull()
  })
})
