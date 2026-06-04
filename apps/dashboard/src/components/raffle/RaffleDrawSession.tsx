import { useState } from 'react'
import type { RaffleDraw, RafflePrizeTier } from '@/services/raffles'
import { drawFromTier } from '@/services/raffles'

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

export function RaffleDrawSession({ raffleId, tiers }: Props) {
  const sorted = [...tiers].sort((a, b) => a.position - b.position)
  const [currentIndex, setCurrentIndex] = useState(0)
  const [phase, setPhase] = useState<SessionPhase>('round_ready')
  const [latestDraw, setLatestDraw] = useState<RaffleDraw | null>(null)
  const [error, setError] = useState<string | null>(null)

  const currentTier = sorted[currentIndex]

  async function handleDraw() {
    if (!currentTier) return
    setPhase('drawing')
    setError(null)
    try {
      const draw = await drawFromTier(raffleId, currentTier.id)
      setLatestDraw(draw)
      setPhase('round_result')
    } catch {
      setError('抽獎失敗，請再試一次')
      setPhase('round_ready')
    }
  }

  function handleNext() {
    const nextIndex = currentIndex + 1
    if (nextIndex >= sorted.length) {
      setPhase('session_complete')
    } else {
      setCurrentIndex(nextIndex)
      setPhase('round_ready')
    }
  }

  if (phase === 'session_complete') {
    return (
      <div data-testid="session-complete" style={{ ...glass, padding: '24px 28px', textAlign: 'center' }}>
        <p style={{ fontSize: 16, fontWeight: 700, color: '#86efac', marginBottom: 8 }}>🎉 所有輪次抽獎完成</p>
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
        {currentTier.drawn_count} / {currentTier.winner_count} 人已抽出
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
            {currentIndex + 1 >= totalRounds ? '完成抽獎' : '繼續下一輪'}
          </button>
        </div>
      )}

      {error && <p style={{ fontSize: 12, color: '#f87171', marginTop: 10 }}>{error}</p>}
    </div>
  )
}
