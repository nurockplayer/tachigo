import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listRaffles, createRaffle, listDraws, drawNext, importCSV, completeRaffle, setDiscordWebhook } from '@/services/raffles'

const getMock = vi.fn()
const postMock = vi.fn()
const patchMock = vi.fn()

vi.mock('@/services/api', () => ({
  default: {
    get: (...a: unknown[]) => getMock(...a),
    post: (...a: unknown[]) => postMock(...a),
    patch: (...a: unknown[]) => patchMock(...a),
  },
}))

const mockRaffle = { id: 'r1', user_id: 'u1', title: 'Test', status: 'draft' as const, created_at: '', updated_at: '' }
beforeEach(() => { getMock.mockReset(); postMock.mockReset(); patchMock.mockReset() })

describe('listRaffles', () => {
  it('returns raffles array', async () => {
    getMock.mockResolvedValue({ data: { success: true, data: { raffles: [mockRaffle] } } })
    expect(await listRaffles()).toEqual([mockRaffle])
    expect(getMock).toHaveBeenCalledWith('/api/v1/dashboard/raffles')
  })
})

describe('createRaffle', () => {
  it('posts title and returns raffle', async () => {
    postMock.mockResolvedValue({ data: { success: true, data: { raffle: mockRaffle } } })
    expect(await createRaffle('Test')).toEqual(mockRaffle)
    expect(postMock).toHaveBeenCalledWith('/api/v1/dashboard/raffles', { title: 'Test' })
  })
})

const mockDraw = {
  id: 'd1', raffle_id: 'r1', entry_id: 'e1',
  claim_token: 'tok', claim_expires_at: '2026-12-31T00:00:00Z',
  drawn_at: '2026-01-01T10:00:00Z',
  entry: { id: 'e1', raffle_id: 'r1', twitch_login: 'viewer1', display_name: 'Viewer One', created_at: '' },
}

describe('listDraws', () => {
  it('fetches draws for a raffle', async () => {
    getMock.mockResolvedValue({ data: { success: true, data: { draws: [mockDraw] } } })
    const result = await listDraws('r1')
    expect(result).toEqual([mockDraw])
    expect(getMock).toHaveBeenCalledWith('/api/v1/dashboard/raffles/r1/draws')
  })
})

describe('drawNext', () => {
  it('posts to draw endpoint and returns draw', async () => {
    postMock.mockResolvedValue({ data: { success: true, data: { draw: mockDraw } } })
    const result = await drawNext('r1')
    expect(result).toEqual(mockDraw)
    expect(postMock).toHaveBeenCalledWith('/api/v1/dashboard/raffles/r1/draws', undefined)
  })
})

describe('importCSV', () => {
  it('posts FormData and returns imported/skipped counts', async () => {
    postMock.mockResolvedValue({ data: { success: true, data: { imported: 10, skipped: 2 } } })
    const file = new File(['login\n'], 'test.csv', { type: 'text/csv' })
    const result = await importCSV('r1', file)
    expect(result).toEqual({ imported: 10, skipped: 2 })
    const [url, body] = postMock.mock.calls[0]
    expect(url).toBe('/api/v1/dashboard/raffles/r1/entries/import-csv')
    expect(body).toBeInstanceOf(FormData)
  })
})

describe('completeRaffle', () => {
  it('posts to complete endpoint', async () => {
    postMock.mockResolvedValue({ data: { success: true, data: {} } })
    await completeRaffle('r1')
    expect(postMock).toHaveBeenCalledWith('/api/v1/dashboard/raffles/r1/complete', undefined)
  })
})

describe('setDiscordWebhook', () => {
  it('patches the endpoint with URL and returns configured=true', async () => {
    patchMock.mockResolvedValue({ data: { success: true, data: { discord_webhook_configured: true } } })
    const result = await setDiscordWebhook('r1', 'https://discord.com/api/webhooks/123/abc')
    expect(result).toBe(true)
    expect(patchMock).toHaveBeenCalledWith(
      '/api/v1/dashboard/raffles/r1/discord-webhook',
      { discord_webhook_url: 'https://discord.com/api/webhooks/123/abc' },
    )
  })

  it('patches with empty string to clear and returns configured=false', async () => {
    patchMock.mockResolvedValue({ data: { success: true, data: { discord_webhook_configured: false } } })
    const result = await setDiscordWebhook('r1', '')
    expect(result).toBe(false)
    expect(patchMock).toHaveBeenCalledWith(
      '/api/v1/dashboard/raffles/r1/discord-webhook',
      { discord_webhook_url: '' },
    )
  })
})
