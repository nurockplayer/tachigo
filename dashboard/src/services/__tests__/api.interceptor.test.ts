import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import MockAdapter from 'axios-mock-adapter'
import client, { hasAuthToken, setAuthToken, clearAuthToken } from '@/services/api'

let mock: InstanceType<typeof MockAdapter>

beforeEach(() => {
  mock = new MockAdapter(client)
  clearAuthToken()
})

afterEach(() => {
  mock.restore()
  clearAuthToken()
})

describe('hasAuthToken', () => {
  it('無 token 時回傳 false', () => {
    expect(hasAuthToken()).toBe(false)
  })

  it('setAuthToken 後回傳 true', () => {
    setAuthToken('some-token')
    expect(hasAuthToken()).toBe(true)
  })

  it('clearAuthToken 後回傳 false', () => {
    setAuthToken('some-token')
    clearAuthToken()
    expect(hasAuthToken()).toBe(false)
  })
})

describe('401 interceptor', () => {
  it('access token 過期時自動刷新並重試原始請求', async () => {
    setAuthToken('expired-token')

    mock
      .onGet('/api/v1/some-resource')
      .replyOnce(401)
      .onGet('/api/v1/some-resource')
      .replyOnce(200, { data: 'ok' })
    mock
      .onPost('/api/v1/auth/refresh')
      .replyOnce(200, { data: { tokens: { access_token: 'new-token' } } })

    const result = await client.get('/api/v1/some-resource')

    expect(result.data).toEqual({ data: 'ok' })
    expect(hasAuthToken()).toBe(true)
    // 重試請求必須帶新 token，不能帶舊的（axios 1.x error.config headers 已 flatten，retry 前需明確覆寫）
    expect(mock.history.get[1].headers?.Authorization).toBe('Bearer new-token')
  })

  it('並發多個 401 時只呼叫一次 /auth/refresh（dedupe）', async () => {
    setAuthToken('expired-token')

    mock
      .onGet('/api/v1/resource-a')
      .replyOnce(401)
      .onGet('/api/v1/resource-a')
      .replyOnce(200, { data: 'a' })
    mock
      .onGet('/api/v1/resource-b')
      .replyOnce(401)
      .onGet('/api/v1/resource-b')
      .replyOnce(200, { data: 'b' })
    mock
      .onPost('/api/v1/auth/refresh')
      .replyOnce(200, { data: { tokens: { access_token: 'new-token' } } })

    const [a, b] = await Promise.all([
      client.get('/api/v1/resource-a'),
      client.get('/api/v1/resource-b'),
    ])

    expect(a.data).toEqual({ data: 'a' })
    expect(b.data).toEqual({ data: 'b' })
    expect(mock.history.post.filter(r => r.url === '/api/v1/auth/refresh')).toHaveLength(1)
  })

  it('/auth/refresh 本身 401 時不觸發 retry loop，直接拋出', async () => {
    setAuthToken('expired-token')

    mock.onGet('/api/v1/some-resource').replyOnce(401)
    mock.onPost('/api/v1/auth/refresh').replyOnce(401)

    await expect(client.get('/api/v1/some-resource')).rejects.toMatchObject({
      response: { status: 401 },
    })
    expect(mock.history.post.filter(r => r.url === '/api/v1/auth/refresh')).toHaveLength(1)
  })

  it('非 401 錯誤不觸發 refresh', async () => {
    mock.onGet('/api/v1/some-resource').replyOnce(500)

    await expect(client.get('/api/v1/some-resource')).rejects.toMatchObject({
      response: { status: 500 },
    })
    expect(mock.history.post.filter(r => r.url === '/api/v1/auth/refresh')).toHaveLength(0)
  })

  it('refresh 失敗後清除 token', async () => {
    setAuthToken('expired-token')

    mock.onGet('/api/v1/some-resource').replyOnce(401)
    mock.onPost('/api/v1/auth/refresh').replyOnce(500)

    await expect(client.get('/api/v1/some-resource')).rejects.toBeDefined()
    expect(hasAuthToken()).toBe(false)
  })
})
