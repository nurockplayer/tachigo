import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { MemoryRouter, Route, Routes, useNavigate, useParams } from 'react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { BaseRecord, DataProvider } from '@refinedev/core'
import StreamerDetailPage from '@/pages/StreamerDetailPage'
import { createMockDataProvider, RefineWrapper, waitFor } from '@/test/refine-wrapper'

const getUserRoleMock = vi.fn()

vi.mock('@/services/auth', () => ({
  getUserRole: () => getUserRoleMock(),
}))

function RoutedApp() {
  return (
    <Routes>
      <Route path="/streamers/:streamerId" element={<StreamerDetailPage />} />
      <Route path="/streamers" element={<div data-testid="list-page">list page</div>} />
    </Routes>
  )
}

function DetailRouteWithNavigation() {
  const navigate = useNavigate()
  const { streamerId } = useParams()

  return (
    <>
      <button onClick={() => navigate('/streamers/uuid-2')}>go-second</button>
      <StreamerDetailPage key={streamerId} />
    </>
  )
}

function NavigableRoutedApp() {
  return (
    <Routes>
      <Route path="/streamers/:streamerId" element={<DetailRouteWithNavigation />} />
      <Route path="/streamers" element={<div data-testid="list-page">list page</div>} />
    </Routes>
  )
}

async function renderAt(path: string, dataProvider: DataProvider) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <RefineWrapper dataProvider={dataProvider}>
        <MemoryRouter initialEntries={[path]}>
          <RoutedApp />
        </MemoryRouter>
      </RefineWrapper>,
    )
  })

  return { container, root }
}

async function renderNavigableAt(path: string, dataProvider: DataProvider) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <RefineWrapper dataProvider={dataProvider}>
        <MemoryRouter initialEntries={[path]}>
          <NavigableRoutedApp />
        </MemoryRouter>
      </RefineWrapper>,
    )
  })

  return { container, root }
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

function defaultDataProvider() {
  return createMockDataProvider({
    getOne: {
      'streamer-stats': vi.fn().mockResolvedValue({ ...defaultStats, channel_id: 'channel-1' } as BaseRecord),
      'channel-configs': vi.fn().mockResolvedValue(defaultConfig as BaseRecord),
    },
  })
}

describe('StreamerDetailPage', () => {
  beforeEach(() => {
    getUserRoleMock.mockReset()
    getUserRoleMock.mockReturnValue('agency')
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('顯示秒數換算與倍率資訊，並顯示 action 按鈕', async () => {
    const { container, root } = await renderAt('/streamers/uuid-1', defaultDataProvider())
    await waitFor(() => expect(container.textContent).toContain('1.5 小時'))
    expect(container.textContent).toContain('10 分')
    expect(container.textContent).toContain('30 秒 / 點')
    expect(container.textContent).toContain('2x')
    expect(container.textContent).toContain('4.0 點')
    expect(container.textContent).toContain('空投')
    expect(container.textContent).toContain('調整倍率')

    cleanupRoot(root, container)
  })

  it('Agency 顯示返回列表按鈕', async () => {
    getUserRoleMock.mockReturnValue('agency')
    const { container, root } = await renderAt('/streamers/uuid-1', defaultDataProvider())
    await waitFor(() => expect(container.textContent).toContain('返回列表'))

    cleanupRoot(root, container)
  })

  it('Admin 顯示返回列表按鈕', async () => {
    getUserRoleMock.mockReturnValue('admin')
    const { container, root } = await renderAt('/streamers/uuid-1', defaultDataProvider())
    await waitFor(() => expect(container.textContent).toContain('返回列表'))

    cleanupRoot(root, container)
  })

  it('Streamer 不顯示返回列表按鈕', async () => {
    getUserRoleMock.mockReturnValue('streamer')
    const { container, root } = await renderAt('/streamers/uuid-1', defaultDataProvider())
    await waitFor(() => expect(container.textContent).toContain('1.5 小時'))
    expect(container.textContent).not.toContain('返回列表')

    cleanupRoot(root, container)
  })

  it('stats API 失敗時顯示整頁錯誤訊息', async () => {
    const dataProvider = createMockDataProvider({
      getOne: {
        'streamer-stats': vi.fn().mockRejectedValue(new Error('boom')),
        'channel-configs': vi.fn().mockResolvedValue(defaultConfig as BaseRecord),
      },
    })

    const { container, root } = await renderAt('/streamers/uuid-1', dataProvider)
    await waitFor(() => expect(container.textContent).toContain('無法載入頻道詳細資料'))

    cleanupRoot(root, container)
  })

  it('config API 失敗時仍顯示 stats，倍率區塊以破折號降級', async () => {
    const dataProvider = createMockDataProvider({
      getOne: {
        'streamer-stats': vi.fn().mockResolvedValue({ ...defaultStats, channel_id: 'channel-1' } as BaseRecord),
        'channel-configs': vi.fn().mockRejectedValue(new Error('config unavailable')),
      },
    })

    const { container, root } = await renderAt('/streamers/uuid-1', dataProvider)
    await waitFor(() => expect(container.textContent).toContain('1.5 小時'))
    expect(container.textContent).toContain('挖礦倍率設定')
    expect(container.textContent).toContain('—')

    cleanupRoot(root, container)
  })

  it('stats 成功後立即顯示主內容，不等 config', async () => {
    let resolveConfig: ((value: BaseRecord) => void) | null = null
    const dataProvider = createMockDataProvider({
      getOne: {
        'streamer-stats': vi.fn().mockResolvedValue({ ...defaultStats, channel_id: 'channel-1' } as BaseRecord),
        'channel-configs': vi.fn().mockImplementation(
          () => new Promise((resolve) => { resolveConfig = resolve }),
        ),
      },
    })

    const { container, root } = await renderAt('/streamers/uuid-1', dataProvider)
    await waitFor(() => expect(container.textContent).toContain('1.5 小時'))
    expect(container.textContent).toContain('挖礦倍率設定')

    await act(async () => {
      resolveConfig?.(defaultConfig as BaseRecord)
    })
    await waitFor(() => expect(container.textContent).toContain('30 秒 / 點'))

    cleanupRoot(root, container)
  })

  it('streamerId 變更時會先清掉舊資料並回到 loading 狀態', async () => {
    let resolveSecond: ((value: BaseRecord) => void) | null = null

    const statsMock = vi.fn().mockImplementation((id: string | number) => {
      if (id === 'uuid-1') {
        return Promise.resolve({ ...defaultStats, channel_id: 'channel-1' } as BaseRecord)
      }

      return new Promise<BaseRecord>((resolve) => {
        resolveSecond = resolve
      })
    })

    const dataProvider = createMockDataProvider({
      getOne: {
        'streamer-stats': statsMock,
        'channel-configs': vi.fn().mockResolvedValue(defaultConfig as BaseRecord),
      },
    })

    const { container, root } = await renderNavigableAt('/streamers/uuid-1', dataProvider)
    await waitFor(() => expect(container.textContent).toContain('1.5 小時'))

    const navigateButton = Array.from(container.querySelectorAll('button')).find(
      (button) => button.textContent === 'go-second',
    )
    expect(navigateButton).toBeTruthy()

    await act(async () => {
      navigateButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })

    expect(container.textContent).not.toContain('1.5 小時')
    expect(container.textContent).toContain('調整倍率')

    await act(async () => {
      resolveSecond?.({ ...defaultStats, current_session_seconds: 7200, channel_id: 'channel-2' } as BaseRecord)
    })
    await waitFor(() => expect(container.textContent).toContain('2.0 小時'))

    cleanupRoot(root, container)
  })
})

beforeEach(() => {
  vi.spyOn(console, 'error').mockImplementation(() => {})
})
