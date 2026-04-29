import { createElement, type ReactNode } from 'react'
import type { TFunction } from 'i18next'

type TranslateFn = TFunction<'common'>

const statusStyle = {
  fontSize: 7,
  color: '#b7f7cc',
  letterSpacing: '0.06em',
  lineHeight: 1.8,
} as const

export function renderCouponRedeemStatus({
  error,
  isRedeemed,
  voucherCode,
  t,
}: {
  error: string
  isRedeemed: boolean
  voucherCode: string | undefined
  t: TranslateFn
}): ReactNode {
  if (error) {
    return createElement(
      'div',
      { style: { fontSize: 7, color: '#ff9d7b', letterSpacing: '0.06em', lineHeight: 1.7 } },
      error,
    )
  }

  if (!isRedeemed) {
    return null
  }

  return createElement(
    'div',
    { style: statusStyle },
    voucherCode ? t('coupon.claimedCode', { code: voucherCode }) : t('coupon.redeemed'),
  )
}
