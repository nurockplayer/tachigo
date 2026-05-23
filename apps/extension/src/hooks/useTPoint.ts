import { useCallback, useEffect, useRef, useState } from 'react'
import type { TPointTransaction } from '../types/twitch'
import { completeTPointTransaction } from '../services/api'

type Status = 'idle' | 'pending' | 'success' | 'error'

export function useTPoint(jwt: string) {
  const [status, setStatus] = useState<Status>('idle')
  const [error, setError] = useState<string | null>(null)
  const mountedRef = useRef(true)
  const jwtRef = useRef(jwt)
  const callbacksRegisteredRef = useRef(false)
  const pendingSkuRef = useRef<string | null>(null)

  useEffect(() => {
    jwtRef.current = jwt
  }, [jwt])

  useEffect(() => {
    return () => {
      mountedRef.current = false
    }
  }, [])

  const registerBitsCallbacks = useCallback(() => {
    const ext = window.Twitch?.ext
    if (!ext?.bits) {
      return false
    }

    if (!callbacksRegisteredRef.current) {
      ext.bits.onTransactionComplete(async (tx: TPointTransaction) => {
        const pendingSku = pendingSkuRef.current
        if (tx.initiator !== 'current_user' || pendingSku === null || tx.product.sku !== pendingSku) {
          return
        }

        try {
          await completeTPointTransaction(jwtRef.current, tx.transactionReceipt, pendingSku)
          pendingSkuRef.current = null
          if (!mountedRef.current) {
            return
          }

          setStatus('success')
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        } catch (err: any) {
          pendingSkuRef.current = null
          if (!mountedRef.current) {
            return
          }

          setError(err?.response?.data?.message ?? 'Transaction failed')
          setStatus('error')
        }
      })

      ext.bits.onTransactionCancelled(() => {
        pendingSkuRef.current = null
        if (!mountedRef.current) {
          return
        }

        setStatus('idle')
      })

      callbacksRegisteredRef.current = true
    }

    return true
  }, [])

  const buyWithTPoint = useCallback(
    (sku: string) => {
      const ext = window.Twitch?.ext
      if (!ext?.bits || !registerBitsCallbacks()) return

      pendingSkuRef.current = sku
      setStatus('pending')
      setError(null)
      ext.bits.useBits(sku)
    },
    [registerBitsCallbacks],
  )

  return { buyWithTPoint, status, error }
}
