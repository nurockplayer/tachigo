import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { MemoryRouter, Route, Routes } from 'react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import StreamerDetailPage from '@/pages/StreamerDetailPage'

const getStreamerStatsMock = vi.fn()
const getChannelConfigMock = vi.fn()
const getCurrentUserRoleMock = vi.fn()

vi.mock('@/services/channels', () => ({
  getStreamerStats: (...args: unknown[]) => getStreamerStatsMock(...args),
  getChannelConfig: (...args: unknown[]) => getChannelConfigMock(...args),
}))

vi.mock('@/services/auth', () => ({
  getCurrentUserRole: () => getCurrentUserRoleMock(),
}))

function RoutedApp() {
  return (
    <Routes>
      <Route path="/streamers/:streamerId" element={<StreamerDetailPage />} />
      <Route path="/streamers" element={<div data-testid="list-page">list page</div>} />
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

const defaultStats = {
  current_session_seconds: 5400,
  daily_seconds: 10800,
  monthly_seconds: 28800,
  yearly_seconds: 432000,
  unique_miners: 1240,
  avg_session_seconds: 600,
  total_token_minted: 3200,
  spendable_in_circulation: 1800,
}

const defaultConfig = {
  channel_id: 'channel-1',
  seconds_per_point: 30,
  multiplier: 2,
}

describe('StreamerDetailPage', () => {
  beforeEach(() => {
    getStreamerStatsMock.mockReset()
    getChannelConfigMock.mockReset()
    getCurrentUserRoleMock.mockReset()
    getCurrentUserRoleMock.mockReturnValue('admin')
    getStreamerStatsMock.mockResolvedValue({
      stats: defaultStats,
      channelId: 'channel-1',
    })
    getChannelConfigMock.mockResolvedValue(defaultConfig)
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('顯示秒數換算與倍率資訊，並顯示 action 按鈕', async () => {
    const { container, root } = await renderAt('/streamers/uuid-1')
    await flush()
    await flush()

    expect(container.textContent).toContain('1.5 小時')
    expect(container.textContent).toContain('10 分')
    expect(container.textContent).toContain('30 秒 / 點')
    expect(container.textContent).toContain('2x')
    expect(container.textContent).toContain('4.0 點')
    expect(container.textContent).toContain('空投')
    expect(container.textContent).toContain('調整倍率')

    cleanupRoot(root, container)
  })

  it('streamer 角色不顯示返回列表按鈕', async () => {
    getCurrentUserRoleMock.mockReturnValue('streamer')

    const { container, root } = await renderAt('/streamers/uuid-1')
    await flush()
    await flush()

    expect(container.textContent).not.toContain('返回列表')

    cleanupRoot(root, container)
  })

  it('stats API 失敗時顯示整頁錯誤訊息', async () => {
    getStreamerStatsMock.mockRejectedValue(new Error('boom'))

    const { container, root } = await renderAt('/streamers/uuid-1')
    await flush()

    expect(container.textContent).toContain('無法載入頻道詳細資料')

    cleanupRoot(root, container)
  })

  it('config API 失敗時仍顯示 stats，倍率區塊以破折號降級', async () => {
    getChannelConfigMock.mockRejectedValue(new Error('config unavailable'))

    const { container, root } = await renderAt('/streamers/uuid-1')
    await flush()
    await flush()

    expect(container.textContent).toContain('1.5 小時')
    expect(container.textContent).toContain('挖礦倍率設定')
    expect(container.textContent).toContain('—')

    cleanupRoot(root, container)
  })
})
