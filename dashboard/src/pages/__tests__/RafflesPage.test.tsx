import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { MemoryRouter, Route, Routes, useParams } from 'react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import RafflesPage from '@/pages/RafflesPage'

const listRafflesMock = vi.fn()
const createRaffleMock = vi.fn()

vi.mock('@/services/raffles', () => ({
  listRaffles: (...a: unknown[]) => listRafflesMock(...a),
  createRaffle: (...a: unknown[]) => createRaffleMock(...a),
}))

function DetailProbe() {
  const { raffleId } = useParams()
  return <div data-testid="detail">{raffleId}</div>
}

function RoutedApp() {
  return (
    <Routes>
      <Route path="/raffles" element={<RafflesPage />} />
      <Route path="/raffles/:raffleId" element={<DetailProbe />} />
    </Routes>
  )
}

async function renderAt(path: string) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  await act(async () => {
    root.render(<MemoryRouter initialEntries={[path]}><RoutedApp /></MemoryRouter>)
  })
  return { container, root }
}

async function flush() {
  await act(async () => { await Promise.resolve() })
}

function cleanupRoot(root: Root, container: HTMLDivElement) {
  act(() => { root.unmount() })
  container.remove()
}

const mockRaffle = { id: 'r1', user_id: 'u1', title: '春季抽獎', status: 'draft' as const, created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z' }

describe('RafflesPage', () => {
  beforeEach(() => { listRafflesMock.mockReset(); createRaffleMock.mockReset() })
  afterEach(() => { document.body.innerHTML = '' })

  it('shows skeleton while loading', async () => {
    listRafflesMock.mockReturnValue(new Promise(() => {}))
    const { container, root } = await renderAt('/raffles')
    expect(container.querySelector('[data-testid="skeleton"]')).toBeTruthy()
    cleanupRoot(root, container)
  })

  it('renders raffle list after load', async () => {
    listRafflesMock.mockResolvedValue([mockRaffle])
    const { container, root } = await renderAt('/raffles')
    await flush()
    expect(container.textContent).toContain('春季抽獎')
    cleanupRoot(root, container)
  })

  it('shows empty state when no raffles', async () => {
    listRafflesMock.mockResolvedValue([])
    const { container, root } = await renderAt('/raffles')
    await flush()
    expect(container.textContent).toContain('尚無')
    cleanupRoot(root, container)
  })

  it('shows error message when API fails', async () => {
    listRafflesMock.mockRejectedValue(new Error('boom'))
    const { container, root } = await renderAt('/raffles')
    await flush()
    expect(container.textContent).toContain('無法載入')
    cleanupRoot(root, container)
  })

  it('navigates to detail page on row click', async () => {
    listRafflesMock.mockResolvedValue([mockRaffle])
    const { container, root } = await renderAt('/raffles')
    await flush()
    const row = container.querySelector('tbody tr') as HTMLElement
    await act(async () => { row.click() })
    expect(container.querySelector('[data-testid="detail"]')?.textContent).toBe('r1')
    cleanupRoot(root, container)
  })

  it('creates a raffle and adds it to the list', async () => {
    const newRaffle = { ...mockRaffle, id: 'r2', title: '夏季抽獎' }
    listRafflesMock.mockResolvedValue([mockRaffle])
    createRaffleMock.mockResolvedValue(newRaffle)
    const { container, root } = await renderAt('/raffles')
    await flush()

    const input = container.querySelector('input[name="title"]') as HTMLInputElement
    await act(async () => {
      const nativeInputValueSetter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set
      nativeInputValueSetter?.call(input, '夏季抽獎')
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })
    const form = container.querySelector('form') as HTMLFormElement
    await act(async () => { form.dispatchEvent(new Event('submit', { bubbles: true })) })
    await flush()

    expect(createRaffleMock).toHaveBeenCalledWith('夏季抽獎')
    expect(container.textContent).toContain('夏季抽獎')
    cleanupRoot(root, container)
  })

  it('falls back to a string message when create API error is not a string', async () => {
    listRafflesMock.mockResolvedValue([])
    createRaffleMock.mockRejectedValue({
      isAxiosError: true,
      response: { data: { error: { message: 'invalid title' } } },
    })
    const { container, root } = await renderAt('/raffles')
    await flush()

    const input = container.querySelector('input[name="title"]') as HTMLInputElement
    await act(async () => {
      const nativeInputValueSetter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set
      nativeInputValueSetter?.call(input, 'new raffle')
      input.dispatchEvent(new Event('input', { bubbles: true }))
    })
    const form = container.querySelector('form') as HTMLFormElement
    await act(async () => { form.dispatchEvent(new Event('submit', { bubbles: true })) })
    await flush()

    expect(container.textContent).toContain('建立失敗')
    cleanupRoot(root, container)
  })

  it('shows the created raffle after the initial list request fails', async () => {
    listRafflesMock.mockRejectedValue(new Error('boom')); createRaffleMock.mockResolvedValue({ ...mockRaffle, id: 'r2', title: 'manual raffle' })
    const { container, root } = await renderAt('/raffles')
    await flush()
    const input = container.querySelector('input[name="title"]') as HTMLInputElement
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.call(input, 'manual raffle'); await act(async () => { input.dispatchEvent(new Event('input', { bubbles: true })) })
    await act(async () => { container.querySelector('form')?.dispatchEvent(new Event('submit', { bubbles: true })) })
    await flush()
    expect(container.textContent).toContain('manual raffle')
    cleanupRoot(root, container)
  })

  it('disables submit button when title is empty', async () => {
    listRafflesMock.mockResolvedValue([])
    const { container, root } = await renderAt('/raffles')
    await flush()
    const btn = container.querySelector('button[type="submit"]') as HTMLButtonElement
    expect(btn.disabled).toBe(true)
    cleanupRoot(root, container)
  })

  it('navigates to detail page on Enter key press', async () => {
    listRafflesMock.mockResolvedValue([mockRaffle])
    const { container, root } = await renderAt('/raffles')
    await flush()
    const row = container.querySelector('tbody tr') as HTMLElement
    await act(async () => {
      row.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    })
    expect(container.querySelector('[data-testid="detail"]')?.textContent).toBe('r1')
    cleanupRoot(root, container)
  })
})
