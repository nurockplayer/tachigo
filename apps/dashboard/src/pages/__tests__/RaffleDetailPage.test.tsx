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
beforeEach(() => {
  vi.spyOn(console, 'error').mockImplementation(() => {})
  vi.mocked(rafflesService.listDraws).mockResolvedValue([])
  vi.mocked(rafflesService.drawNext).mockResolvedValue(mockDraw)
  vi.mocked(rafflesService.importCSV).mockResolvedValue({ imported: 0, skipped: 0 })
  vi.mocked(rafflesService.completeRaffle).mockResolvedValue(undefined)
  vi.mocked(rafflesService.setDiscordWebhook).mockResolvedValue(true)
  vi.mocked(rafflesService.activateRaffle).mockResolvedValue({ ...mockRaffle, status: 'active' as const })
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
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="draw-btn"]')).toBeTruthy())

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
})
