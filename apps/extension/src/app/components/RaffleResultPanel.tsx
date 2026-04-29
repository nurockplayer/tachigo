import { useTranslation } from 'react-i18next'
import { useRaffleResult } from '../../hooks/useRaffleResult'

interface RaffleResultPanelProps {
  raffleId: string
  onBack: () => void
}

export function RaffleResultPanel({ raffleId, onBack }: RaffleResultPanelProps) {
  const { t } = useTranslation()
  const { draws, loading, error } = useRaffleResult(raffleId || null)

  return (
    <div
      style={{
        width: 320,
        height: 600,
        background: '#0d0d1a',
        display: 'flex',
        flexDirection: 'column',
        fontFamily: 'var(--pixel-font-family)',
        userSelect: 'none',
        position: 'relative',
        overflow: 'hidden',
      }}
    >
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '14px 16px',
          borderBottom: '1px solid rgba(145,70,255,0.15)',
        }}
      >
        <button
          onClick={onBack}
          style={{
            background: 'none',
            border: 'none',
            color: '#9146FF',
            fontSize: 8,
            cursor: 'pointer',
            fontFamily: 'var(--pixel-font-family)',
            letterSpacing: '0.06em',
            padding: 0,
          }}
        >
          ‹ {t('raffle_result.back')}
        </button>
        <span
          style={{
            fontSize: 8,
            color: 'rgba(255,255,255,0.5)',
            letterSpacing: '0.1em',
          }}
        >
          {t('raffle_result.title')}
        </span>
      </div>

      {/* Content */}
      <div
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '16px',
        }}
      >
        {loading ? (
          <div
            style={{
              textAlign: 'center',
              color: 'rgba(255,255,255,0.3)',
              fontSize: 8,
              letterSpacing: '0.08em',
              paddingTop: 40,
            }}
          >
            {t('raffle_result.loading')}
          </div>
        ) : error ? (
          <div
            style={{
              textAlign: 'center',
              color: '#ff4444',
              fontSize: 7,
              letterSpacing: '0.06em',
              paddingTop: 40,
            }}
          >
            {t('raffle_result.error_load_failed')}
          </div>
        ) : draws.length === 0 ? (
          <div
            style={{
              textAlign: 'center',
              color: 'rgba(255,255,255,0.3)',
              fontSize: 8,
              letterSpacing: '0.08em',
              paddingTop: 40,
            }}
          >
            {t('raffle_result.no_winners')}
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {[...draws].sort((a, b) => a.drawn_at.localeCompare(b.drawn_at)).map((draw, i) => (
              <div
                key={draw.id}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 10,
                  padding: '10px 12px',
                  background: 'rgba(145,70,255,0.08)',
                  border: '1px solid rgba(145,70,255,0.15)',
                  borderRadius: 6,
                }}
              >
                <span
                  style={{
                    fontSize: 10,
                    color: '#9146FF',
                    fontFamily: 'var(--pixel-font-family)',
                    minWidth: 24,
                    flexShrink: 0,
                  }}
                >
                  #{i + 1}
                </span>
                <span
                  style={{
                    fontSize: 9,
                    color: 'rgba(255,255,255,0.85)',
                    letterSpacing: '0.06em',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {draw.entry.display_name}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
