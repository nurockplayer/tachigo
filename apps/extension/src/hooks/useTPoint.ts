import { useCallback, useState } from 'react'
import type { TPointTransaction } from '../types/twitch'
import { completeTPointTransaction } from '../services/api'

type Status = 'idle' | 'pending' | 'success' | 'error'

export function useTPoint(jwt: string) {
  const [status, setStatus] = useState<Status>('idle')
  const [error, setError] = useState<string | null>(null)

  const buyWithTPoint = useCallback(
    (sku: string) => {
      const ext = window.Twitch?.ext
      if (!ext?.bits) return

      setStatus('pending')
      setError(null)

      ext.bits.onTransactionComplete(async (tx: TPointTransaction) => {
        try {
          await completeTPointTransaction(jwt, tx.transactionReceipt, tx.product.sku)
          setStatus('success')
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        } catch (err: any) {
          setError(err?.response?.data?.message ?? 'Transaction failed')
          setStatus('error')
        }
      })

      ext.bits.onTransactionCancelled(() => {
        setStatus('idle')
      })

      ext.bits.useBits(sku)
    },
    [jwt],
  )

  return { buyWithTPoint, status, error }
}
