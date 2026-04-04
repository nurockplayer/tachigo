import { beforeEach, describe, expect, it } from 'vitest'
import AxiosMockAdapter from 'axios-mock-adapter'
import client from '@/services/api'
import {
  getAccessToken,
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

describe('restoreSession', () => {
  it('refresh 成功時更新 token 並存 localStorage', async () => {
    localStorage.setItem('refresh_token', 'old-refresh')
    mock.onPost('/api/v1/auth/refresh').reply(200, {
      data: { tokens: TOKENS },
    })

    await restoreSession()

    expect(getAccessToken()).toBe('new-access')
    expect(localStorage.getItem('refresh_token')).toBe('new-refresh')
  })

  it('refresh 失敗時清除 localStorage 的 refresh_token', async () => {
    localStorage.setItem('refresh_token', 'bad-refresh')
    mock.onPost('/api/v1/auth/refresh').reply(401)

    await expect(restoreSession()).rejects.toThrow()
    expect(localStorage.getItem('refresh_token')).toBeNull()
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
})

describe('auth basics', () => {
  it('login 後 isAuthenticated 為 true，logout 後回到 false', async () => {
    mock.onPost('/api/v1/auth/login').reply(200, {
      data: {
        user: {},
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
