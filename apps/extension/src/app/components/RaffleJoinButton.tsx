import { useState } from 'react'
import type { CSSProperties } from 'react'
import { joinRaffle } from '../../services/api'
import { useTwitch } from '../../hooks/useTwitch'

interface RaffleJoinButtonProps {
  raffleId: string
  entryOpen: boolean
  raffleActive: boolean
}

const wrapperStyle = {
  background: '#0d0d1a',
  fontFamily: 'var(--pixel-font-family)',
  userSelect: 'none',
} satisfies CSSProperties

const mutedTextStyle = {
  color: 'rgba(255,255,255,0.5)',
  fontFamily: 'var(--pixel-font-family)',
  fontSize: 8,
  letterSpacing: '0.08em',
} satisfies CSSProperties

const buttonStyle = {
  background: '#9146FF',
  border: 'none',
  borderRadius: 6,
  color: '#ffffff',
  cursor: 'pointer',
  fontFamily: 'var(--pixel-font-family)',
  fontSize: 9,
  letterSpacing: '0.08em',
  padding: '10px 14px',
} satisfies CSSProperties

/**
 * Displays a raffle join button for the Twitch Extension panel.
 * Manages join state locally; renders nothing until the backend session is ready
 * or when the raffle is not active.
 */
export function RaffleJoinButton({ raffleId, entryOpen, raffleActive }: RaffleJoinButtonProps) {
  const { backendReady } = useTwitch()
  const [joined, setJoined] = useState(false)
  const [notEligible, setNotEligible] = useState(false)
  const [joining, setJoining] = useState(false)

  if (!backendReady || !raffleActive) {
    return null
  }

  if (joined) {
    return (
      <div style={wrapperStyle}>
        <button type="button" disabled style={{ ...buttonStyle, cursor: 'default', opacity: 0.5 }}>
          已加入 ✓
        </button>
      </div>
    )
  }

  if (notEligible) {
    return (
      <div style={wrapperStyle}>
        <span style={mutedTextStyle}>需訂閱才能參加</span>
      </div>
    )
  }

  if (!entryOpen) {
    return (
      <div style={wrapperStyle}>
        <span style={mutedTextStyle}>報名已截止</span>
      </div>
    )
  }

  async function handleJoin() {
    if (joining) {
      return
    }

    setJoining(true)
    const result = await joinRaffle(raffleId)
    if (result.status === 200 || result.status === 409) {
      setJoined(true)
      return
    }

    if (result.status === 403) {
      setNotEligible(true)
      return
    }

    setJoining(false)
  }

  return (
    <div style={wrapperStyle}>
      <button
        type="button"
        disabled={joining}
        onClick={() => {
          void handleJoin()
        }}
        style={{ ...buttonStyle, opacity: joining ? 0.5 : 1 }}
      >
        我要抽獎
      </button>
    </div>
  )
}
