export type CouponItemKey = 'tachiya95' | 'freeShip' | 'bundle120'

export interface DemoCouponMeta {
  id: string
  itemKey: CouponItemKey
  price: number
  code: string
}

export const demoCouponMetas: DemoCouponMeta[] = [
  {
    id: 'tachiya-95',
    itemKey: 'tachiya95',
    price: 18,
    code: 'TACHIYA95',
  },
  {
    id: 'free-ship',
    itemKey: 'freeShip',
    price: 24,
    code: 'SHIPFREE24',
  },
  {
    id: 'bundle-120',
    itemKey: 'bundle120',
    price: 40,
    code: 'DROP120',
  },
]

export function findCouponMetaById(couponId: string): DemoCouponMeta | undefined {
  return demoCouponMetas.find((coupon) => coupon.id === couponId)
}
