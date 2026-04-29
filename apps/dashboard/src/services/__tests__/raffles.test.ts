import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listRaffles, createRaffle } from '@/services/raffles'

const getMock = vi.fn()
const postMock = vi.fn()

vi.mock('@/services/api', () => ({
  default: { get: (...a: unknown[]) => getMock(...a), post: (...a: unknown[]) => postMock(...a) },
}))

const mockRaffle = { id: 'r1', user_id: 'u1', title: 'Test', status: 'draft' as const, created_at: '', updated_at: '' }
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
