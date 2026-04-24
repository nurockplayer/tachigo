import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listRaffles, createRaffle, getRaffle, importCSV, drawNext, listDraws, completeRaffle } from '@/services/raffles'

const getMock = vi.fn()
const postMock = vi.fn()

vi.mock('@/services/api', () => ({
  default: { get: (...a: unknown[]) => getMock(...a), post: (...a: unknown[]) => postMock(...a) },
}))

const mockRaffle = { id: 'r1', user_id: 'u1', title: 'Test', status: 'draft' as const, created_at: '', updated_at: '' }
const mockDraw = {
  id: 'd1', raffle_id: 'r1', entry_id: 'e1', claim_token: 'tok', claim_expires_at: '', drawn_at: '',
  entry: { id: 'e1', raffle_id: 'r1', twitch_login: 'alice', display_name: 'Alice', created_at: '' },
}

beforeEach(() => { getMock.mockReset(); postMock.mockReset() })

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

describe('getRaffle', () => {
  it('fetches raffle by id', async () => {
    getMock.mockResolvedValue({ data: { success: true, data: { raffle: mockRaffle } } })
    expect(await getRaffle('r1')).toEqual(mockRaffle)
    expect(getMock).toHaveBeenCalledWith('/api/v1/dashboard/raffles/r1')
  })
})

describe('importCSV', () => {
  it('posts FormData and returns imported count', async () => {
    postMock.mockResolvedValue({ data: { success: true, data: { imported: 42 } } })
    const file = new File(['a,b'], 'entries.csv', { type: 'text/csv' })
    expect(await importCSV('r1', file)).toBe(42)
    const [url, body, config] = postMock.mock.calls[0]
    expect(url).toBe('/api/v1/dashboard/raffles/r1/entries/import-csv')
    expect(body).toBeInstanceOf(FormData)
    expect(config.headers['Content-Type']).toBe('multipart/form-data')
  })
})

describe('drawNext', () => {
  it('posts to draws endpoint and returns draw', async () => {
    postMock.mockResolvedValue({ data: { success: true, data: { draw: mockDraw } } })
    expect(await drawNext('r1')).toEqual(mockDraw)
    expect(postMock).toHaveBeenCalledWith('/api/v1/dashboard/raffles/r1/draws')
  })
})

describe('listDraws', () => {
  it('fetches draws for raffle', async () => {
    getMock.mockResolvedValue({ data: { success: true, data: { draws: [mockDraw] } } })
    expect(await listDraws('r1')).toEqual([mockDraw])
    expect(getMock).toHaveBeenCalledWith('/api/v1/dashboard/raffles/r1/draws')
  })
})

describe('completeRaffle', () => {
  it('posts to complete endpoint', async () => {
    const completed = { ...mockRaffle, status: 'completed' as const }
    postMock.mockResolvedValue({ data: { success: true, data: { raffle: completed } } })
    expect(await completeRaffle('r1')).toEqual(completed)
    expect(postMock).toHaveBeenCalledWith('/api/v1/dashboard/raffles/r1/complete')
  })
})
