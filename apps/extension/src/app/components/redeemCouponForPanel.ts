import type { CouponRedeemResult } from '../../extension/types'

interface RedeemPanelMessages {
  alreadyRedeemed: string
  insufficientBalance: string
  genericError: string
}

export async function redeemCouponForPanel({
  couponId,
  cost,
  messages,
  onRedeem,
  setError,
}: {
  couponId: string
  cost: number
  messages: RedeemPanelMessages
  onRedeem: (couponId: string, cost: number) => Promise<CouponRedeemResult | 'error'>
  setError: (message: string) => void
}): Promise<void> {
  try {
    const result = await onRedeem(couponId, cost)
    if (result === 'already_redeemed') {
      setError(messages.alreadyRedeemed)
      return
    }
    if (result === 'insufficient') {
      setError(messages.insufficientBalance)
      return
    }
    if (result === 'error') {
      setError(messages.genericError)
      return
    }
    setError('')
  } catch (err) {
    setError(err instanceof Error && err.message ? err.message : messages.genericError)
  }
}
