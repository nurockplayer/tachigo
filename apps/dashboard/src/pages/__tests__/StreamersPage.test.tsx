import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { MemoryRouter, Route, Routes, useParams } from 'react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { DataProvider } from '@refinedev/core'
import StreamersPage from '@/pages/StreamersPage'
import { createMockDataProvider, flushAsync, RefineWrapper, waitFor } from '@/test/refine-wrapper'

function DetailRouteProbe() {
  const { streamerId } = useParams()
  return <div data-testid="detail-page">{streamerId}</div>
}

function RoutedApp() {
  return (
    <Routes>
      <Route path="/streamers" element={<StreamersPage />} />
      <Route path="/streamers/:streamerId" element={<DetailRouteProbe />} />
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

function cleanupRoot(root: Root, container: HTMLDivElement) {
  act(() => { root.unmount() })
  container.remove()
}

function axiosError(status: number) {
  return { isAxiosError: true, response: { status } }
}

describe('StreamersPage', () => {
  afterEach(() => { document.body.innerHTML = '' })

  it('renders streamers from the API response', async () => {
    const dataProvider = createMockDataProvider({
      getList: {
        'streamer-channels': vi.fn().mockRejectedValue(axiosError(401)),
        'streamers': vi.fn().mockResolvedValue([
          { id: 'uuid-1', channel_id: 'channel-1', display_name: 'Alice' },
          { id: 'uuid-2', channel_id: 'channel-2', display_name: 'Bob' },
        ]),
      },
    })

    const { container, root } = await renderAt('/streamers', dataProvider)
    await waitFor(() => expect(container.textContent).toContain('Alice'))
    expect(container.textContent).toContain('Bob')

    cleanupRoot(root, container)
  })

  it('顯示 summary metrics 欄位（本日開台、挖礦觀眾、總產出點數）', async () => {
    const dataProvider = createMockDataProvider({
      getList: {
        'streamer-channels': vi.fn().mockRejectedValue(axiosError(401)),
        'streamers': vi.fn().mockResolvedValue([
          {
            id: 'uuid-1',
            channel_id: 'channel-1',
            display_name: 'Alice',
            daily_seconds: 12600,
            unique_miners: 1240,
            total_token_minted: 3200,
          },
        ]),
      },
    })

    const { container, root } = await renderAt('/streamers', dataProvider)
    await waitFor(() => expect(container.textContent).toContain('3.5 hr'))
    expect(container.textContent).toContain('1,240')
    expect(container.textContent).toContain('3,200')

    cleanupRoot(root, container)
  })

  it('summary metrics 為 undefined 時顯示破折號', async () => {
    const dataProvider = createMockDataProvider({
      getList: {
        'streamer-channels': vi.fn().mockRejectedValue(axiosError(401)),
        'streamers': vi.fn().mockResolvedValue([
          { id: 'uuid-1', channel_id: 'channel-1', display_name: 'Alice' },
        ]),
      },
    })

    const { container, root } = await renderAt('/streamers', dataProvider)
    await waitFor(() => expect(container.textContent).toContain('Alice'))

    const dashes = (container.textContent?.match(/—/g) ?? []).length
    expect(dashes).toBeGreaterThanOrEqual(3)

    cleanupRoot(root, container)
  })

  it('opens the detail page when pressing Enter on a row', async () => {
    const dataProvider = createMockDataProvider({
      getList: {
        'streamer-channels': vi.fn().mockRejectedValue(axiosError(401)),
        'streamers': vi.fn().mockResolvedValue([
          { id: 'uuid-1', channel_id: 'channel-1', display_name: 'Alice' },
          { id: 'uuid-2', channel_id: 'channel-2', display_name: 'Bob' },
        ]),
      },
    })

    const { container, root } = await renderAt('/streamers', dataProvider)
    await waitFor(() => expect(container.querySelector('tbody tr')).toBeTruthy())

    const firstRow = container.querySelector('tbody tr')!
    act(() => {
      firstRow.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    })

    expect(container.querySelector('[data-testid="detail-page"]')?.textContent).toBe('uuid-1')

    cleanupRoot(root, container)
  })

  it('redirects a streamer to the first available channel', async () => {
    const channelsMock = vi.fn().mockResolvedValue([
      { id: 'uuid-1', user_id: 'user-1', channel_id: 'channel-1', display_name: 'Alice' },
      { id: 'uuid-2', user_id: 'user-2', channel_id: 'channel-2', display_name: 'Bob' },
    ])
    const streamersMock = vi.fn()
    const dataProvider = createMockDataProvider({
      getList: {
        'streamer-channels': channelsMock,
        'streamers': streamersMock,
      },
    })

    const { container, root } = await renderAt('/streamers', dataProvider)
    await waitFor(() =>
      expect(container.querySelector('[data-testid="detail-page"]')?.textContent).toBe('uuid-1'),
    )

    expect(streamersMock).not.toHaveBeenCalled()
    expect(channelsMock).toHaveBeenCalledTimes(1)

    cleanupRoot(root, container)
  })

  it('keeps streamer on the listing page when no channel exists', async () => {
    const channelsMock = vi.fn().mockResolvedValue([])
    const streamersMock = vi.fn()
    const dataProvider = createMockDataProvider({
      getList: {
        'streamer-channels': channelsMock,
        'streamers': streamersMock,
      },
    })

    const { container, root } = await renderAt('/streamers', dataProvider)
    await waitFor(() => expect(container.textContent).toContain('尚無'))

    expect(streamersMock).not.toHaveBeenCalled()
    expect(channelsMock).toHaveBeenCalledTimes(1)
    expect(container.querySelector('[data-testid="detail-page"]')).toBeNull()

    cleanupRoot(root, container)
  })

  it('shows an error message when the API request fails', async () => {
    const dataProvider = createMockDataProvider({
      getList: {
        'streamer-channels': vi.fn().mockRejectedValue(axiosError(401)),
        'streamers': vi.fn().mockRejectedValue(new Error('boom')),
      },
    })

    const { container, root } = await renderAt('/streamers', dataProvider)
    await waitFor(() => expect(container.textContent).toContain('無法'))

    cleanupRoot(root, container)
  })

  it('shows an empty state when the list is empty', async () => {
    const dataProvider = createMockDataProvider({
      getList: {
        'streamer-channels': vi.fn().mockRejectedValue(axiosError(401)),
        'streamers': vi.fn().mockResolvedValue([]),
      },
    })

    const { container, root } = await renderAt('/streamers', dataProvider)
    await waitFor(() => expect(container.textContent).toContain('尚無'))

    cleanupRoot(root, container)
  })

  it('falls back to getStreamers only when getMyChannels returns 401', async () => {
    const channelsMock = vi.fn().mockRejectedValue(axiosError(401))
    const streamersMock = vi.fn().mockResolvedValue([
      { id: 'uuid-1', channel_id: 'channel-1', display_name: 'Alice' },
    ])
    const dataProvider = createMockDataProvider({
      getList: {
        'streamer-channels': channelsMock,
        'streamers': streamersMock,
      },
    })

    const { container, root } = await renderAt('/streamers', dataProvider)
    await waitFor(() => expect(container.textContent).toContain('Alice'))

    expect(channelsMock).toHaveBeenCalledTimes(1)
    expect(streamersMock).toHaveBeenCalledTimes(1)

    cleanupRoot(root, container)
  })

  it('falls back to getStreamers when getMyChannels returns 403', async () => {
    const channelsMock = vi.fn().mockRejectedValue(axiosError(403))
    const streamersMock = vi.fn().mockResolvedValue([
      { id: 'uuid-1', channel_id: 'channel-1', display_name: 'Alice' },
    ])
    const dataProvider = createMockDataProvider({
      getList: {
        'streamer-channels': channelsMock,
        'streamers': streamersMock,
      },
    })

    const { container, root } = await renderAt('/streamers', dataProvider)
    await waitFor(() => expect(container.textContent).toContain('Alice'))

    expect(channelsMock).toHaveBeenCalledTimes(1)
    expect(streamersMock).toHaveBeenCalledTimes(1)

    cleanupRoot(root, container)
  })

  it('shows an error state when getMyChannels returns 500', async () => {
    const channelsMock = vi.fn().mockRejectedValue(axiosError(500))
    const streamersMock = vi.fn()
    const dataProvider = createMockDataProvider({
      getList: {
        'streamer-channels': channelsMock,
        'streamers': streamersMock,
      },
    })

    const { container, root } = await renderAt('/streamers', dataProvider)
    await waitFor(() => expect(container.textContent).toContain('無法載入直播主資料'))

    expect(streamersMock).not.toHaveBeenCalled()

    cleanupRoot(root, container)
  })

  it('shows an error state when fallback getStreamers also fails after a 401', async () => {
    const streamersMock = vi.fn().mockRejectedValue(new Error('fallback failed'))
    const dataProvider = createMockDataProvider({
      getList: {
        'streamer-channels': vi.fn().mockRejectedValue(axiosError(401)),
        'streamers': streamersMock,
      },
    })

    const { container, root } = await renderAt('/streamers', dataProvider)
    await waitFor(() => expect(container.textContent).toContain('無法載入直播主資料'))

    expect(streamersMock).toHaveBeenCalledTimes(1)

    cleanupRoot(root, container)
  })
})

beforeEach(() => {
  vi.spyOn(console, 'error').mockImplementation(() => {})
})
