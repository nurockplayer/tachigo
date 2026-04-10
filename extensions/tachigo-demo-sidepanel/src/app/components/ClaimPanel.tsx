import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useSound } from '../hooks/useSound'

const RATE = 0.1

// ─── Gear SVG icon ───────────────────────────────────────────
function GearIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
    </svg>
  )
}

// ─── Token block ─────────────────────────────────────────────
interface TokenBlockProps {
  label: string
  tokenSymbol: string
  balance: number
  balanceLabel: string
  value: string
  readOnly: boolean
  maxValue?: number
  onValueChange?: (v: string) => void
  onMax?: () => void
  maxLabel: string
}

function TokenBlock({ label, tokenSymbol, balance, balanceLabel, value, readOnly, maxValue, onValueChange, onMax, maxLabel }: TokenBlockProps) {
  return (
    <div
      style={{
        background: 'rgba(255,255,255,0.04)',
        border: '1px solid rgba(145,70,255,0.2)',
        borderRadius: 10,
        padding: '14px 16px',
        overflow: 'hidden',
      }}
    >
      {/* Label row */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 10,
        }}
      >
        <span style={{ fontSize: 7, color: '#666', letterSpacing: '0.1em' }}>{label}</span>
        <span
          style={{
            fontSize: 7,
            color: '#555',
            letterSpacing: '0.05em',
          }}
        >
          {balanceLabel.replace('{{amount}}', balance.toLocaleString())}
        </span>
      </div>

      {/* Input row */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        {/* Token symbol pill */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 5,
            padding: '4px 8px',
            background: 'rgba(145,70,255,0.12)',
            border: '1px solid rgba(145,70,255,0.25)',
            borderRadius: 6,
            flexShrink: 0,
          }}
        >
          <div
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              background: tokenSymbol === 'CPC' ? '#FFB000' : '#9146FF',
              boxShadow: tokenSymbol === 'CPC'
                ? '0 0 6px rgba(255,176,0,0.7)'
                : '0 0 6px rgba(145,70,255,0.7)',
            }}
          />
          <span
            style={{
              fontSize: 9,
              color: 'white',
              letterSpacing: '0.06em',
              fontFamily: 'var(--pixel-font-family)',
            }}
          >
            {tokenSymbol}
          </span>
        </div>

        {/* Numeric input / output */}
        <input
          type="text"
          inputMode="numeric"
          value={value}
          readOnly={readOnly}
          onChange={readOnly ? undefined : (e) => onValueChange?.(e.target.value)}
          placeholder="0"
          style={{
            flex: 1,
            minWidth: 0,
            background: 'transparent',
            border: 'none',
            borderBottom: readOnly
              ? '1px solid rgba(145,70,255,0.1)'
              : '1px solid rgba(145,70,255,0.4)',
            color: readOnly ? '#666' : 'white',
            fontSize: 20,
            fontFamily: 'var(--pixel-font-family)',
            outline: 'none',
            textAlign: 'right',
            padding: '2px 0',
          }}
        />

        {/* MAX button (only on editable block) */}
        {!readOnly && onMax && (
          <button
            onClick={onMax}
            style={{
              padding: '3px 7px',
              borderRadius: 4,
              border: '1px solid rgba(145,70,255,0.35)',
              background: 'rgba(145,70,255,0.1)',
              color: '#9146FF',
              fontSize: 7,
              cursor: 'pointer',
              fontFamily: 'var(--pixel-font-family)',
              letterSpacing: '0.06em',
              flexShrink: 0,
            }}
          >
            {maxLabel}
          </button>
        )}
      </div>
    </div>
  )
}

// ─── Main ClaimPanel ─────────────────────────────────────────
interface ClaimPanelProps {
  onBack: () => void
  cpcBalance: number
  tcgBalance: number
  onClaim: (cpcAmount: number) => void
}

export function ClaimPanel({ onBack, cpcBalance, tcgBalance, onClaim }: ClaimPanelProps) {
  const { t } = useTranslation()
  const { playClaimSound } = useSound()
  const [cpcInput, setCpcInput] = useState('')
  const [claimed, setClaimed] = useState(false)

  const numericCpc = parseFloat(cpcInput)
  const tcgOutput = cpcInput !== '' && !isNaN(numericCpc) && numericCpc > 0
    ? (numericCpc * RATE).toFixed(2)
    : ''
  const isDisabled = !cpcInput || isNaN(numericCpc) || numericCpc <= 0
    || numericCpc > cpcBalance || claimed

  const handleClaim = () => {
    if (isDisabled) return
    playClaimSound()
    onClaim(numericCpc)
    setClaimed(true)
  }

  // Reset after 2s
  useEffect(() => {
    if (!claimed) return
    const timer = setTimeout(() => {
      setClaimed(false)
      setCpcInput('')
    }, 2000)
    return () => clearTimeout(timer)
  }, [claimed])

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
      {/* ── Header ───────────────────────────────────────── */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '14px 16px',
          borderBottom: '1px solid rgba(145,70,255,0.15)',
        }}
      >
        {/* Back button */}
        <button
          onClick={onBack}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 4,
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
          ‹ BACK
        </button>

        {/* Settings gear (demo, no action) */}
        <div style={{ color: '#444', display: 'flex', alignItems: 'center' }}>
          <GearIcon />
        </div>
      </div>

      {/* ── Main content ─────────────────────────────────── */}
      <div
        style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          padding: '0 20px',
          gap: 0,
        }}
      >
        {/* FROM block */}
        <TokenBlock
          label={t('claim.from')}
          tokenSymbol="CPC"
          balance={cpcBalance}
          balanceLabel={t('claim.balance', { amount: cpcBalance.toLocaleString() })}
          value={cpcInput}
          readOnly={false}
          maxValue={cpcBalance}
          onValueChange={setCpcInput}
          onMax={() => setCpcInput(String(cpcBalance))}
          maxLabel={t('claim.max')}
        />

        {/* Direction arrow */}
        <div
          style={{
            textAlign: 'center',
            color: 'rgba(145,70,255,0.5)',
            fontSize: 18,
            lineHeight: 1,
            padding: '12px 0',
          }}
        >
          ↓
        </div>

        {/* TO block */}
        <TokenBlock
          label={t('claim.to')}
          tokenSymbol="TCG"
          balance={tcgBalance}
          balanceLabel={t('claim.balance', { amount: tcgBalance.toLocaleString() })}
          value={tcgOutput}
          readOnly
          maxLabel=""
        />

        {/* Rate info */}
        <div
          style={{
            textAlign: 'center',
            fontSize: 7,
            color: '#444',
            letterSpacing: '0.08em',
            marginTop: 18,
          }}
        >
          {t('claim.rate')}
        </div>

        {/* Claim button */}
        <button
          onClick={handleClaim}
          disabled={isDisabled}
          style={{
            marginTop: 24,
            width: '100%',
            padding: '14px 0',
            borderRadius: 8,
            border: 'none',
            background: claimed
              ? 'rgba(145,70,255,0.3)'
              : isDisabled
                ? 'rgba(145,70,255,0.15)'
                : '#9146FF',
            color: isDisabled && !claimed ? '#555' : 'white',
            fontSize: 10,
            letterSpacing: '0.1em',
            cursor: isDisabled ? 'not-allowed' : 'pointer',
            fontFamily: 'var(--pixel-font-family)',
            opacity: isDisabled && !claimed ? 0.6 : 1,
            transition: 'background 0.2s, opacity 0.2s',
            boxShadow: !isDisabled && !claimed ? '0 0 16px rgba(145,70,255,0.4)' : 'none',
          }}
        >
          {claimed ? t('claim.success') : t('claim.button')}
        </button>
      </div>
    </div>
  )
}
