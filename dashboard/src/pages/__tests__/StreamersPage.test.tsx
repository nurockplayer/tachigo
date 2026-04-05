import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StreamersPage from '@/pages/StreamersPage'

const navigateMock = vi.fn()
const getStreamerChannelsMock = vi.fn()
const usePermissionsMock = vi.fn()

;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true

vi.mock('react-router', async () => {
  const actual = await vi.importActual<typeof import('react-router')>('react-router')
  return {
    ...actual,
    useNavigate: () => navigateMock,
  }
})

vi.mock('@refinedev/core', async () => {
  const actual = await vi.importActual<typeof import('@refinedev/core')>('@refinedev/core')
  return {
    ...actual,
    usePermissions: () => usePermissionsMock(),
  }
})

vi.mock('@/services/channels', () => ({
  getStreamerChannels: (...args: unknown[]) => getStreamerChannelsMock(...args),
}))

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
    getStreamerChannelsMock.mockReset()
    usePermissionsMock.mockReset()
  })

  it('載入成功時顯示 API 資料，缺少的統計欄位顯示破折號', async () => {
    usePermissionsMock.mockReturnValue({ data: 'admin' })
    getStreamerChannelsMock.mockResolvedValue([
      {
        id: '1',
        channel_id: 'channel-1',
        display_name: 'Nurock',
      },
    ])

    const { container, unmount } = await renderPage()

    expect(container.textContent).toContain('Nurock')
    expect(container.textContent).toContain('—')
    expect(container.textContent).not.toContain('示範資料')

    await unmount()
  })

  it('streamer 角色在載入完成後自動跳轉到自己的詳細頁', async () => {
    usePermissionsMock.mockReturnValue({ data: 'streamer' })
    getStreamerChannelsMock.mockResolvedValue([
      {
        id: '1',
        channel_id: 'channel-1',
        display_name: 'Nurock',
      },
    ])

    const { unmount } = await renderPage()

    expect(navigateMock).toHaveBeenCalledWith('/streamers/channel-1', { replace: true })

    await unmount()
  })

  it('API 失敗時顯示錯誤訊息', async () => {
    usePermissionsMock.mockReturnValue({ data: 'admin' })
    getStreamerChannelsMock.mockRejectedValue(new Error('boom'))

    const { container, unmount } = await renderPage()

    expect(container.textContent).toContain('無法載入實況主資料')

    await unmount()
  })

  it('API 回傳空陣列時顯示無資料訊息', async () => {
    usePermissionsMock.mockReturnValue({ data: 'admin' })
    getStreamerChannelsMock.mockResolvedValue([])

    const { container, unmount } = await renderPage()

    expect(container.textContent).toContain('目前沒有可顯示的實況主資料')

    await unmount()
  })
})
