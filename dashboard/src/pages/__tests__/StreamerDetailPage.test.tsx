import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StreamerDetailPage from '@/pages/StreamerDetailPage'

const navigateMock = vi.fn()
const usePermissionsMock = vi.fn()
const getChannelStatsMock = vi.fn()
const getChannelConfigMock = vi.fn()

;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true

vi.mock('react-router', async () => {
  const actual = await vi.importActual<typeof import('react-router')>('react-router')
  return {
    ...actual,
    useNavigate: () => navigateMock,
    useParams: () => ({ streamerId: 'channel-1' }),
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
  getChannelStats: (...args: unknown[]) => getChannelStatsMock(...args),
  getChannelConfig: (...args: unknown[]) => getChannelConfigMock(...args),
}))

async function renderPage() {
  const { MemoryRouter } = await import('react-router')
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <MemoryRouter>
        <StreamerDetailPage />
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

describe('StreamerDetailPage', () => {
  beforeEach(() => {
    navigateMock.mockReset()
    usePermissionsMock.mockReset()
    getChannelStatsMock.mockReset()
    getChannelConfigMock.mockReset()
  })

  it('把秒數資料換算後顯示，且 config 失敗時降級為破折號', async () => {
    usePermissionsMock.mockReturnValue({ data: 'admin' })
    getChannelStatsMock.mockResolvedValue({
      current_session_seconds: 5400,
      daily_seconds: 7200,
      monthly_seconds: 14400,
      yearly_seconds: 28800,
      avg_session_seconds: 600,
      unique_miners: 42,
      total_token_minted: 128,
    })
    getChannelConfigMock.mockRejectedValue(new Error('config unavailable'))

    const { container, unmount } = await renderPage()

    expect(container.textContent).toContain('1.5 小時')
    expect(container.textContent).toContain('10 分')
    expect(container.textContent).toContain('42')
    expect(container.textContent).toContain('128')
    expect(container.textContent).toContain('每分鐘產出')
    expect(container.textContent).toContain('—')
    expect(container.textContent).not.toContain('ID: channel-1')

    const buttons = Array.from(container.querySelectorAll('button'))
    expect(buttons.some((button) => button.textContent === '空投' && !button.hasAttribute('disabled'))).toBe(
      true,
    )
    expect(
      buttons.some((button) => button.textContent === '調整倍率' && !button.hasAttribute('disabled')),
    ).toBe(true)

    await unmount()
  })

  it('streamer 角色隱藏返回列表按鈕', async () => {
    usePermissionsMock.mockReturnValue({ data: 'streamer' })
    getChannelStatsMock.mockResolvedValue({
      current_session_seconds: 0,
      daily_seconds: 0,
      monthly_seconds: 0,
      yearly_seconds: 0,
    })
    getChannelConfigMock.mockResolvedValue({
      channel_id: 'channel-1',
      seconds_per_point: 60,
    })

    const { container, unmount } = await renderPage()

    expect(container.textContent).not.toContain('返回列表')

    await unmount()
  })

  it('stats API 失敗時顯示錯誤訊息', async () => {
    usePermissionsMock.mockReturnValue({ data: 'admin' })
    getChannelStatsMock.mockRejectedValue(new Error('server error'))
    getChannelConfigMock.mockRejectedValue(new Error('config unavailable'))

    const { container, unmount } = await renderPage()

    expect(container.textContent).toContain('無法載入頻道詳細資料')

    await unmount()
  })
})
