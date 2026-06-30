import { useCallback, useEffect, useMemo, useState } from 'react'
import type { RaffleDraw, RafflePrizeTier } from '@/services/raffles'
import { drawFromTier, listDraws } from '@/services/raffles'

type SessionPhase = 'round_ready' | 'drawing' | 'round_result' | 'session_complete'

interface Props {
  raffleId: string
  tiers: RafflePrizeTier[]
}

const glass: React.CSSProperties = {
  background: 'rgba(4,14,52,.55)',
  border: '1px solid rgba(80,160,255,.22)',
  borderRadius: 14,
  backdropFilter: 'blur(14px)',
  WebkitBackdropFilter: 'blur(14px)',
}

function FinalWinnerList({ tiers, draws }: { tiers: RafflePrizeTier[]; draws: RaffleDraw[] }) {
  return (
    <div data-testid="final-winner-list" style={{ textAlign: 'left' }}>
      {tiers.map(tier => {
        const tierDraws = draws.filter(d => d.prize_tier_id === tier.id)
        return (
          <div key={tier.id} data-testid={`final-winner-tier-${tier.id}`} style={{ marginBottom: 16 }}>
            <p style={{ fontSize: 13, fontWeight: 700, color: '#7dd3fc', marginBottom: 6 }}>{tier.name}</p>
            {tierDraws.map(draw => (
              <p key={draw.id} style={{ fontSize: 14, color: '#fde68a', marginBottom: 4 }}>
                {draw.entry.display_name || draw.entry.twitch_login}
              </p>
            ))}
          </div>
        )
      })}
    </div>
  )
}

export function RaffleDrawSession({ raffleId, tiers }: Props) {
  const sorted = useMemo(() => [...tiers].sort((a, b) => a.position - b.position), [tiers])
  const firstPendingIndex = useMemo(
    () => sorted.findIndex(t => t.drawn_count < t.winner_count),
    [sorted],
  )
  const [currentIndex, setCurrentIndex] = useState(firstPendingIndex === -1 ? 0 : firstPendingIndex)
  const [phase, setPhase] = useState<SessionPhase>(firstPendingIndex === -1 ? 'session_complete' : 'round_ready')
  const [latestDraw, setLatestDraw] = useState<RaffleDraw | null>(null)
  const [localDrawnCounts, setLocalDrawnCounts] = useState<Record<string, number>>({})
  const [error, setError] = useState<string | null>(null)
  const [allDraws, setAllDraws] = useState<RaffleDraw[]>([])
  const [winnerListStatus, setWinnerListStatus] = useState<'loading' | 'ready' | 'error'>('loading')

  const currentTier = sorted[currentIndex]

  async function handleDraw() {
    if (!currentTier || phase === 'drawing') return
    setPhase('drawing')
    setError(null)
    try {
      const draw = await drawFromTier(raffleId, currentTier.id)
      setLatestDraw(draw)
      setLocalDrawnCounts(prev => ({
        ...prev,
        [currentTier.id]: (prev[currentTier.id] ?? 0) + 1,
      }))
      setPhase('round_result')
    } catch {
      setError('抽獎失敗，請再試一次')
      setPhase('round_ready')
    }
  }

  function handleNext() {
    const localDrawn = localDrawnCounts[currentTier.id] ?? 0
    const needed = currentTier.winner_count
    if (localDrawn + currentTier.drawn_count < needed) {
      setPhase('round_ready')
      return
    }
    let nextIndex = currentIndex + 1
    while (
      nextIndex < sorted.length &&
      sorted[nextIndex].drawn_count + (localDrawnCounts[sorted[nextIndex].id] ?? 0) >= sorted[nextIndex].winner_count
    ) {
      nextIndex += 1
    }
    if (nextIndex >= sorted.length) {
      setPhase('session_complete')
    } else {
      setCurrentIndex(nextIndex)
      setPhase('round_ready')
    }
  }

  const fetchFinalDraws = useCallback(async () => {
    setWinnerListStatus('loading')
    try {
      const draws = await listDraws(raffleId)
      setAllDraws(draws)
      setWinnerListStatus('ready')
    } catch {
      setWinnerListStatus('error')
    }
  }, [raffleId])

  useEffect(() => {
    if (phase !== 'session_complete') return
    const id = window.setTimeout(() => {
      void fetchFinalDraws()
    }, 0)
    return () => window.clearTimeout(id)
  }, [phase, fetchFinalDraws])

  if (phase === 'session_complete') {
    return (
      <div data-testid="session-complete" style={{ ...glass, padding: '24px 28px', textAlign: 'center' }}>
        <p style={{ fontSize: 16, fontWeight: 700, color: '#86efac', marginBottom: 16 }}>🎉 所有輪次抽獎完成</p>
        {winnerListStatus === 'loading' && (
          <p data-testid="winner-list-loading" style={{ fontSize: 13, color: 'rgba(148,210,255,.6)' }}>載入得獎名單中...</p>
        )}
        {winnerListStatus === 'error' && (
          <div data-testid="winner-list-error">
            <p style={{ fontSize: 13, color: '#f87171', marginBottom: 10 }}>得獎名單載入失敗</p>
            <button
              data-testid="retry-winner-list-button"
              onClick={() => { void fetchFinalDraws() }}
              style={{ background: 'rgba(248,113,113,.15)', border: '1px solid rgba(248,113,113,.3)', color: '#f87171', borderRadius: 8, padding: '8px 20px', fontSize: 13, fontWeight: 700, cursor: 'pointer' }}
            >
              重試
            </button>
          </div>
        )}
        {winnerListStatus === 'ready' && <FinalWinnerList tiers={sorted} draws={allDraws} />}
      </div>
    )
  }

  if (!currentTier) return null

  const totalRounds = sorted.length

  return (
    <div data-testid="raffle-draw-session" style={{ ...glass, padding: '20px 24px' }}>
      <p data-testid="round-progress" style={{ fontSize: 11, color: 'rgba(148,210,255,.5)', marginBottom: 12 }}>
        第 {currentIndex + 1} / {totalRounds} 輪
      </p>

      <p data-testid="current-tier-name" style={{ fontSize: 20, fontWeight: 800, color: '#e0f2fe', marginBottom: 4 }}>
        {currentTier.name}
      </p>
      <p data-testid="current-tier-description" style={{ fontSize: 13, color: 'rgba(148,210,255,.6)', marginBottom: 16 }}>
        {currentTier.prize_description}
      </p>
      <p style={{ fontSize: 11, color: 'rgba(148,210,255,.4)', marginBottom: 16 }}>
        {(localDrawnCounts[currentTier.id] ?? 0) + currentTier.drawn_count} / {currentTier.winner_count} 人已抽出
      </p>

      {phase === 'round_ready' && (
        <button
          data-testid="draw-round-button"
          onClick={() => { void handleDraw() }}
          style={{ background: 'rgba(14,165,233,.2)', border: '1px solid rgba(14,165,233,.4)', color: '#38bdf8', borderRadius: 8, padding: '10px 28px', fontSize: 14, fontWeight: 700, cursor: 'pointer' }}
        >
          抽這一輪
        </button>
      )}

      {phase === 'drawing' && (
        <p data-testid="drawing-indicator" style={{ fontSize: 14, color: '#7dd3fc' }}>抽獎中...</p>
      )}

      {phase === 'round_result' && latestDraw && (
        <div data-testid="round-result">
          <p style={{ fontSize: 13, color: 'rgba(148,210,255,.5)', marginBottom: 6 }}>得獎者</p>
          <p data-testid="winner-name" style={{ fontSize: 24, fontWeight: 800, color: '#fde68a', marginBottom: 20 }}>
            {latestDraw.entry.display_name || latestDraw.entry.twitch_login}
          </p>
          <button
            data-testid="next-round-button"
            onClick={handleNext}
            style={{ background: 'rgba(34,197,94,.15)', border: '1px solid rgba(34,197,94,.3)', color: '#4ade80', borderRadius: 8, padding: '10px 24px', fontSize: 13, fontWeight: 700, cursor: 'pointer' }}
          >
            {(() => {
              const localDrawn = localDrawnCounts[currentTier.id] ?? 0
              if (localDrawn + currentTier.drawn_count < currentTier.winner_count) return '繼續抽本輪'
              if (currentIndex + 1 >= totalRounds) return '完成抽獎'
              return '繼續下一輪'
            })()}
          </button>
        </div>
      )}

      {error && <p style={{ fontSize: 12, color: '#f87171', marginTop: 10 }}>{error}</p>}
    </div>
  )
}
