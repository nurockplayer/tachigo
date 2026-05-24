import { ClaimPanel } from '../components/ClaimPanel'
import { CouponShopPanel } from '../components/CouponShopPanel'
import { RaffleResultPanel } from '../components/RaffleResultPanel'
import { PlaceholderSurface } from './PlaceholderSurface'
import { useNavigation } from './useNavigation'
import type { CouponRedeemOutcome } from '../couponRedeem'

interface OverlayHostProps {
  cpcBalance: number
  tcgBalance: number
  redeemedCouponIds: string[]
  voucherCodes: Record<string, string>
  onClaim: (cpcAmount: number) => void
  onCouponRedeem: (couponId: string, cost: number) => Promise<CouponRedeemOutcome>
}

function MenuOverlay() {
  const { popOverlay, pushOverlay } = useNavigation()

  return (
    <PlaceholderSurface
      eyebrow="Menu"
      title="Gear Hub"
      body="Dev-only navigation hub for the extension MVP skeleton."
      action={
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {(['account', 'settings', 'character-switch', 'collection', 'missions', 'equipment'] as const).map((kind) => (
            <button key={kind} type="button" onClick={() => pushOverlay({ kind })} style={menuButtonStyle}>
              {kind}
            </button>
          ))}
          <button type="button" onClick={popOverlay} style={menuButtonStyle}>
            close
          </button>
        </div>
      }
    />
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
  tcgBalance,
  redeemedCouponIds,
  voucherCodes,
  onClaim,
  onCouponRedeem,
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
