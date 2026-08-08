import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { demoCouponMetas, type DemoCouponMeta } from '../../extension/couponCatalog'
import type { CouponRedeemResult } from '../../extension/types'
import { hudPanelBackground } from '../theme/backgrounds'
import { renderCouponRedeemStatus } from './couponRedeemStatus'
import { redeemCouponForPanel } from './redeemCouponForPanel'

const couponMetas: DemoCouponMeta[] = demoCouponMetas
const shopCategories = [
  { key: 'tachiya', translationKey: 'tachiya', active: true },
  { key: 'recruit', translationKey: 'recruit', active: false },
  { key: 'gear', translationKey: 'gear', active: false },
  { key: 'future', translationKey: 'future', active: false },
] as const

type ShopStep = 'categories' | 'market'

interface CouponShopPanelProps {
  onBack: () => void
  tcgBalance: number
  redeemedCouponIds: string[]
  voucherCodes: Record<string, string>
  onRedeem: (couponId: string, cost: number) => Promise<CouponRedeemResult | 'error'>
}

export function CouponShopPanel({
  onBack,
  tcgBalance,
  redeemedCouponIds,
  voucherCodes,
  onRedeem,
}: CouponShopPanelProps) {
  const { t } = useTranslation()
  const tDyn = t as (key: string, options?: Record<string, unknown>) => string
  const [shopStep, setShopStep] = useState<ShopStep>('categories')
  const [selectedId, setSelectedId] = useState(couponMetas[0]?.id ?? '')
  const [error, setError] = useState('')
  const [isRedeeming, setIsRedeeming] = useState(false)

  const selectedCoupon = useMemo(
    () => couponMetas.find((coupon) => coupon.id === selectedId) ?? couponMetas[0],
    [selectedId],
  )

  const itemPath = (field: 'brand' | 'title' | 'description' | 'tag') =>
    `coupon.items.${selectedCoupon.itemKey}.${field}` as const

  const handleRedeem = async () => {
    if (!selectedCoupon || isRedeeming) {
      return
    }

    if (redeemedCouponIds.includes(selectedCoupon.id)) {
      setError(t('coupon.alreadyRedeemed'))
      return
    }

    setIsRedeeming(true)
    try {
      await redeemCouponForPanel({
        couponId: selectedCoupon.id,
        cost: selectedCoupon.price,
        messages: {
          alreadyRedeemed: t('coupon.alreadyRedeemed'),
          insufficientBalance: t('coupon.insufficientBalance'),
          genericError: t('common.error'),
        },
        onRedeem,
        setError,
      })
    } finally {
      setIsRedeeming(false)
    }
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
          type="button"
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
          {t('coupon.back')}
        </button>
        <div style={{ fontSize: 8, color: '#b794ff', letterSpacing: '0.12em' }}>
          {shopStep === 'categories' ? t('coupon.categories.header') : t('coupon.header')}
        </div>
      </div>

      {shopStep === 'categories' ? (
        <div style={{ padding: '16px', display: 'flex', flexDirection: 'column', gap: 12 }}>
          <div
            style={{
              border: '1px solid rgba(125,211,252,0.22)',
              borderRadius: 14,
              padding: '16px 14px',
              background:
                'linear-gradient(180deg, rgba(14, 116, 144, 0.24) 0%, rgba(15, 23, 42, 0.68) 100%)',
              boxShadow: '0 14px 34px rgba(2, 6, 23, 0.3)',
            }}
          >
            <div style={{ fontSize: 7, color: '#67e8f9', letterSpacing: '0.14em', marginBottom: 8 }}>
              {t('coupon.categories.eyebrow')}
            </div>
            <h2 style={{ margin: 0, color: '#fff7da', fontSize: 18, lineHeight: 1.5 }}>
              {t('coupon.categories.title')}
            </h2>
            <p style={{ margin: '10px 0 0', fontFamily: 'var(--ui-font-family)', fontSize: 12, lineHeight: 1.55 }}>
              {t('coupon.categories.subtitle')}
            </p>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 10 }}>
            {shopCategories.map((category) => {
              const title = tDyn(`coupon.categories.cards.${category.translationKey}.title`)
              const description = tDyn(`coupon.categories.cards.${category.translationKey}.description`)

              return (
                <button
                  key={category.key}
                  type="button"
                  aria-label={title}
                  onClick={() => {
                    if (category.active) {
                      setShopStep('market')
                    }
                  }}
                  disabled={!category.active}
                  style={{
                    minHeight: 128,
                    border: category.active
                      ? '1px solid rgba(255, 211, 107, 0.46)'
                      : '1px solid rgba(148, 163, 184, 0.16)',
                    borderRadius: 12,
                    padding: 12,
                    background: category.active
                      ? 'linear-gradient(155deg, rgba(255,211,107,0.2), rgba(88,28,135,0.28) 52%, rgba(8,47,73,0.72))'
                      : 'linear-gradient(155deg, rgba(15,23,42,0.82), rgba(30,41,59,0.64))',
                    color: category.active ? '#fff7da' : 'rgba(226,232,240,0.52)',
                    cursor: category.active ? 'pointer' : 'not-allowed',
                    textAlign: 'left',
                    fontFamily: 'var(--pixel-font-family)',
                    display: 'flex',
                    flexDirection: 'column',
                    justifyContent: 'space-between',
                    gap: 12,
                    boxShadow: category.active
                      ? '0 0 22px rgba(255,211,107,0.16), inset 0 1px 0 rgba(255,255,255,0.12)'
                      : 'inset 0 1px 0 rgba(255,255,255,0.06)',
                  }}
                >
                  <span style={{ display: 'grid', gap: 8 }}>
                    <span style={{ fontSize: 13, lineHeight: 1.35 }}>{title}</span>
                    <span
                      style={{
                        fontFamily: 'var(--ui-font-family)',
                        fontSize: 11,
                        lineHeight: 1.45,
                        color: category.active ? 'rgba(248,241,223,0.78)' : 'rgba(203,213,225,0.46)',
                      }}
                    >
                      {description}
                    </span>
                  </span>
                  <span
                    style={{
                      alignSelf: 'flex-start',
                      borderRadius: 999,
                      padding: '5px 7px',
                      background: category.active ? 'rgba(255,211,107,0.18)' : 'rgba(148,163,184,0.1)',
                      color: category.active ? '#ffd36b' : 'rgba(203,213,225,0.48)',
                      fontSize: 6,
                      letterSpacing: '0.1em',
                    }}
                  >
                    {category.active ? t('coupon.categories.active') : t('coupon.categories.locked')}
                  </span>
                </button>
              )
            })}
          </div>
        </div>
      ) : (
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
              {tDyn(itemPath('tag'))}
            </div>
          </div>
          <div style={{ fontSize: 8, color: '#b794ff', letterSpacing: '0.12em' }}>
            {tDyn(`coupon.items.${selectedCoupon.itemKey}.brand`)}
          </div>
          <div style={{ fontSize: 14, color: '#fff7da', lineHeight: 1.5 }}>{tDyn(itemPath('title'))}</div>
          <div style={{ fontSize: 7, color: 'rgba(225,218,255,0.74)', lineHeight: 1.8 }}>
            {tDyn(itemPath('description'))}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12 }}>
            <div style={{ fontSize: 8, color: '#d7c2ff', letterSpacing: '0.1em' }}>
              {t('coupon.cost', { amount: selectedCoupon.price })}
            </div>
            <button
              type="button"
              onClick={handleRedeem}
              disabled={isRedeeming}
              style={{
                border: '1px solid rgba(255,176,0,0.35)',
                background: 'linear-gradient(180deg, #FFD36B 0%, #FFB000 100%)',
                color: '#4b1700',
                padding: '8px 12px',
                borderRadius: 8,
                fontSize: 8,
                cursor: isRedeeming ? 'wait' : 'pointer',
                fontFamily: 'var(--pixel-font-family)',
                letterSpacing: '0.08em',
                boxShadow: '0 0 16px rgba(255,176,0,0.24)',
                opacity: isRedeeming ? 0.7 : 1,
              }}
            >
              {t('coupon.redeem')}
            </button>
          </div>
          {renderCouponRedeemStatus({
            error,
            isRedeemed: redeemedCouponIds.includes(selectedCoupon.id),
            voucherCode: voucherCodes[selectedCoupon.id],
            t,
          })}
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div style={{ fontSize: 7, color: '#9146FF', letterSpacing: '0.14em' }}>{t('coupon.listTitle')}</div>
          {couponMetas.map((coupon) => {
            const isSelected = coupon.id === selectedId
            const isRedeemed = redeemedCouponIds.includes(coupon.id)

            return (
              <button
                type="button"
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
                    {tDyn(`coupon.items.${coupon.itemKey}.brand`)}
                  </div>
                  <div
                    style={{
                      fontSize: 8,
                      color: isSelected ? '#F5E5B8' : '#E8D7A8',
                      lineHeight: 1.7,
                    }}
                  >
                    {tDyn(`coupon.items.${coupon.itemKey}.title`)}
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
                    {isRedeemed ? t('coupon.redeemed') : tDyn(`coupon.items.${coupon.itemKey}.tag`)}
                  </div>
                </div>
              </button>
            )
          })}
        </div>
        </div>
      )}
    </div>
  )
}
