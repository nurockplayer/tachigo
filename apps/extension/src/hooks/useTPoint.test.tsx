// @vitest-environment jsdom

import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { completeTPointTransaction } from '../services/api'
import type { TPointTransaction } from '../types/twitch'
import { useTPoint } from './useTPoint'

vi.mock('../services/api', () => ({
  completeTPointTransaction: vi.fn().mockResolvedValue({}),
}))

const mockedCompleteTPointTransaction = vi.mocked(completeTPointTransaction)

function makeTransaction(initiator: TPointTransaction['initiator']): TPointTransaction {
  return {
    transactionId: 'tx-1',
    product: {
      sku: 'TPOINT100',
      displayName: '100 T-Points',
      cost: { amount: 100, type: 'bits' },
      inDevelopment: false,
    },
    userId: 'viewer-1',
    displayName: 'Viewer',
    initiator,
    transactionReceipt: 'receipt-jwt',
  }
}

describe('useTPoint', () => {
  let onTransactionComplete: ((tx: TPointTransaction) => void) | undefined
  let onTransactionCancelled: (() => void) | undefined
  let usedBitsSkus: string[]

  beforeEach(() => {
    mockedCompleteTPointTransaction.mockClear()
    onTransactionComplete = undefined
    onTransactionCancelled = undefined
    usedBitsSkus = []

    window.Twitch = {
      ext: {
        bits: {
          getProducts: vi.fn(),
          useBits: (sku: string) => {
            usedBitsSkus.push(sku)
          },
          onTransactionComplete: vi.fn((callback) => {
            onTransactionComplete = callback
          }),
          onTransactionCancelled: vi.fn((callback) => {
            onTransactionCancelled = callback
          }),
        },
        onAuthorized: vi.fn(),
        onContext: vi.fn(),
      },
    }
  })

  it('does not submit receipt for transactions initiated by another viewer', async () => {
    const { result } = renderHook(() => useTPoint('extension-jwt'))

    act(() => {
      result.current.buyWithTPoint('TPOINT100')
    })

    expect(usedBitsSkus).toEqual(['TPOINT100'])
    expect(onTransactionComplete).toBeDefined()

    await act(async () => {
      onTransactionComplete?.(makeTransaction('other'))
    })

    expect(mockedCompleteTPointTransaction).not.toHaveBeenCalled()
    expect(result.current.status).toBe('pending')
  })

  it('submits receipt for transactions initiated by the current viewer', async () => {
    const { result } = renderHook(() => useTPoint('extension-jwt'))

    act(() => {
      result.current.buyWithTPoint('TPOINT100')
    })

    await act(async () => {
      onTransactionComplete?.(makeTransaction('current_user'))
    })

    await waitFor(() => {
      expect(result.current.status).toBe('success')
    })
    expect(mockedCompleteTPointTransaction).toHaveBeenCalledWith('extension-jwt', 'receipt-jwt', 'TPOINT100')
  })

  it('returns to idle when the transaction is cancelled', () => {
    const { result } = renderHook(() => useTPoint('extension-jwt'))

    act(() => {
      result.current.buyWithTPoint('TPOINT100')
    })
    expect(result.current.status).toBe('pending')

    act(() => {
      onTransactionCancelled?.()
    })

    expect(result.current.status).toBe('idle')
  })
})
