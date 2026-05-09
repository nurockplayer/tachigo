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
})
