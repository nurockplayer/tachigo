import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { MemoryRouter, Route, Routes } from 'react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import StreamersPage from '@/pages/StreamersPage'

const getStreamersMock = vi.fn()
const getCurrentUserRoleMock = vi.fn()

vi.mock('@/services/channels', () => ({
  getStreamers: (...args: unknown[]) => getStreamersMock(...args),
}))

vi.mock('@/services/auth', () => ({
  getCurrentUserRole: () => getCurrentUserRoleMock(),
}))

function RoutedApp() {
  return (
    <Routes>
      <Route path="/streamers" element={<StreamersPage />} />
      <Route path="/streamers/:streamerId" element={<div data-testid="detail-page">detail page</div>} />
    </Routes>
  )
}

async function renderAt(path: string) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <MemoryRouter initialEntries={[path]}>
        <RoutedApp />
      </MemoryRouter>,
    )
  })

  return { container, root }
}

async function flush() {
  await act(async () => {
    await Promise.resolve()
  })
}

function cleanupRoot(root: Root, container: HTMLDivElement) {
  act(() => {
    root.unmount()
  })
  container.remove()
}

describe('StreamersPage', () => {
  beforeEach(() => {
    getStreamersMock.mockReset()
    getCurrentUserRoleMock.mockReset()
    getCurrentUserRoleMock.mockReturnValue('admin')
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('顯示 API 回傳的實況主列表', async () => {
    getStreamersMock.mockResolvedValue([
      { id: 'uuid-1', channel_id: 'channel-1', display_name: 'Alice' },
      { id: 'uuid-2', channel_id: 'channel-2', display_name: 'Bob' },
    ])

    const { container, root } = await renderAt('/streamers')
    await flush()

    expect(container.textContent).toContain('Alice')
    expect(container.textContent).toContain('channel-2')

    cleanupRoot(root, container)
  })

  it('streamer 角色載入後會直接導向自己的詳細頁', async () => {
    getCurrentUserRoleMock.mockReturnValue('streamer')
    getStreamersMock.mockResolvedValue([{ id: 'uuid-1', channel_id: 'channel-1', display_name: 'Alice' }])

    const { container, root } = await renderAt('/streamers')
    await flush()
    await flush()

    expect(container.querySelector('[data-testid="detail-page"]')?.textContent).toContain('detail page')

    cleanupRoot(root, container)
  })

  it('API 失敗時顯示錯誤訊息', async () => {
    getStreamersMock.mockRejectedValue(new Error('boom'))

    const { container, root } = await renderAt('/streamers')
    await flush()

    expect(container.textContent).toContain('無法載入實況主資料')

    cleanupRoot(root, container)
  })

  it('空列表時顯示無資料提示', async () => {
    getStreamersMock.mockResolvedValue([])

    const { container, root } = await renderAt('/streamers')
    await flush()

    expect(container.textContent).toContain('目前沒有可顯示的實況主資料')

    cleanupRoot(root, container)
  })
})
