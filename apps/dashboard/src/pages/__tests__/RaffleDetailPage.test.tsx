import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { MemoryRouter, Route, Routes } from 'react-router'
import type { BaseRecord } from '@refinedev/core'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import RaffleDetailPage from '@/pages/RaffleDetailPage'
import * as rafflesService from '@/services/raffles'
import { createMockDataProvider, RefineWrapper, waitFor } from '@/test/refine-wrapper'

vi.mock('@/services/raffles', async (importOriginal) => {
  const actual = await importOriginal<typeof rafflesService>()
  return {
    ...actual,
    listDraws: vi.fn().mockResolvedValue([]),
    drawNext: vi.fn().mockResolvedValue({}),
    importCSV: vi.fn().mockResolvedValue({ imported: 0, skipped: 0 }),
    completeRaffle: vi.fn().mockResolvedValue(undefined),
    setDiscordWebhook: vi.fn().mockResolvedValue(true),
    activateRaffle: vi.fn().mockResolvedValue({ id: 'r1', status: 'active' }),
    listPrizeTiers: vi.fn().mockResolvedValue([]),
    createPrizeTier: vi.fn().mockResolvedValue({}),
    deletePrizeTier: vi.fn().mockResolvedValue(undefined),
    drawFromTier: vi.fn().mockResolvedValue({}),
    setRaffleMode: vi.fn().mockResolvedValue({}),
    setRaffleEntryOpen: vi.fn().mockResolvedValue({}),
    getRaffleEntryStats: vi.fn().mockResolvedValue({
      eligible_count: 0,
      ineligible_count: 0,
      ineligible_reasons: {},
      total_joined: 0,
    }),
  }
})

const mockRaffle = {
  id: 'r1',
  user_id: 'u1',
  title: '春季抽獎',
  status: 'active' as const,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}
const mockDraw: rafflesService.RaffleDraw = {
  id: 'd1',
  raffle_id: 'r1',
  entry_id: 'e1',
  claim_token: 'tok',
  claim_expires_at: '2026-12-31T00:00:00Z',
  drawn_at: new Date().toISOString(),
  entry: {
    id: 'e1',
    raffle_id: 'r1',
    twitch_login: 'viewer1',
    display_name: 'Viewer One',
    created_at: '',
  },
}
const mockPrizeTier: rafflesService.RafflePrizeTier = {
  id: 't1',
  raffle_id: 'r1',
  name: '一等獎',
  prize_description: 'Switch 主機',
  winner_count: 2,
  drawn_count: 0,
  created_at: '2026-01-01T00:00:00Z',
}
async function renderAt(raffleId: string, dp: ReturnType<typeof createMockDataProvider>) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  await act(async () => {
    root.render(
      <RefineWrapper dataProvider={dp}>
        <MemoryRouter initialEntries={[`/raffles/${raffleId}`]}>
          <Routes>
            <Route path="/raffles/:raffleId" element={<RaffleDetailPage />} />
          </Routes>
        </MemoryRouter>
      </RefineWrapper>,
    )
  })
  return { container, root }
}
function cleanup(root: Root, container: HTMLDivElement) {
  act(() => {
    root.unmount()
  })
  container.remove()
}
function statValue(container: HTMLElement, label: string): string | null {
  const labelEl = Array.from(container.querySelectorAll('p')).find(p => p.textContent === label)
  return (labelEl?.nextElementSibling as HTMLElement | null)?.textContent ?? null
}
beforeEach(() => {
  vi.spyOn(console, 'error').mockImplementation(() => {})
  vi.mocked(rafflesService.listDraws).mockResolvedValue([])
  vi.mocked(rafflesService.drawNext).mockResolvedValue(mockDraw)
  vi.mocked(rafflesService.importCSV).mockResolvedValue({ imported: 0, skipped: 0 })
  vi.mocked(rafflesService.completeRaffle).mockResolvedValue(undefined)
  vi.mocked(rafflesService.setDiscordWebhook).mockResolvedValue(true)
  vi.mocked(rafflesService.activateRaffle).mockResolvedValue({ ...mockRaffle, status: 'active' as const })
  vi.mocked(rafflesService.listPrizeTiers).mockResolvedValue([])
  vi.mocked(rafflesService.createPrizeTier).mockResolvedValue(mockPrizeTier)
  vi.mocked(rafflesService.deletePrizeTier).mockResolvedValue(undefined)
  vi.mocked(rafflesService.drawFromTier).mockResolvedValue(mockDraw)
  vi.mocked(rafflesService.setRaffleMode).mockResolvedValue(mockRaffle)
  vi.mocked(rafflesService.setRaffleEntryOpen).mockResolvedValue(mockRaffle)
  vi.mocked(rafflesService.getRaffleEntryStats).mockResolvedValue({
    eligible_count: 0,
    ineligible_count: 0,
    ineligible_reasons: {},
    total_joined: 0,
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

describe('RaffleDetailPage — loading & raffle display', () => {
  it('shows skeleton while loading', async () => {
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockReturnValue(new Promise(() => {})) },
    })
    const { container, root } = await renderAt('r1', dp)
    expect(container.querySelector('[data-testid="skeleton"]')).toBeTruthy()
    cleanup(root, container)
  })
  it('displays raffle title and status after load', async () => {
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.textContent).toContain('春季抽獎'))
    expect(container.textContent).toContain('進行中')
    cleanup(root, container)
  })

  it('shows error when raffle fails to load', async () => {
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockRejectedValue(new Error('boom')) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.textContent).toContain('無法載入抽獎活動'))
    cleanup(root, container)
  })
})

describe('RaffleDetailPage — winner list', () => {
  it('shows empty state when no draws', async () => {
    vi.mocked(rafflesService.listDraws).mockResolvedValue([])
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="empty-winners"]')).toBeTruthy())
    cleanup(root, container)
  })

  it('renders winner display_name', async () => {
    vi.mocked(rafflesService.listDraws).mockResolvedValue([mockDraw])
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.textContent).toContain('Viewer One'))
    cleanup(root, container)
  })

  it('falls back to twitch_login when display_name is empty', async () => {
    const draw = { ...mockDraw, entry: { ...mockDraw.entry, display_name: '' } }
    vi.mocked(rafflesService.listDraws).mockResolvedValue([draw])
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.textContent).toContain('viewer1'))
    cleanup(root, container)
  })

  it('renders prize tier badge for tier-specific draws', async () => {
    vi.mocked(rafflesService.listDraws).mockResolvedValue([
      { ...mockDraw, prize_tier_id: 't1', prize_tier: mockPrizeTier },
    ])
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.textContent).toContain('Viewer One'))
    expect(container.textContent).toContain('一等獎')
    cleanup(root, container)
  })
})

describe('RaffleDetailPage — prize tiers', () => {
  it('hides prize tiers while raffle is still draft', async () => {
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(draftRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.textContent).toContain('春季抽獎'))
    expect(container.querySelector('[data-testid="prize-tiers-section"]')).toBeFalsy()
    cleanup(root, container)
  })

  it('shows active raffle prize tiers and disables completed tiers', async () => {
    vi.mocked(rafflesService.listPrizeTiers).mockResolvedValue([
      mockPrizeTier,
      { ...mockPrizeTier, id: 't2', name: '二等獎', winner_count: 1, drawn_count: 1 },
    ])
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="prize-tier-row-t1"]')).toBeTruthy())
    expect(container.textContent).toContain('Switch 主機')
    expect(container.querySelector('[data-testid="prize-tier-draw-t1"]')).toBeTruthy()
    expect((container.querySelector('[data-testid="prize-tier-draw-t2"]') as HTMLButtonElement).disabled).toBe(true)
    cleanup(root, container)
  })

  it('creates a prize tier from the active raffle panel', async () => {
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="prize-tier-toggle"]')).toBeTruthy())

    await act(async () => {
      container.querySelector('[data-testid="prize-tier-toggle"]')!
        .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    const name = container.querySelector('[data-testid="prize-tier-name"]') as HTMLInputElement
    const description = container.querySelector('[data-testid="prize-tier-description"]') as HTMLInputElement
    const count = container.querySelector('[data-testid="prize-tier-winner-count"]') as HTMLInputElement
    await act(async () => {
      name.value = '一等獎'
      name.dispatchEvent(new Event('input', { bubbles: true }))
      description.value = 'Switch 主機'
      description.dispatchEvent(new Event('input', { bubbles: true }))
      count.value = '2'
      count.dispatchEvent(new Event('input', { bubbles: true }))
    })
    await act(async () => {
      container.querySelector('[data-testid="prize-tier-add"]')!
        .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    await waitFor(() => expect(rafflesService.createPrizeTier).toHaveBeenCalledWith('r1', {
      name: '一等獎',
      prize_description: 'Switch 主機',
      winner_count: 2,
    }))
    cleanup(root, container)
  })

  it('does not create a prize tier when winner count is empty', async () => {
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="prize-tier-toggle"]')).toBeTruthy())

    await act(async () => {
      container.querySelector('[data-testid="prize-tier-toggle"]')!
        .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    const name = container.querySelector('[data-testid="prize-tier-name"]') as HTMLInputElement
    const count = container.querySelector('[data-testid="prize-tier-winner-count"]') as HTMLInputElement
    vi.mocked(rafflesService.createPrizeTier).mockClear()
    await act(async () => {
      name.value = '一等獎'
      name.dispatchEvent(new Event('input', { bubbles: true }))
      count.value = ''
      count.dispatchEvent(new Event('input', { bubbles: true }))
    })

    const add = container.querySelector('[data-testid="prize-tier-add"]') as HTMLButtonElement
    expect(add.disabled).toBe(true)

    await act(async () => {
      add.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    expect(rafflesService.createPrizeTier).not.toHaveBeenCalled()
    cleanup(root, container)
  })

  it('draws and deletes prize tiers through tier actions', async () => {
    vi.mocked(rafflesService.listPrizeTiers).mockResolvedValue([mockPrizeTier])
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="prize-tier-draw-t1"]')).toBeTruthy())

    await act(async () => {
      container.querySelector('[data-testid="prize-tier-draw-t1"]')!
        .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => expect(rafflesService.drawFromTier).toHaveBeenCalledWith('r1', 't1'))

    await act(async () => {
      container.querySelector('[data-testid="prize-tier-delete-t1"]')!
        .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => expect(rafflesService.deletePrizeTier).toHaveBeenCalledWith('r1', 't1'))
    cleanup(root, container)
  })
})

const draftRaffle = { ...mockRaffle, status: 'draft' as const }

describe('RaffleDetailPage — CSV upload', () => {
  it('shows success message after upload', async () => {
    vi.mocked(rafflesService.importCSV).mockResolvedValue({ imported: 50, skipped: 2 })
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(draftRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="csv-input"]')).toBeTruthy())

    const input = container.querySelector('[data-testid="csv-input"]') as HTMLInputElement
    const file = new File(['login\n'], 'test.csv', { type: 'text/csv' })
    Object.defineProperty(input, 'files', { value: [file], configurable: true })
    await act(async () => {
      input.dispatchEvent(new Event('change', { bubbles: true }))
    })
    await waitFor(() => {
      const el = container.querySelector('[data-testid="csv-success"]')
      expect(el?.textContent).toContain('50 人')
      expect(el?.textContent).toContain('略過 2 人')
    })
    cleanup(root, container)
  })

  it('shows error message when upload fails', async () => {
    vi.mocked(rafflesService.importCSV).mockRejectedValue(new Error('network'))
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(draftRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="csv-input"]')).toBeTruthy())

    const input = container.querySelector('[data-testid="csv-input"]') as HTMLInputElement
    const file = new File(['login\n'], 'bad.csv', { type: 'text/csv' })
    Object.defineProperty(input, 'files', { value: [file], configurable: true })
    await act(async () => {
      input.dispatchEvent(new Event('change', { bubbles: true }))
    })
    await waitFor(() => expect(container.querySelector('[data-testid="csv-error"]')?.textContent).toContain('上傳失敗'))
    cleanup(root, container)
  })
})

describe('RaffleDetailPage — draw button', () => {
  it('calls drawNext when clicked', async () => {
    const drawMock = vi.mocked(rafflesService.drawNext).mockResolvedValue(mockDraw)
    vi.mocked(rafflesService.getRaffleEntryStats).mockResolvedValue({
      eligible_count: 1,
      ineligible_count: 0,
      ineligible_reasons: {},
      total_joined: 1,
    })
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="draw-btn"]')).toBeTruthy())
    await waitFor(() => expect((container.querySelector('[data-testid="draw-btn"]') as HTMLButtonElement).disabled).toBe(false))

    await act(async () => {
      container.querySelector('[data-testid="draw-btn"]')!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => expect(drawMock).toHaveBeenCalledWith('r1'))
    cleanup(root, container)
  })

  it('disables draw button after 409 exhausted response', async () => {
    vi.mocked(rafflesService.drawNext).mockRejectedValue({ response: { status: 409 } })
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="draw-btn"]')).toBeTruthy())

    await act(async () => {
      container.querySelector('[data-testid="draw-btn"]')!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => {
      expect((container.querySelector('[data-testid="draw-btn"]') as HTMLButtonElement).disabled).toBe(true)
    })
    cleanup(root, container)
  })

  it('disables draw button when raffle is completed', async () => {
    const completedRaffle = { ...mockRaffle, status: 'completed' as const }
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(completedRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.textContent).toContain('春季抽獎'))
    expect((container.querySelector('[data-testid="draw-btn"]') as HTMLButtonElement).disabled).toBe(true)
    cleanup(root, container)
  })

  it('re-enables draw button after re-importing new entries when all previous draws are exhausted', async () => {
    vi.mocked(rafflesService.listDraws).mockResolvedValue([mockDraw, { ...mockDraw, id: 'd2' }, { ...mockDraw, id: 'd3' }])
    vi.mocked(rafflesService.importCSV).mockResolvedValue({ imported: 3, skipped: 0 })
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(draftRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="csv-input"]')).toBeTruthy())

    const input = container.querySelector('[data-testid="csv-input"]') as HTMLInputElement
    const file = new File(['login\n'], 'first.csv', { type: 'text/csv' })
    Object.defineProperty(input, 'files', { value: [file], configurable: true })
    await act(async () => {
      input.dispatchEvent(new Event('change', { bubbles: true }))
    })
    await waitFor(() => {
      expect((container.querySelector('[data-testid="draw-btn"]') as HTMLButtonElement).disabled).toBe(true)
    })

    vi.mocked(rafflesService.importCSV).mockResolvedValue({ imported: 2, skipped: 0 })
    const file2 = new File(['login2\n'], 'second.csv', { type: 'text/csv' })
    Object.defineProperty(input, 'files', { value: [file2], configurable: true })
    await act(async () => {
      input.dispatchEvent(new Event('change', { bubbles: true }))
    })
    await waitFor(() => {
      expect((container.querySelector('[data-testid="draw-btn"]') as HTMLButtonElement).disabled).toBe(false)
    })
    cleanup(root, container)
  })
})

describe('RaffleDetailPage — end activity', () => {
  it('shows confirm dialog when end button clicked', async () => {
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="end-btn"]')).toBeTruthy())

    await act(async () => {
      container.querySelector('[data-testid="end-btn"]')!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    expect(container.querySelector('[data-testid="confirm-end"]')).toBeTruthy()
    expect(container.querySelector('[data-testid="end-btn"]')).toBeFalsy()
    cleanup(root, container)
  })

  it('cancels confirm dialog on 取消', async () => {
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="end-btn"]')).toBeTruthy())

    await act(async () => {
      container.querySelector('[data-testid="end-btn"]')!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await act(async () => {
      container.querySelector('[data-testid="confirm-no"]')!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    expect(container.querySelector('[data-testid="confirm-end"]')).toBeFalsy()
    expect(container.querySelector('[data-testid="end-btn"]')).toBeTruthy()
    cleanup(root, container)
  })

  it('calls completeRaffle and hides end button after confirm', async () => {
    const completeMock = vi.mocked(rafflesService.completeRaffle).mockResolvedValue(undefined)
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="end-btn"]')).toBeTruthy())

    await act(async () => {
      container.querySelector('[data-testid="end-btn"]')!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await act(async () => {
      container.querySelector('[data-testid="confirm-yes"]')!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => expect(completeMock).toHaveBeenCalledWith('r1'))
    await waitFor(() => expect(container.querySelector('[data-testid="end-btn"]')).toBeFalsy())
    expect((container.querySelector('[data-testid="draw-btn"]') as HTMLButtonElement).disabled).toBe(true)
    cleanup(root, container)
  })
})

describe('RaffleDetailPage — Discord webhook', () => {
  it('shows Discord webhook settings section', async () => {
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="discord-webhook-input"]')).toBeTruthy())
    cleanup(root, container)
  })

  it('calls setDiscordWebhook with URL when save button clicked', async () => {
    const webhookMock = vi.mocked(rafflesService.setDiscordWebhook).mockResolvedValue(true)
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="discord-webhook-input"]')).toBeTruthy())

    const input = container.querySelector('[data-testid="discord-webhook-input"]') as HTMLInputElement
    await act(async () => {
      input.value = 'https://discord.com/api/webhooks/123/abc'
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })
    await act(async () => {
      container.querySelector('[data-testid="discord-webhook-save"]')!
        .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => expect(webhookMock).toHaveBeenCalledWith('r1', 'https://discord.com/api/webhooks/123/abc'))
    cleanup(root, container)
  })

  it('shows configured status after successful save', async () => {
    vi.mocked(rafflesService.setDiscordWebhook).mockResolvedValue(true)
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="discord-webhook-input"]')).toBeTruthy())

    const input = container.querySelector('[data-testid="discord-webhook-input"]') as HTMLInputElement
    await act(async () => {
      input.value = 'https://discord.com/api/webhooks/123/abc'
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })
    await act(async () => {
      container.querySelector('[data-testid="discord-webhook-save"]')!
        .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => expect(container.querySelector('[data-testid="discord-webhook-status"]')?.textContent).toContain('已設定'))
    cleanup(root, container)
  })

  it('calls setDiscordWebhook with empty string when clear button clicked', async () => {
    const webhookMock = vi.mocked(rafflesService.setDiscordWebhook).mockResolvedValue(false)
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="discord-webhook-clear"]')).toBeTruthy())

    await act(async () => {
      container.querySelector('[data-testid="discord-webhook-clear"]')!
        .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => expect(webhookMock).toHaveBeenCalledWith('r1', ''))
    cleanup(root, container)
  })

  it('shows error message when save fails', async () => {
    vi.mocked(rafflesService.setDiscordWebhook).mockRejectedValue({ response: { data: { error: 'invalid discord webhook url' } } })
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="discord-webhook-input"]')).toBeTruthy())

    const input = container.querySelector('[data-testid="discord-webhook-input"]') as HTMLInputElement
    await act(async () => {
      input.value = 'not-a-valid-url'
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })
    await act(async () => {
      container.querySelector('[data-testid="discord-webhook-save"]')!
        .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => expect(container.querySelector('[data-testid="discord-webhook-error"]')?.textContent).toContain('invalid discord webhook url'))
    cleanup(root, container)
  })
})

describe('RaffleDetailPage — activate button', () => {
  it('shows activate button when status is draft', async () => {
    const draftRaffle = { ...mockRaffle, status: 'draft' as const }
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(draftRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="activate-btn"]')).toBeTruthy())
    cleanup(root, container)
  })

  it('calls activateRaffle when activate button is clicked', async () => {
    const activateMock = vi.mocked(rafflesService.activateRaffle)
    const draftRaffle = { ...mockRaffle, status: 'draft' as const }
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(draftRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="activate-btn"]')).toBeTruthy())

    await act(async () => {
      container.querySelector('[data-testid="activate-btn"]')!
        .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => expect(activateMock).toHaveBeenCalledWith('r1'))
    cleanup(root, container)
  })

  it('hides activate button when status is active', async () => {
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="draw-btn"]')).toBeTruthy())
    expect(container.querySelector('[data-testid="activate-btn"]')).toBeFalsy()
    cleanup(root, container)
  })

  it('shows CSV locked message when status is active', async () => {
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="csv-locked"]')).toBeTruthy())
    cleanup(root, container)
  })

  it('shows error message when activateRaffle fails', async () => {
    vi.mocked(rafflesService.activateRaffle).mockRejectedValue({
      response: { data: { error: 'raffle is not in draft status' } },
    })
    const draftRaffle = { ...mockRaffle, status: 'draft' as const }
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(draftRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="activate-btn"]')).toBeTruthy())

    await act(async () => {
      container.querySelector('[data-testid="activate-btn"]')!
        .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() =>
      expect(container.querySelector('[data-testid="activate-error"]')?.textContent)
        .toContain('raffle is not in draft status'),
    )
    cleanup(root, container)
  })
})

describe('RaffleDetailPage — winner modal', () => {
  it('shows winner modal after draw completes', async () => {
    vi.mocked(rafflesService.drawNext).mockResolvedValue(mockDraw)
    vi.mocked(rafflesService.listDraws).mockResolvedValue([mockDraw])
    vi.mocked(rafflesService.getRaffleEntryStats).mockResolvedValue({
      eligible_count: 2,
      ineligible_count: 0,
      ineligible_reasons: {},
      total_joined: 2,
    })
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="draw-btn"]')).toBeTruthy())
    await waitFor(() => expect((container.querySelector('[data-testid="draw-btn"]') as HTMLButtonElement).disabled).toBe(false))

    await act(async () => {
      container.querySelector('[data-testid="draw-btn"]')!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => expect(container.textContent).toContain('恭喜中獎'), 2000)
    expect(container.textContent).toContain('Viewer One')
    cleanup(root, container)
  })

  it('closes modal when 繼續抽獎 button clicked', async () => {
    vi.mocked(rafflesService.drawNext).mockResolvedValue(mockDraw)
    vi.mocked(rafflesService.listDraws).mockResolvedValue([mockDraw])
    vi.mocked(rafflesService.getRaffleEntryStats).mockResolvedValue({
      eligible_count: 2,
      ineligible_count: 0,
      ineligible_reasons: {},
      total_joined: 2,
    })
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="draw-btn"]')).toBeTruthy())
    await waitFor(() => expect((container.querySelector('[data-testid="draw-btn"]') as HTMLButtonElement).disabled).toBe(false))

    await act(async () => {
      container.querySelector('[data-testid="draw-btn"]')!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => expect(container.textContent).toContain('恭喜中獎'), 2000)

    const closeBtn = Array.from(container.querySelectorAll('button')).find(b => b.textContent?.includes('繼續抽獎'))
    await act(async () => {
      closeBtn!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    expect(container.textContent).not.toContain('恭喜中獎')
    cleanup(root, container)
  })
})

describe('RaffleDetailPage functional layer', () => {
  it('mode switch calls setRaffleMode and updates UI', async () => {
    const modeMock = vi.mocked(rafflesService.setRaffleMode).mockResolvedValue({ ...mockRaffle, mode: 'subscribers_only' })
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue({ ...mockRaffle, mode: 'public', entry_open: false } as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="raffle-mode-subscribers"]')).toBeTruthy())

    const subscribers = container.querySelector('[data-testid="raffle-mode-subscribers"]') as HTMLInputElement
    await act(async () => {
      subscribers.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    await waitFor(() => expect(modeMock).toHaveBeenCalledWith('r1', 'subscribers_only'))
    expect(subscribers.checked).toBe(true)
    cleanup(root, container)
  })

  it('entry open/close button calls setRaffleEntryOpen and reflects state', async () => {
    const entryOpenMock = vi.mocked(rafflesService.setRaffleEntryOpen).mockResolvedValue({ ...mockRaffle, entry_open: true })
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue({ ...mockRaffle, mode: 'public', entry_open: false } as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="entry-open-toggle"]')?.textContent).toContain('開放報名'))

    await act(async () => {
      container.querySelector('[data-testid="entry-open-toggle"]')!
        .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    await waitFor(() => expect(entryOpenMock).toHaveBeenCalledWith('r1', true))
    expect(container.querySelector('[data-testid="entry-open-toggle"]')?.textContent).toContain('截止報名')
    cleanup(root, container)
  })

  it('subscribers_only + 403 insufficient_scope shows re-auth prompt', async () => {
    vi.mocked(rafflesService.getRaffleEntryStats).mockRejectedValue({
      response: { status: 403, data: { error: 'insufficient_scope' } },
    })
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue({ ...mockRaffle, mode: 'subscribers_only', entry_open: false } as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)

    await waitFor(() => expect(container.querySelector('[data-testid="twitch-reauth-prompt"]')?.textContent).toContain('請重新授權 Twitch'))
    expect(container.querySelector('[data-testid="twitch-reauth-prompt"] a')?.getAttribute('href')).toContain('/api/v1/auth/twitch')
    cleanup(root, container)
  })

  it('syncs total entries and remaining count from entry stats', async () => {
    vi.mocked(rafflesService.listDraws).mockResolvedValue([mockDraw, { ...mockDraw, id: 'd2' }])
    vi.mocked(rafflesService.getRaffleEntryStats).mockResolvedValue({
      eligible_count: 8,
      ineligible_count: 2,
      ineligible_reasons: {},
      total_joined: 10,
    })
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue({ ...mockRaffle, mode: 'public', entry_open: true } as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)

    await waitFor(() => expect(container.querySelector('[data-testid="entry-stats-panel"]')).toBeTruthy())
    await waitFor(() => expect(statValue(container, '匯入人數')).toBe('10'))
    expect(statValue(container, '剩餘')).toBe('8')
    cleanup(root, container)
  })

  it('entry-open toggle 403 sets controlReauthRequired and shows re-auth prompt', async () => {
    vi.mocked(rafflesService.setRaffleEntryOpen).mockRejectedValue({
      response: { status: 403, data: { error: 'insufficient_scope' } },
    })
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue({ ...mockRaffle, mode: 'subscribers_only', entry_open: false } as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="entry-open-toggle"]')).toBeTruthy())
    expect(container.querySelector('[data-testid="twitch-reauth-prompt"]')).toBeFalsy()

    await act(async () => {
      container.querySelector('[data-testid="entry-open-toggle"]')!
        .dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    await waitFor(() => expect(container.querySelector('[data-testid="twitch-reauth-prompt"]')?.textContent).toContain('請重新授權 Twitch'))
    cleanup(root, container)
  })

  it('disables mode radios while modeChanging and entry-open button while entryOpenChanging', async () => {
    let resolveMode: (value: rafflesService.Raffle) => void = () => {}
    vi.mocked(rafflesService.setRaffleMode).mockImplementation(() => new Promise<rafflesService.Raffle>((resolve) => { resolveMode = resolve }))
    let resolveEntryOpen: (value: rafflesService.Raffle) => void = () => {}
    vi.mocked(rafflesService.setRaffleEntryOpen).mockImplementation(() => new Promise<rafflesService.Raffle>((resolve) => { resolveEntryOpen = resolve }))

    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue({ ...mockRaffle, mode: 'public', entry_open: false } as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="raffle-mode-subscribers"]')).toBeTruthy())

    const publicRadio = container.querySelector('[data-testid="raffle-mode-public"]') as HTMLInputElement
    const subscribersRadio = container.querySelector('[data-testid="raffle-mode-subscribers"]') as HTMLInputElement
    const entryToggle = container.querySelector('[data-testid="entry-open-toggle"]') as HTMLButtonElement

    await act(async () => {
      subscribersRadio.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => expect(subscribersRadio.disabled).toBe(true))
    expect(publicRadio.disabled).toBe(true)
    expect(entryToggle.disabled).toBe(false)

    await act(async () => {
      resolveMode({ ...mockRaffle, mode: 'subscribers_only' })
    })
    await waitFor(() => expect(subscribersRadio.disabled).toBe(false))

    await act(async () => {
      entryToggle.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => expect(entryToggle.disabled).toBe(true))
    expect(subscribersRadio.disabled).toBe(false)
    expect(publicRadio.disabled).toBe(false)

    await act(async () => {
      resolveEntryOpen({ ...mockRaffle, mode: 'subscribers_only', entry_open: true })
    })
    await waitFor(() => expect(entryToggle.disabled).toBe(false))

    cleanup(root, container)
  })
})
