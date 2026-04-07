import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StreamerDetailPage from '@/pages/StreamerDetailPage'

const navigateMock = vi.fn()
const usePermissionsMock = vi.fn()
const getStreamerStatsMock = vi.fn()
const getChannelConfigMock = vi.fn()

;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true

vi.mock('react-router', async () => {
  const actual = await vi.importActual<typeof import('react-router')>('react-router')
  return {
    ...actual,
    useNavigate: () => navigateMock,
    useParams: () => ({ streamerId: 'uuid-streamer-1' }),
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
  getStreamerStats: (...args: unknown[]) => getStreamerStatsMock(...args),
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
    getStreamerStatsMock.mockReset()
    getChannelConfigMock.mockReset()
  })

  it('把秒數資料與倍率設定換算後顯示', async () => {
    usePermissionsMock.mockReturnValue({ data: 'admin' })
    getStreamerStatsMock.mockResolvedValue({
      stats: {
        current_session_seconds: 5400,
        daily_seconds: 7200,
        monthly_seconds: 14400,
        yearly_seconds: 28800,
        avg_session_seconds: 600,
        unique_miners: 42,
        total_token_minted: 128,
        spendable_in_circulation: 60,
      },
      channelId: 'channel-1',
    })
    getChannelConfigMock.mockResolvedValue({
      channel_id: 'channel-1',
      seconds_per_point: 30,
      multiplier: 2,
    })

    const { container, unmount } = await renderPage()

    expect(container.textContent).toContain('1.5 小時')
    expect(container.textContent).toContain('10 分')
    expect(container.textContent).toContain('42')
    expect(container.textContent).toContain('128')
    expect(container.textContent).toContain('30 秒 / 點')
    expect(container.textContent).toContain('2x')
    expect(container.textContent).toContain('4.0 點')

    const buttons = Array.from(container.querySelectorAll('button'))
    expect(buttons.some((b) => b.textContent === '空投')).toBe(true)
    expect(buttons.some((b) => b.textContent === '調整倍率')).toBe(true)

    await unmount()
  })

  it('streamer 角色隱藏返回列表按鈕', async () => {
    usePermissionsMock.mockReturnValue({ data: 'streamer' })
    getStreamerStatsMock.mockResolvedValue({
      stats: {
        current_session_seconds: 0,
        daily_seconds: 0,
        monthly_seconds: 0,
        yearly_seconds: 0,
        avg_session_seconds: 0,
        unique_miners: 0,
        total_token_minted: 0,
        spendable_in_circulation: 0,
      },
      channelId: 'channel-1',
    })
    getChannelConfigMock.mockResolvedValue({
      channel_id: 'channel-1',
      seconds_per_point: 60,
      multiplier: 1,
    })

    const { container, unmount } = await renderPage()

    expect(container.textContent).not.toContain('返回列表')

    await unmount()
  })

  it('stats API 失敗時顯示錯誤訊息', async () => {
    usePermissionsMock.mockReturnValue({ data: 'admin' })
    getStreamerStatsMock.mockRejectedValue(new Error('server error'))

    const { container, unmount } = await renderPage()

    expect(container.textContent).toContain('無法載入頻道詳細資料')

    await unmount()
  })

  it('config API 失敗時靜默降級為破折號', async () => {
    usePermissionsMock.mockReturnValue({ data: 'admin' })
    getStreamerStatsMock.mockResolvedValue({
      stats: {
        current_session_seconds: 5400,
        daily_seconds: 7200,
        monthly_seconds: 14400,
        yearly_seconds: 28800,
        avg_session_seconds: 600,
        unique_miners: 42,
        total_token_minted: 128,
        spendable_in_circulation: 60,
      },
      channelId: 'channel-1',
    })
    getChannelConfigMock.mockRejectedValue(new Error('config unavailable'))

    const { container, unmount } = await renderPage()

    expect(container.textContent).toContain('1.5 小時')
    expect(container.textContent).not.toContain('無法載入頻道詳細資料')
    expect(container.textContent).toContain('每秒點數基準')
    expect(container.textContent).toContain('目前倍率')
    expect(container.textContent).toContain('每分鐘產出')
    expect(container.textContent).toContain('—')

    await unmount()
  })
})
