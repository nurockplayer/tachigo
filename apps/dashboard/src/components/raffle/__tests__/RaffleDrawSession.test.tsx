import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { RaffleDrawSession } from '../RaffleDrawSession'
import type { RafflePrizeTier } from '@/services/raffles'

vi.mock('@/services/raffles', () => ({
  drawFromTier: vi.fn(),
}))

const mockTiers: RafflePrizeTier[] = [
  { id: 't1', raffle_id: 'r1', name: '一等獎', prize_description: 'Switch', winner_count: 1, drawn_count: 0, position: 0, created_at: '' },
  { id: 't2', raffle_id: 'r1', name: '二等獎', prize_description: 'AirPods', winner_count: 2, drawn_count: 0, position: 1, created_at: '' },
]

beforeEach(() => {
  vi.clearAllMocks()
})

describe('RaffleDrawSession', () => {
  it('顯示第一輪的獎項名稱與描述', () => {
    render(<RaffleDrawSession raffleId="r1" tiers={mockTiers} />)
    expect(screen.getByTestId('current-tier-name').textContent).toContain('一等獎')
    expect(screen.getByTestId('current-tier-description').textContent).toContain('Switch')
    expect(screen.getByTestId('draw-round-button')).not.toBeNull()
  })

  it('顯示輪次進度（第 1 / 2 輪）', () => {
    render(<RaffleDrawSession raffleId="r1" tiers={mockTiers} />)
    expect(screen.getByTestId('round-progress').textContent).toContain('第 1 / 2 輪')
  })
})
