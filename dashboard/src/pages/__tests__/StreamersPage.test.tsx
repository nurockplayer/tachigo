import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StreamersPage from '@/pages/StreamersPage'

const navigateMock = vi.fn()
const getStreamersMock = vi.fn()
const getCurrentUserRoleMock = vi.fn()

;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true

vi.mock('react-router', async () => {
  const actual = await vi.importActual<typeof import('react-router')>('react-router')
  return {
    ...actual,
    useNavigate: () => navigateMock,
  }
})

vi.mock('@/services/channels', () => ({
  getStreamers: (...args: unknown[]) => getStreamersMock(...args),
}))

vi.mock('@/services/auth', async () => {
  const actual = await vi.importActual<typeof import('@/services/auth')>('@/services/auth')
  return {
    ...actual,
    getCurrentUserRole: () => getCurrentUserRoleMock(),
  }
})

async function renderPage() {
  const { MemoryRouter } = await import('react-router')
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <MemoryRouter>
        <StreamersPage />
      </MemoryRouter>,
    )
  })

  await act(async () => {
    await Promise.resolve()
  })

  return {
    container,
    unmount: async () => {
      await act(async () => {
        root.unmount()
      })
      container.remove()
    },
  }
}

describe('StreamersPage', () => {
  beforeEach(() => {
    navigateMock.mockReset()
    getStreamersMock.mockReset()
    getCurrentUserRoleMock.mockReset()
  })

  it('載入成功時顯示 API 資料', async () => {
    getCurrentUserRoleMock.mockReturnValue('admin')
    getStreamersMock.mockResolvedValue([
      {
        id: 'uuid-1',
        user_id: 'user-1',
        channel_id: 'channel-1',
        display_name: 'Nurock',
      },
    ])

    const { container, unmount } = await renderPage()

    expect(container.textContent).toContain('Nurock')
    expect(container.textContent).toContain('channel-1')

    await unmount()
  })

  it('streamer 角色在載入完成後自動跳轉到自己的詳細頁', async () => {
    getCurrentUserRoleMock.mockReturnValue('streamer')
    getStreamersMock.mockResolvedValue([
      {
        id: 'uuid-1',
        user_id: 'user-1',
        channel_id: 'channel-1',
        display_name: 'Nurock',
      },
    ])

    const { unmount } = await renderPage()

    expect(navigateMock).toHaveBeenCalledWith('/streamers/uuid-1', { replace: true })

    await unmount()
  })

  it('API 失敗時顯示錯誤訊息', async () => {
    getCurrentUserRoleMock.mockReturnValue('admin')
    getStreamersMock.mockRejectedValue(new Error('boom'))

    const { container, unmount } = await renderPage()

    expect(container.textContent).toContain('無法載入實況主資料')

    await unmount()
  })

  it('API 回傳空陣列時顯示無資料訊息', async () => {
    getCurrentUserRoleMock.mockReturnValue('admin')
    getStreamersMock.mockResolvedValue([])

    const { container, unmount } = await renderPage()

    expect(container.textContent).toContain('目前沒有可顯示的實況主資料')

    await unmount()
  })
})
