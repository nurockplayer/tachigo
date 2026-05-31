import { AccountPanel } from '../components/AccountPanel'
import { ClaimPanel } from '../components/ClaimPanel'
import { CouponShopPanel } from '../components/CouponShopPanel'
import { RaffleResultPanel } from '../components/RaffleResultPanel'
import { SettingsPanel } from '../components/SettingsPanel'
import { PlaceholderSurface } from './PlaceholderSurface'
import { useNavigation } from './useNavigation'
import type { CouponRedeemOutcome } from '../couponRedeem'
import type { Overlay } from './types'
import type { SettingsState } from '../../extension/types'
import type { AppLanguage } from '../../i18n'

interface OverlayHostProps {
  cpcBalance: number
  currentLanguage: AppLanguage
  tcgBalance: number
  redeemedCouponIds: string[]
  settings: SettingsState
  voucherCodes: Record<string, string>
  onChangeLanguage: (language: AppLanguage) => void
  onClaim: (cpcAmount: number) => void
  onCouponRedeem: (couponId: string, cost: number) => Promise<CouponRedeemOutcome>
  onSettingsChange: (settings: SettingsState) => void
}

const menuEntries: Array<{
  kind: Extract<Overlay, 'account' | 'settings' | 'character-switch' | 'collection' | 'missions' | 'equipment'>
  label: string
  code: string
  accent: string
}> = [
  { kind: 'account', label: 'Account', code: 'AC', accent: '#38bdf8' },
  { kind: 'settings', label: 'Settings', code: 'ST', accent: '#facc15' },
  { kind: 'character-switch', label: 'Character', code: 'CH', accent: '#34d399' },
  { kind: 'collection', label: 'Collection', code: 'CO', accent: '#f472b6' },
  { kind: 'missions', label: 'Missions', code: 'MS', accent: '#a78bfa' },
  { kind: 'equipment', label: 'Equipment', code: 'EQ', accent: '#fb923c' },
]

function MenuOverlay() {
  const { popOverlay, pushOverlay } = useNavigation()

  return (
    <div
      style={{
        position: 'relative',
        width: '100%',
        height: '100%',
        minHeight: 600,
        overflow: 'hidden',
        padding: '28px 22px 24px',
        boxSizing: 'border-box',
        background:
          'linear-gradient(180deg, rgba(12, 18, 32, 0.98), rgba(4, 47, 46, 0.96) 56%, rgba(17, 24, 39, 0.98))',
        color: '#f8fafc',
        fontFamily: 'var(--ui-font-family)',
      }}
    >
      <div
        aria-hidden="true"
        style={{
          position: 'absolute',
          inset: 0,
          background:
            'linear-gradient(90deg, rgba(125,211,252,0.08) 1px, transparent 1px), linear-gradient(180deg, rgba(125,211,252,0.06) 1px, transparent 1px)',
          backgroundSize: '28px 28px',
          opacity: 0.7,
        }}
      />
      <div style={{ position: 'relative', zIndex: 1, display: 'grid', height: '100%', gap: 18 }}>
        <header style={{ display: 'grid', gap: 8 }}>
          <div
            style={{
              color: '#7dd3fc',
              fontFamily: 'var(--pixel-font-family)',
              fontSize: 9,
              letterSpacing: 0,
              textTransform: 'uppercase',
            }}
          >
            Menu
          </div>
          <h1 style={{ margin: 0, fontFamily: 'var(--pixel-font-family)', fontSize: 28, lineHeight: 1 }}>
            Gear Hub
          </h1>
        </header>

        <div
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
            alignContent: 'start',
            gap: 10,
          }}
        >
          {menuEntries.map((entry) => (
            <button
              key={entry.kind}
              type="button"
              aria-label={entry.label}
              onClick={() => pushOverlay({ kind: entry.kind })}
              style={{
                minHeight: 92,
                border: '1px solid rgba(226, 232, 240, 0.12)',
                borderRadius: 8,
                padding: 12,
                background: 'rgba(15, 23, 42, 0.78)',
                color: '#f8fafc',
                cursor: 'pointer',
                textAlign: 'left',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                gap: 12,
                boxShadow: `inset 0 3px 0 ${entry.accent}, 0 10px 26px rgba(2, 6, 23, 0.24)`,
                fontFamily: 'var(--ui-font-family)',
              }}
            >
              <span
                aria-hidden="true"
                style={{
                  display: 'inline-grid',
                  placeItems: 'center',
                  width: 28,
                  height: 28,
                  borderRadius: 999,
                  background: entry.accent,
                  color: '#020617',
                  fontFamily: 'var(--pixel-font-family)',
                  fontSize: 9,
                }}
              >
                {entry.code}
              </span>
              <span style={{ flex: 1, fontWeight: 700, fontSize: 13, lineHeight: 1.2 }}>{entry.label}</span>
            </button>
          ))}
        </div>

        <button
          type="button"
          onClick={popOverlay}
          style={{
            alignSelf: 'end',
            minHeight: 36,
            border: '1px solid rgba(148, 163, 184, 0.22)',
            borderRadius: 8,
            background: 'rgba(15, 23, 42, 0.58)',
            color: '#e2e8f0',
            cursor: 'pointer',
            fontFamily: 'var(--pixel-font-family)',
            fontSize: 9,
          }}
        >
          Close
        </button>
      </div>
    </div>
  )
}

const menuButtonStyle = {
  border: '1px solid rgba(125,211,252,0.32)',
  background: 'rgba(15,23,42,0.74)',
  color: '#e0f2fe',
  borderRadius: 6,
  padding: '9px 10px',
  fontFamily: 'var(--pixel-font-family)',
  fontSize: 9,
  cursor: 'pointer',
  textTransform: 'uppercase',
} as const

export function OverlayHost({
  cpcBalance,
  currentLanguage,
  tcgBalance,
  redeemedCouponIds,
  settings,
  voucherCodes,
  onChangeLanguage,
  onClaim,
  onCouponRedeem,
  onSettingsChange,
}: OverlayHostProps) {
  const { state, popOverlay } = useNavigation()

  return (
    <>
      {state.overlayStack.map((entry, index) => (
        <div
          key={`${entry.kind}-${index}`}
          style={{
            position: 'absolute',
            inset: 0,
            zIndex: 20 + index,
            background: 'rgba(2,6,23,0.72)',
          }}
        >
          {entry.kind === 'claim' ? (
            <ClaimPanel onBack={popOverlay} cpcBalance={cpcBalance} tcgBalance={tcgBalance} onClaim={onClaim} />
          ) : entry.kind === 'shop' ? (
            <CouponShopPanel
              onBack={popOverlay}
              tcgBalance={tcgBalance}
              redeemedCouponIds={redeemedCouponIds}
              voucherCodes={voucherCodes}
              onRedeem={onCouponRedeem}
            />
          ) : entry.kind === 'raffle-result' ? (
            entry.params.raffleId.trim() === '' ? (
              <PlaceholderSurface
                eyebrow="Dev-only"
                title="Invalid Raffle"
                body="Enter a raffle id before opening the result panel."
                action={
                  <button type="button" onClick={popOverlay} style={menuButtonStyle}>
                    back
                  </button>
                }
              />
            ) : (
              <RaffleResultPanel raffleId={entry.params.raffleId.trim()} onBack={popOverlay} />
            )
          ) : entry.kind === 'menu' ? (
            <MenuOverlay />
          ) : entry.kind === 'account' ? (
            <AccountPanel onBack={popOverlay} />
          ) : entry.kind === 'settings' ? (
            <SettingsPanel
              currentLanguage={currentLanguage}
              settings={settings}
              onBack={popOverlay}
              onChangeLanguage={onChangeLanguage}
              onChange={onSettingsChange}
            />
          ) : (
            <PlaceholderSurface
              eyebrow="Dev-only"
              title={entry.kind}
              body="Placeholder overlay reserved for a follow-up issue."
              action={
                <button type="button" onClick={popOverlay} style={menuButtonStyle}>
                  back
                </button>
              }
            />
          )}
        </div>
      ))}
    </>
  )
}
