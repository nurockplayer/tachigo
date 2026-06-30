import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { RaffleDrawSession } from '../RaffleDrawSession'
import type { RafflePrizeTier } from '@/services/raffles'
import * as rafflesService from '@/services/raffles'

vi.mock('@/services/raffles', () => ({
  drawFromTier: vi.fn(),
  listDraws: vi.fn(),
}))

const mockTiers: RafflePrizeTier[] = [
  { id: 't1', raffle_id: 'r1', name: '一等獎', prize_description: 'Switch', winner_count: 1, drawn_count: 0, position: 0, created_at: '' },
  { id: 't2', raffle_id: 'r1', name: '二等獎', prize_description: 'AirPods', winner_count: 2, drawn_count: 0, position: 1, created_at: '' },
]

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(rafflesService.listDraws).mockResolvedValue([])
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

  it('按下「抽這一輪」後顯示得獎者名稱', async () => {
    vi.mocked(rafflesService.drawFromTier).mockResolvedValueOnce({
      id: 'd1', raffle_id: 'r1', entry_id: 'e1',
      claim_token: '', claim_expires_at: '', drawn_at: '',
      entry: { id: 'e1', raffle_id: 'r1', twitch_login: 'viewer1', display_name: 'Viewer One', created_at: '' },
      prize_tier_id: 't1',
    })

    render(<RaffleDrawSession raffleId="r1" tiers={mockTiers} />)
    fireEvent.click(screen.getByTestId('draw-round-button'))

    await waitFor(() => {
      expect(screen.getByTestId('winner-name')).not.toBeNull()
      expect(screen.getByTestId('winner-name').textContent).toContain('Viewer One')
    })
    expect(screen.getByTestId('next-round-button').textContent).toContain('繼續下一輪')
  })

  it('按下「繼續下一輪」後推進到第二輪', async () => {
    vi.mocked(rafflesService.drawFromTier).mockResolvedValueOnce({
      id: 'd1', raffle_id: 'r1', entry_id: 'e1',
      claim_token: '', claim_expires_at: '', drawn_at: '',
      entry: { id: 'e1', raffle_id: 'r1', twitch_login: 'viewer1', display_name: 'Viewer One', created_at: '' },
      prize_tier_id: 't1',
    })

    render(<RaffleDrawSession raffleId="r1" tiers={mockTiers} />)
    fireEvent.click(screen.getByTestId('draw-round-button'))
    await waitFor(() => screen.getByTestId('next-round-button'))
    fireEvent.click(screen.getByTestId('next-round-button'))

    expect(screen.getByTestId('current-tier-name').textContent).toContain('二等獎')
    expect(screen.getByTestId('round-progress').textContent).toContain('第 2 / 2 輪')
  })

  it('最後一輪結束後顯示完成畫面', async () => {
    const singleTier: RafflePrizeTier[] = [
      { id: 't1', raffle_id: 'r1', name: '唯一獎', prize_description: '', winner_count: 1, drawn_count: 0, position: 0, created_at: '' },
    ]
    vi.mocked(rafflesService.drawFromTier).mockResolvedValueOnce({
      id: 'd1', raffle_id: 'r1', entry_id: 'e1',
      claim_token: '', claim_expires_at: '', drawn_at: '',
      entry: { id: 'e1', raffle_id: 'r1', twitch_login: 'winner', display_name: 'Winner', created_at: '' },
      prize_tier_id: 't1',
    })

    render(<RaffleDrawSession raffleId="r1" tiers={singleTier} />)
    fireEvent.click(screen.getByTestId('draw-round-button'))
    await waitFor(() => expect(screen.getByTestId('next-round-button').textContent).toContain('完成抽獎'))
    fireEvent.click(screen.getByTestId('next-round-button'))

    expect(screen.getByTestId('session-complete')).not.toBeNull()
  })

  it('winner_count: 2 的輪次：第一次抽完按鈕顯示「繼續抽本輪」，第二次抽完才顯示完成', async () => {
    const twoWinnerTier: RafflePrizeTier[] = [
      { id: 't1', raffle_id: 'r1', name: '一等獎', prize_description: 'Switch', winner_count: 2, drawn_count: 0, position: 0, created_at: '' },
    ]
    const mockEntry = { id: 'e1', raffle_id: 'r1', twitch_login: 'viewer1', display_name: 'Viewer One', created_at: '' }
    vi.mocked(rafflesService.drawFromTier)
      .mockResolvedValueOnce({
        id: 'd1', raffle_id: 'r1', entry_id: 'e1',
        claim_token: '', claim_expires_at: '', drawn_at: '',
        entry: mockEntry,
        prize_tier_id: 't1',
      })
      .mockResolvedValueOnce({
        id: 'd2', raffle_id: 'r1', entry_id: 'e2',
        claim_token: '', claim_expires_at: '', drawn_at: '',
        entry: { ...mockEntry, id: 'e2', twitch_login: 'viewer2', display_name: 'Viewer Two' },
        prize_tier_id: 't1',
      })

    render(<RaffleDrawSession raffleId="r1" tiers={twoWinnerTier} />)

    // 第一次抽
    fireEvent.click(screen.getByTestId('draw-round-button'))
    await waitFor(() => expect(screen.getByTestId('next-round-button').textContent).toContain('繼續抽本輪'))

    // 點「繼續抽本輪」→ 回到 round_ready
    fireEvent.click(screen.getByTestId('next-round-button'))
    await waitFor(() => expect(screen.getByTestId('draw-round-button')).not.toBeNull())

    // 第二次抽
    fireEvent.click(screen.getByTestId('draw-round-button'))
    await waitFor(() => expect(screen.getByTestId('next-round-button').textContent).toContain('完成抽獎'))

    // 點「完成抽獎」→ session_complete
    fireEvent.click(screen.getByTestId('next-round-button'))
    expect(screen.getByTestId('session-complete')).not.toBeNull()
  })

  it('API 失敗時顯示錯誤訊息，回到 round_ready', async () => {
    vi.mocked(rafflesService.drawFromTier).mockRejectedValueOnce(new Error('network error'))

    const { container } = render(<RaffleDrawSession raffleId="r1" tiers={mockTiers} />)
    fireEvent.click(screen.getByTestId('draw-round-button'))

    await waitFor(() => {
      expect(screen.getByTestId('draw-round-button')).not.toBeNull()
      expect(container.textContent).toContain('抽獎失敗，請再試一次')
    })
  })

  it('全部輪次抽完後，依輪次分組顯示完整得獎名單', async () => {
    const completedTiers: RafflePrizeTier[] = [
      { id: 't1', raffle_id: 'r1', name: '一等獎', prize_description: 'Switch', winner_count: 1, drawn_count: 1, position: 0, created_at: '' },
      { id: 't2', raffle_id: 'r1', name: '二等獎', prize_description: 'AirPods', winner_count: 2, drawn_count: 2, position: 1, created_at: '' },
    ]
    vi.mocked(rafflesService.listDraws).mockResolvedValueOnce([
      {
        id: 'd1', raffle_id: 'r1', entry_id: 'e1',
        claim_token: '', claim_expires_at: '', drawn_at: '',
        entry: { id: 'e1', raffle_id: 'r1', twitch_login: 'viewer1', display_name: 'Winner One', created_at: '' },
        prize_tier_id: 't1',
      },
      {
        id: 'd2', raffle_id: 'r1', entry_id: 'e2',
        claim_token: '', claim_expires_at: '', drawn_at: '',
        entry: { id: 'e2', raffle_id: 'r1', twitch_login: 'viewer2', display_name: 'Winner Two', created_at: '' },
        prize_tier_id: 't2',
      },
      {
        id: 'd3', raffle_id: 'r1', entry_id: 'e3',
        claim_token: '', claim_expires_at: '', drawn_at: '',
        entry: { id: 'e3', raffle_id: 'r1', twitch_login: 'viewer3', display_name: 'Winner Three', created_at: '' },
        prize_tier_id: 't2',
      },
    ])

    render(<RaffleDrawSession raffleId="r1" tiers={completedTiers} />)

    await waitFor(() => {
      expect(screen.getByTestId('final-winner-list')).not.toBeNull()
    })

    const tier1Group = screen.getByTestId('final-winner-tier-t1')
    expect(tier1Group.textContent).toContain('一等獎')
    expect(tier1Group.textContent).toContain('Winner One')
    expect(tier1Group.textContent).not.toContain('Winner Two')

    const tier2Group = screen.getByTestId('final-winner-tier-t2')
    expect(tier2Group.textContent).toContain('二等獎')
    expect(tier2Group.textContent).toContain('Winner Two')
    expect(tier2Group.textContent).toContain('Winner Three')
  })
})
