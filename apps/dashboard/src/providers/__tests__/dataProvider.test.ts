import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const mockClient = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
}))

vi.mock('@/services/api', () => ({
  default: mockClient,
  apiBaseURL: 'https://api.example.test/',
}))

describe('dataProvider API URL', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.unstubAllEnvs()
    mockClient.get.mockReset()
    mockClient.post.mockReset()
    mockClient.put.mockReset()
    mockClient.patch.mockReset()
    mockClient.delete.mockReset()
  })

  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('getApiUrl 與實際 CRUD 請求都使用 VITE_TACHIGO_API_URL', async () => {
    vi.stubEnv('VITE_TACHIGO_API_URL', 'https://api.example.test/')
    vi.stubEnv('VITE_API_URL', 'https://legacy.example.test')
    mockClient.get.mockResolvedValueOnce({ data: { data: { streamers: [] } } })

    const { dataProvider } = await import('@/providers/dataProvider')

    expect(dataProvider.getApiUrl()).toBe('https://api.example.test/api/v1')

    await dataProvider.getList({ resource: 'streamers' })

    expect(mockClient.get).toHaveBeenCalledWith(
      'https://api.example.test/api/v1/dashboard/streamers',
      {
        params: {
          pagination: undefined,
          sorters: undefined,
          filters: undefined,
        },
      },
    )
  })

  it('channel-configs update 走後端已註冊的 PUT config endpoint', async () => {
    vi.stubEnv('VITE_TACHIGO_API_URL', 'https://api.example.test/')
    mockClient.put.mockResolvedValueOnce({ data: { data: { config: { ratio: 2 } } } })

    const { dataProvider } = await import('@/providers/dataProvider')

    await dataProvider.update({
      resource: 'channel-configs',
      id: 'channel/1',
      variables: { ratio: 2 },
    })

    expect(mockClient.put).toHaveBeenCalledWith(
      'https://api.example.test/api/v1/dashboard/channels/channel%2F1/config',
      { ratio: 2 },
    )
    expect(mockClient.patch).not.toHaveBeenCalled()
  })

  it('transactions getList 走 points history endpoint 並傳遞 channel_id', async () => {
    vi.stubEnv('VITE_TACHIGO_API_URL', 'https://api.example.test/')
    mockClient.get.mockResolvedValueOnce({
      data: {
        data: {
          transactions: [
            {
              id: 'tx-1',
              type: 'earn',
              amount: 12,
            },
          ],
        },
      },
    })

    const { dataProvider } = await import('@/providers/dataProvider')

    const result = await dataProvider.getList({
      resource: 'transactions',
      meta: { params: { channel_id: 'channel-123' } },
    })

    expect(mockClient.get).toHaveBeenCalledWith(
      'https://api.example.test/api/v1/users/me/points/history',
      {
        params: {
          channel_id: 'channel-123',
          pagination: undefined,
          sorters: undefined,
          filters: undefined,
        },
      },
    )
    expect(result.data).toEqual([
      {
        id: 'tx-1',
        type: 'earn',
        amount: 12,
      },
    ])
  })

  it('settings resource 目前明確不支援，避免打到不存在的 dashboard settings endpoint', async () => {
    vi.stubEnv('VITE_TACHIGO_API_URL', 'https://api.example.test/')

    const { dataProvider } = await import('@/providers/dataProvider')

    await expect(dataProvider.getList({ resource: 'settings' })).rejects.toThrow(
      'Dashboard settings resource is not wired to a backend endpoint yet',
    )

    expect(mockClient.get).not.toHaveBeenCalled()
  })
})
