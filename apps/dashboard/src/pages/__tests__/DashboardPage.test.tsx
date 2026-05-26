import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { MemoryRouter } from 'react-router'
import { afterEach, describe, expect, it } from 'vitest'
import { createMockDataProvider, RefineWrapper, waitFor } from '@/test/refine-wrapper'
import DashboardPage from '@/pages/DashboardPage'

async function renderDashboard(dataProvider: ReturnType<typeof createMockDataProvider>) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  await act(async () => {
    root.render(
      <RefineWrapper dataProvider={dataProvider}>
        <MemoryRouter>
          <DashboardPage />
        </MemoryRouter>
      </RefineWrapper>,
    )
  })
  return { container, root }
}

function cleanup(root: Root, container: HTMLDivElement) {
  act(() => { root.unmount() })
  container.remove()
}

const mockChannel = { id: 'ch1', channel_id: 'ch1', display_name: '測試主播' }
const mockStats = {
  id: 'ch1',
  channel_id: 'ch1',
  monthly_seconds: 153000,   // 42.5 hr
  unique_miners: 1284,
  total_token_minted: 98320,
}
const mockRaffles = [
  { id: 'r1', title: '五月抽獎', status: 'active',    created_at: '2026-05-20T00:00:00Z' },
  { id: 'r2', title: '四月抽獎', status: 'draft',     created_at: '2026-05-15T00:00:00Z' },
  { id: 'r3', title: '三月抽獎', status: 'completed', created_at: '2026-04-30T00:00:00Z' },
]

describe('DashboardPage', () => {
  afterEach(() => { document.body.innerHTML = '' })

  it('顯示載入中 skeleton', async () => {
    const dp = createMockDataProvider({
      getList: {
        'streamer-channels': async () => new Promise(() => {}),
        raffles: async () => new Promise(() => {}),
      },
    })
    const { container, root } = await renderDashboard(dp)
    expect(container.querySelector('[data-testid="stats-skeleton"]')).not.toBeNull()
    cleanup(root, container)
  })

  it('載入完成後顯示統計卡片', async () => {
    const dp = createMockDataProvider({
      getList: {
        'streamer-channels': async () => [mockChannel],
        raffles: async () => mockRaffles,
      },
      getOne: {
        'streamer-stats': async () => mockStats,
      },
    })
    const { container, root } = await renderDashboard(dp)
    await waitFor(() => {
      expect(container.textContent).toContain('42.5')
      expect(container.textContent).toContain('1,284')
      expect(container.textContent).toContain('98,320')
    })
    cleanup(root, container)
  })

  it('顯示最近 3 筆抽獎', async () => {
    const dp = createMockDataProvider({
      getList: {
        'streamer-channels': async () => [mockChannel],
        raffles: async () => mockRaffles,
      },
      getOne: { 'streamer-stats': async () => mockStats },
    })
    const { container, root } = await renderDashboard(dp)
    await waitFor(() => {
      expect(container.textContent).toContain('五月抽獎')
      expect(container.textContent).toContain('四月抽獎')
      expect(container.textContent).toContain('三月抽獎')
    })
    cleanup(root, container)
  })

  it('無 channel 時統計顯示「—」不報錯', async () => {
    const dp = createMockDataProvider({
      getList: {
        'streamer-channels': async () => [],
        raffles: async () => [],
      },
    })
    const { container, root } = await renderDashboard(dp)
    await waitFor(() => {
      const dashes = container.querySelectorAll('[data-testid="stat-empty"]')
      expect(dashes.length).toBeGreaterThanOrEqual(3)
    })
    cleanup(root, container)
  })
})
