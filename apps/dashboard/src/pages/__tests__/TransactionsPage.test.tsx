import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { MemoryRouter } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { BaseRecord, DataProvider } from '@refinedev/core'
import TransactionsPage from '@/pages/TransactionsPage'
import { createMockDataProvider, RefineWrapper, waitFor } from '@/test/refine-wrapper'

async function renderPage(dataProvider: DataProvider) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <RefineWrapper dataProvider={dataProvider}>
        <MemoryRouter>
          <TransactionsPage />
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

describe('TransactionsPage', () => {
  afterEach(() => { document.body.innerHTML = '' })

  it('loads point history with the first available channel id', async () => {
    const transactionsMock = vi.fn().mockResolvedValue([
      {
        id: 'tx-1',
        type: 'watch',
        amount: 25,
        note: 'watch reward',
        created_at: '2026-01-01T00:00:00Z',
      },
    ] satisfies BaseRecord[])
    const dataProvider = createMockDataProvider({
      getList: {
        'streamer-channels': vi.fn().mockResolvedValue([
          { id: 'streamer-1', channel_id: 'channel-1', display_name: 'Alice' },
        ] satisfies BaseRecord[]),
        'transactions': transactionsMock,
      },
    })

    const { container, root } = await renderPage(dataProvider)
    await waitFor(() => expect(container.textContent).toContain('watch reward'))

    expect(transactionsMock).toHaveBeenCalledWith(expect.objectContaining({
      meta: expect.objectContaining({
        params: { channel_id: 'channel-1' },
      }),
    }))

    cleanupRoot(root, container)
  })

  it('shows a missing marker when transaction amount and delta are absent', async () => {
    const dataProvider = createMockDataProvider({
      getList: {
        'streamer-channels': vi.fn().mockResolvedValue([
          { id: 'streamer-1', channel_id: 'channel-1', display_name: 'Alice' },
        ] satisfies BaseRecord[]),
        'transactions': vi.fn().mockResolvedValue([
          {
            id: 'tx-1',
            type: 'adjustment',
            note: 'missing amount',
            created_at: '2026-01-01T00:00:00Z',
          },
        ] satisfies BaseRecord[]),
      },
    })

    const { container, root } = await renderPage(dataProvider)
    await waitFor(() => expect(container.textContent).toContain('missing amount'))

    const amountCell = container.querySelector('tbody tr td:nth-child(2)')
    expect(amountCell?.textContent).toBe('—')

    cleanupRoot(root, container)
  })
})
