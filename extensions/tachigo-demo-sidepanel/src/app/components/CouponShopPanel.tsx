import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { hudPanelBackground } from '../theme/backgrounds'

interface CouponItem {
  id: string
  brand: string
  title: string
  description: string
  price: number
  tag: string
  code: string
}

const COUPONS: CouponItem[] = [
  {
    id: 'tachiya-95',
    brand: 'TACHIYA',
    title: '95折折扣碼',
    description: '適用於精選周邊與 VTuber 聯名商品',
    price: 18,
    tag: 'HOT',
    code: 'TACHIYA95',
  },
  {
    id: 'free-ship',
    brand: 'TACHI MART',
    title: '免運券',
    description: '全站單筆訂單滿額即可使用',
    price: 24,
    tag: 'SHIP',
    code: 'SHIPFREE24',
  },
  {
    id: 'bundle-120',
    brand: 'CREATOR DROP',
    title: '現折 $120',
    description: '限本月合作創作者商品專區',
    price: 40,
    tag: 'DROP',
    code: 'DROP120',
  },
]

interface CouponShopPanelProps {
  onBack: () => void
  tcgBalance: number
  onRedeem: (cost: number) => boolean
}

export function CouponShopPanel({ onBack, tcgBalance, onRedeem }: CouponShopPanelProps) {
  const { t } = useTranslation()
  const [selectedId, setSelectedId] = useState(COUPONS[0]?.id ?? '')
  const [redeemedCodes, setRedeemedCodes] = useState<string[]>([])
  const [error, setError] = useState('')

  const selectedCoupon = useMemo(
    () => COUPONS.find((coupon) => coupon.id === selectedId) ?? COUPONS[0],
    [selectedId],
  )

  const handleRedeem = () => {
    if (!selectedCoupon) {
      return
    }

    if (redeemedCodes.includes(selectedCoupon.id)) {
      setError(t('coupon.alreadyRedeemed'))
      return
    }

    const success = onRedeem(selectedCoupon.price)
    if (!success) {
      setError(t('coupon.insufficientBalance'))
      return
    }

    setError('')
    setRedeemedCodes((current) => [...current, selectedCoupon.id])
  }

  return (
    <div
      style={{
        width: 320,
        height: 600,
        display: 'flex',
        flexDirection: 'column',
        color: '#f8f1df',
        background: hudPanelBackground,
        fontFamily: 'var(--pixel-font-family)',
        overflow: 'hidden',
        position: 'relative',
      }}
    >
      <div
        style={{
          padding: '14px 16px 12px',
          borderBottom: '1px solid rgba(145,70,255,0.15)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <button
          onClick={onBack}
          style={{
            background: 'none',
            border: 'none',
            color: '#9146FF',
            cursor: 'pointer',
            fontSize: 8,
            letterSpacing: '0.08em',
            padding: 0,
            fontFamily: 'var(--pixel-font-family)',
          }}
        >
          ‹ BACK
        </button>
        <div style={{ fontSize: 8, color: '#b794ff', letterSpacing: '0.12em' }}>
          {t('coupon.header')}
        </div>
      </div>

      <div style={{ padding: '16px', display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div
          style={{
            border: '1px solid rgba(145,70,255,0.24)',
            borderRadius: 12,
            padding: '14px 14px 12px',
            background: 'linear-gradient(180deg, rgba(145,70,255,0.16) 0%, rgba(255,255,255,0.03) 100%)',
            boxShadow: '0 12px 36px rgba(0,0,0,0.34)',
          }}
        >
          <div style={{ fontSize: 7, color: '#b794ff', letterSpacing: '0.14em', marginBottom: 8 }}>
            {t('coupon.balanceLabel')}
          </div>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 8 }}>
            <div style={{ fontSize: 32, color: '#fff2bf', lineHeight: 1 }}>{tcgBalance.toFixed(2)}</div>
            <div style={{ fontSize: 8, color: '#9146FF', letterSpacing: '0.1em' }}>TCG</div>
          </div>
          <div style={{ marginTop: 10, fontSize: 7, color: 'rgba(225,218,255,0.7)', lineHeight: 1.7 }}>
            {t('coupon.subtitle')}
          </div>
        </div>

        <div
          style={{
            borderRadius: 14,
            padding: 14,
            background: 'linear-gradient(135deg, rgba(145,70,255,0.22) 0%, rgba(25,12,44,0.92) 100%)',
            border: '1px solid rgba(145,70,255,0.28)',
            display: 'flex',
            flexDirection: 'column',
            gap: 10,
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10 }}>
            <div style={{ fontSize: 7, color: '#d7c2ff', letterSpacing: '0.14em' }}>
              {t('coupon.featured')}
            </div>
            <div
              style={{
                padding: '4px 6px',
                borderRadius: 999,
                background: 'rgba(145,70,255,0.16)',
                color: '#efe2ff',
                fontSize: 6,
                letterSpacing: '0.12em',
              }}
            >
              {selectedCoupon.tag}
            </div>
          </div>
          <div style={{ fontSize: 8, color: '#b794ff', letterSpacing: '0.12em' }}>{selectedCoupon.brand}</div>
          <div style={{ fontSize: 14, color: '#fff7da', lineHeight: 1.5 }}>{selectedCoupon.title}</div>
          <div style={{ fontSize: 7, color: 'rgba(225,218,255,0.74)', lineHeight: 1.8 }}>
            {selectedCoupon.description}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
            <div style={{ fontSize: 8, color: '#d7c2ff', letterSpacing: '0.1em' }}>
              {t('coupon.cost', { amount: selectedCoupon.price })}
            </div>
            <button
              onClick={handleRedeem}
              style={{
                border: '1px solid rgba(255,176,0,0.35)',
                background: 'linear-gradient(180deg, #FFD36B 0%, #FFB000 100%)',
                color: '#4b1700',
                padding: '8px 12px',
                borderRadius: 8,
                fontSize: 8,
                cursor: 'pointer',
                fontFamily: 'var(--pixel-font-family)',
                letterSpacing: '0.08em',
                boxShadow: '0 0 16px rgba(255,176,0,0.24)',
              }}
            >
              {t('coupon.redeem')}
            </button>
          </div>
          {error ? (
            <div style={{ fontSize: 7, color: '#ff9d7b', letterSpacing: '0.06em', lineHeight: 1.7 }}>{error}</div>
          ) : redeemedCodes.includes(selectedCoupon.id) ? (
            <div style={{ fontSize: 7, color: '#b7f7cc', letterSpacing: '0.06em', lineHeight: 1.8 }}>
              {t('coupon.claimedCode', { code: selectedCoupon.code })}
            </div>
          ) : null}
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div style={{ fontSize: 7, color: '#9146FF', letterSpacing: '0.14em' }}>{t('coupon.listTitle')}</div>
          {COUPONS.map((coupon) => {
            const isSelected = coupon.id === selectedId
            const isRedeemed = redeemedCodes.includes(coupon.id)

            return (
              <button
                key={coupon.id}
                onClick={() => {
                  setSelectedId(coupon.id)
                  setError('')
                }}
                style={{
                  textAlign: 'left',
                  border: isSelected ? '1px solid rgba(225,176,82,0.36)' : '1px solid rgba(205,164,92,0.14)',
                  background: isSelected
                    ? 'linear-gradient(180deg, rgba(225,176,82,0.14) 0%, rgba(225,176,82,0.06) 100%)'
                    : 'linear-gradient(180deg, rgba(205,164,92,0.05) 0%, rgba(255,255,255,0.02) 100%)',
                  borderRadius: 10,
                  padding: '12px 12px 10px',
                  color: '#f8f1df',
                  cursor: 'pointer',
                  fontFamily: 'var(--pixel-font-family)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: 10,
                }}
              >
                <div style={{ minWidth: 0 }}>
                  <div
                    style={{
                      fontSize: 8,
                      color: isSelected ? '#E5B257' : '#C99B49',
                      letterSpacing: '0.08em',
                      marginBottom: 6,
                    }}
                  >
                    {coupon.brand}
                  </div>
                  <div
                    style={{
                      fontSize: 8,
                      color: isSelected ? '#F5E5B8' : '#E8D7A8',
                      lineHeight: 1.7,
                    }}
                  >
                    {coupon.title}
                  </div>
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: 6, flexShrink: 0 }}>
                  <div
                    style={{
                      fontSize: 8,
                      color: isSelected ? '#EBC36A' : '#D6AE58',
                    }}
                  >
                    {coupon.price} TCG
                  </div>
                  <div
                    style={{
                      fontSize: 6,
                      color: isRedeemed ? '#EFDCA6' : '#8F7140',
                      letterSpacing: '0.1em',
                    }}
                  >
                    {isRedeemed ? t('coupon.redeemed') : coupon.tag}
                  </div>
                </div>
              </button>
            )
          })}
        </div>
      </div>
    </div>
  )
}
