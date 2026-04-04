import { useCallback, useEffect, useRef, useState } from 'react'
import { sendClick } from '../services/api'

const OPTIMISTIC_COOLDOWN_MS = 5_000
const ANIM_DURATION_MS = 1_500
const COUNTDOWN_INTERVAL_MS = 100

export function useClickBoost(channelId: string | undefined, enabled: boolean) {
  const [cooldownMs, setCooldownMs] = useState(0)
  const [isAnimating, setIsAnimating] = useState(false)
  const [gain, setGain] = useState<number | null>(null)
  const [balance, setBalance] = useState<number | null>(null)

  const onCooldownRef = useRef(false)
  const cooldownTimerRef = useRef<number | null>(null)
  const animTimerRef = useRef<number | null>(null)

  const stopCooldown = useCallback(() => {
    if (cooldownTimerRef.current !== null) {
      clearInterval(cooldownTimerRef.current)
      cooldownTimerRef.current = null
    }
    onCooldownRef.current = false
    setCooldownMs(0)
  }, [])

  const startCooldown = useCallback(
    (ms: number) => {
      stopCooldown()
      onCooldownRef.current = true
      const endAt = Date.now() + ms
      setCooldownMs(ms)
      cooldownTimerRef.current = window.setInterval(() => {
        const remaining = endAt - Date.now()
        if (remaining <= 0) {
          stopCooldown()
        } else {
          setCooldownMs(remaining)
        }
      }, COUNTDOWN_INTERVAL_MS)
    },
    [stopCooldown],
  )

  const handleClick = useCallback(async () => {
    if (!channelId || !enabled || onCooldownRef.current) return

    // Optimistic: lock UI immediately, don't wait for API round-trip.
    startCooldown(OPTIMISTIC_COOLDOWN_MS)

    try {
      const result = await sendClick(channelId)
      setBalance(result.balance)
      setGain(result.delta)
      setIsAnimating(true)
      if (animTimerRef.current !== null) clearTimeout(animTimerRef.current)
      animTimerRef.current = window.setTimeout(() => {
        setIsAnimating(false)
        setGain(null)
      }, ANIM_DURATION_MS)
    } catch (err: unknown) {
      // If the server returns a precise retry-after, honour it.
      const retryAfterMs = (
        err as { response?: { data?: { retry_after_ms?: number } } }
      )?.response?.data?.retry_after_ms

      if (typeof retryAfterMs === 'number' && retryAfterMs > 0) {
        startCooldown(retryAfterMs)
      } else {
        // Unexpected error — unblock so the viewer can try again.
        stopCooldown()
      }
    }
  }, [channelId, enabled, startCooldown, stopCooldown])

  // Cleanup on unmount.
  useEffect(() => {
    return () => {
      if (cooldownTimerRef.current !== null) clearInterval(cooldownTimerRef.current)
      if (animTimerRef.current !== null) clearTimeout(animTimerRef.current)
    }
  }, [])

  return { handleClick, cooldownMs, isAnimating, gain, balance }
}
