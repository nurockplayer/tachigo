import { redeemCoupon, type RedeemCouponResponse } from '../services/api.ts'
import type { CouponRedeemResult } from '../extension/types.ts'

export type CouponRedeemOutcome = CouponRedeemResult | 'error'

type CouponRedeemDeps = {
  couponId: string
  cost: number
  jwt: string | null | undefined
  redeemedCouponIdsRef: { current: string[] }
  setTcgBalance: (balance: number) => void
  setVoucherCodes: (updater: (currentCodes: Record<string, string>) => Record<string, string>) => void
  setRedeemedCouponIds: (couponIds: string[]) => void
  redeemCouponFn?: (couponId: string, amount: number, token: string) => Promise<RedeemCouponResponse>
}

function isInsufficientFundsError(error: unknown) {
  return error instanceof Error && /insufficient|balance|402/i.test(error.message)
}

export async function executeCouponRedeem({
  couponId,
  cost,
  jwt,
  redeemedCouponIdsRef,
  setTcgBalance,
  setVoucherCodes,
  setRedeemedCouponIds,
  redeemCouponFn = redeemCoupon,
}: CouponRedeemDeps): Promise<CouponRedeemOutcome> {
  if (!Number.isFinite(cost) || cost <= 0) {
    return 'insufficient'
  }

  if (redeemedCouponIdsRef.current.includes(couponId)) {
    return 'already_redeemed'
  }

  if (!jwt) {
    return 'error'
  }

  try {
    const result = await redeemCouponFn(couponId, cost, jwt)
    redeemedCouponIdsRef.current = [...redeemedCouponIdsRef.current, couponId]
    setTcgBalance(result.balance)
    setVoucherCodes((currentCodes) => ({
      ...currentCodes,
      [couponId]: result.voucher_code,
    }))
    setRedeemedCouponIds(redeemedCouponIdsRef.current)
    return 'success'
  } catch (error) {
    return isInsufficientFundsError(error) ? 'insufficient' : 'error'
  }
}
