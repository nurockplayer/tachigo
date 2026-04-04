import { useCallback, useEffect, useRef, useState } from 'react'
import { sendClick } from '../services/api'

interface UseClickOptions {
  enabled?: boolean
  channelId: string
}

interface UseClickResult {
  cooldownMs: number        // remaining cooldown in ms (0 = ready)
  isReady: boolean
  lastGain: number | null
  onClick: () => void
}

export function useClick(options: UseClickOptions): UseClickResult {
  const { enabled = true, channelId } = options

  const [cooldownMs, setCooldownMs] = useState(0)
  const [lastGain, setLastGain] = useState<number | null>(null)
  const tickRef = useRef<number | null>(null)
  const pendingRef = useRef(false)

  // Tick down the cooldown counter every 100 ms
  useEffect(() => {
    if (cooldownMs <= 0) return
    tickRef.current = window.setTimeout(() => {
      setCooldownMs((prev) => Math.max(0, prev - 100))
    }, 100)
    return () => {
      if (tickRef.current !== null) window.clearTimeout(tickRef.current)
    }
  }, [cooldownMs])

  const onClick = useCallback(() => {
    if (!enabled || !channelId || cooldownMs > 0 || pendingRef.current) return
    pendingRef.current = true

    sendClick(channelId)
      .then((res) => {
        if (res.pointsEarned > 0) {
          setLastGain(res.pointsEarned)
          // Clear gain label after 1.5 s
          window.setTimeout(() => setLastGain(null), 1500)
        }
        // Apply server cooldown (always 10 s) so UI stays in sync
        setCooldownMs(10_000)
      })
      .catch(() => {
        // On network error still apply a short client-side cooldown to avoid spam
        setCooldownMs(2_000)
      })
      .finally(() => {
        pendingRef.current = false
      })
  }, [enabled, channelId, cooldownMs])

  return {
    cooldownMs,
    isReady: cooldownMs === 0,
    lastGain,
    onClick,
  }
}
