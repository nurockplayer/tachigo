import { useEffect, useState } from 'react'
import { getRaffleResult } from '../services/api'
import type { RaffleResultDraw } from '../extension/types'

const POLL_INTERVAL_MS = 5_000

export function useRaffleResult(raffleId: string | null) {
  const [draws, setDraws] = useState<RaffleResultDraw[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!raffleId) {
      setDraws([])
      setLoading(false)
      setError(null)
      return
    }

    let isDisposed = false
    let timer: number | null = null

    const fetchOnce = async () => {
      try {
        const result = await getRaffleResult(raffleId)
        if (!isDisposed) {
          setDraws(result)
          setError(null)
          setLoading(false)
        }
      } catch {
        if (!isDisposed) {
          setError('Failed to load raffle results')
          setLoading(false)
        }
      }
    }

    const scheduleNext = () => {
      timer = window.setTimeout(() => {
        void pollLoop()
      }, POLL_INTERVAL_MS)
    }

    const pollLoop = async () => {
      await fetchOnce()
      if (!isDisposed) {
        scheduleNext()
      }
    }

    setLoading(true)
    void pollLoop()

    return () => {
      isDisposed = true
      if (timer !== null) {
        window.clearTimeout(timer)
      }
    }
  }, [raffleId])

  return { draws, loading, error }
}
