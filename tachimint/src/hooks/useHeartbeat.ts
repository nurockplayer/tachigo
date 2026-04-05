import { useEffect, useRef, useState } from 'react'
import { sendHeartbeat } from '../services/api'

interface UseHeartbeatOptions {
  enabled?: boolean
  intervalMs?: number
}

export function useHeartbeat(extensionJwt: string, options: UseHeartbeatOptions = {}) {
  const { enabled = true, intervalMs = 30_000 } = options
  const [balance, setBalance] = useState<number | null>(null)
  const [gain, setGain] = useState<number | null>(null)
  const [isAnimating, setIsAnimating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const lastBalanceRef = useRef<number | null>(null)
  const stopAnimationTimerRef = useRef<number | null>(null)

  useEffect(() => {
    if (!enabled || !extensionJwt) return

    let isDisposed = false
    let heartbeatTimer: number | null = null

    const clearAnimationTimer = () => {
      if (stopAnimationTimerRef.current !== null) {
        window.clearTimeout(stopAnimationTimerRef.current)
        stopAnimationTimerRef.current = null
      }
    }

    const runHeartbeat = async () => {
      try {
        const data = await sendHeartbeat(extensionJwt)
        if (isDisposed) return

        const nextBalance = data.balance
        const prevBalance = lastBalanceRef.current
        setBalance(nextBalance)

        if (prevBalance !== null && nextBalance > prevBalance) {
          setGain(nextBalance - prevBalance)
          setIsAnimating(true)
          clearAnimationTimer()
          stopAnimationTimerRef.current = window.setTimeout(() => {
            setIsAnimating(false)
            setGain(null)
          }, 1500)
        }

        lastBalanceRef.current = nextBalance
        setError(null)
      } catch {
        if (!isDisposed) {
          setError('Heartbeat failed')
        }
      }
    }

    void runHeartbeat()
    heartbeatTimer = window.setInterval(() => {
      void runHeartbeat()
    }, intervalMs)

    return () => {
      isDisposed = true
      if (heartbeatTimer !== null) {
        window.clearInterval(heartbeatTimer)
      }
      clearAnimationTimer()
    }
  }, [enabled, extensionJwt, intervalMs])

  // Allow external callers (e.g. click boost) to sync the baseline so the
  // next heartbeat gain animation doesn't double-count already-awarded points.
  const syncBalance = (newBalance: number) => {
    lastBalanceRef.current = newBalance
    setBalance(newBalance)
  }

  return { balance, gain, isAnimating, error, syncBalance }
}
