import { LanguageSwitcher } from '../components/LanguageSwitcher'
import type { AppLanguage } from '../../i18n'
import { useNavigation } from './useNavigation'

interface DevNavigationControlsProps {
  isDev: boolean
  isPopupMode: boolean
  currentLanguage: AppLanguage
  raffleId: string
  onChangeLanguage: (language: AppLanguage) => void
  onChangeRaffleId: (raffleId: string) => void
  onOpenPopupMode: () => void
}

const buttonStyle = {
  padding: '4px 12px',
  borderRadius: 4,
  border: '1px solid rgba(255,255,255,0.1)',
  background: 'transparent',
  color: 'rgba(145,70,255,0.72)',
  fontSize: 9,
  fontFamily: 'var(--pixel-font-family)',
  cursor: 'pointer',
  letterSpacing: '0.08em',
} as const

export function DevNavigationControls({
  isDev,
  isPopupMode,
  currentLanguage,
  raffleId,
  onChangeLanguage,
  onChangeRaffleId,
  onOpenPopupMode,
}: DevNavigationControlsProps) {
  const { goScene, pushOverlay, closeAllOverlays } = useNavigation()

  if (!isDev) {
    return null
  }

  const go = (scene: Parameters<typeof goScene>[0]) => {
    closeAllOverlays()
    goScene(scene)
  }

  const trimmedRaffleId = raffleId.trim()

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 8,
        width: 320,
        maxWidth: '100%',
        position: 'relative',
        zIndex: 2,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8, flexWrap: 'wrap' }}>
        <button type="button" onClick={() => go('entry')} style={buttonStyle}>
          ENTRY
        </button>
        <button type="button" onClick={() => go('login')} style={buttonStyle}>
          LOGIN
        </button>
        <button type="button" onClick={() => go('loading')} style={buttonStyle}>
          LOAD
        </button>
        <button type="button" onClick={() => go('mining')} style={buttonStyle}>
          MINING
        </button>
        <button type="button" onClick={() => pushOverlay({ kind: 'shop' })} style={buttonStyle}>
          SHOP
        </button>
        <button type="button" onClick={() => pushOverlay({ kind: 'claim' })} style={buttonStyle}>
          CLAIM
        </button>
        <button type="button" onClick={() => pushOverlay({ kind: 'menu' })} style={buttonStyle}>
          MENU
        </button>
        <input
          type="text"
          placeholder="raffle id"
          value={raffleId}
          onChange={(event) => onChangeRaffleId(event.target.value)}
          style={{
            padding: '3px 6px',
            borderRadius: 4,
            border: '1px solid rgba(255,255,255,0.1)',
            background: 'rgba(145,70,255,0.06)',
            color: 'rgba(255,255,255,0.5)',
            fontSize: 8,
            fontFamily: 'var(--pixel-font-family)',
            width: 72,
            outline: 'none',
          }}
        />
        <button
          type="button"
          onClick={() => {
            if (trimmedRaffleId) {
              pushOverlay({ kind: 'raffle-result', params: { raffleId: trimmedRaffleId } })
            }
          }}
          style={buttonStyle}
        >
          RAFFLE
        </button>
        <span style={{ fontSize: 9, color: 'rgba(100,100,140,0.3)', fontFamily: 'var(--pixel-font-family)' }}>
          320-430 fluid
        </span>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8, flexWrap: 'wrap' }}>
        <LanguageSwitcher currentLanguage={currentLanguage} onChangeLanguage={onChangeLanguage} />
        {!isPopupMode ? (
          <button type="button" onClick={onOpenPopupMode} style={buttonStyle}>
            POPUP
          </button>
        ) : null}
      </div>
    </div>
  )
}
