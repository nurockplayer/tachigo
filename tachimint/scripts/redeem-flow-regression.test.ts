import assert from 'node:assert/strict'
import http from 'node:http'
import test from 'node:test'
import i18next from 'i18next'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

import { renderCouponRedeemStatus } from '../src/app/components/couponRedeemStatus.ts'
import { executeCouponRedeem } from '../src/app/couponRedeem.ts'

type RecordedRequest = {
  method: string
  url: string
  authorization: string | undefined
  body: unknown
}

async function withApiServer(
  handler: (requests: RecordedRequest[]) => http.RequestListener,
  run: (baseUrl: string, requests: RecordedRequest[]) => Promise<void>,
) {
  const requests: RecordedRequest[] = []
  const server = http.createServer(handler(requests))

  await new Promise<void>((resolve) => {
    server.listen(0, '127.0.0.1', resolve)
  })

  const address = server.address()
  if (!address || typeof address === 'string') {
    throw new Error('failed to resolve test server address')
  }

  const baseUrl = `http://127.0.0.1:${address.port}`

  try {
    await run(baseUrl, requests)
  } finally {
    await new Promise<void>((resolve, reject) => {
      server.close((err) => {
        if (err) reject(err)
        else resolve()
      })
    })
  }
}

async function readJsonBody(req: http.IncomingMessage): Promise<unknown> {
  const chunks: Buffer[] = []
  for await (const chunk of req) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk))
  }

  if (chunks.length === 0) {
    return null
  }

  return JSON.parse(Buffer.concat(chunks).toString('utf8'))
}

test('redeemCoupon unwraps nested ok() response data', async () => {
  await withApiServer(
    (requests) => async (req, res) => {
      const body = await readJsonBody(req)
      requests.push({
        method: req.method ?? 'GET',
        url: req.url ?? '/',
        authorization: req.headers.authorization,
        body,
      })

      if (req.method === 'POST' && req.url === '/spend/redeem') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({
          success: true,
          data: {
            balance: 24,
            voucher_code: 'REAL-VOUCHER-24',
          },
        }))
        return
      }

      res.writeHead(404, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ success: false, error: 'not found' }))
    },
    async (baseUrl, requests) => {
      const originalBaseUrl = process.env.VITE_TACHIGO_API_URL
      process.env.VITE_TACHIGO_API_URL = baseUrl

      try {
        const api = await import(`../src/services/api.ts?redeem=${Date.now()}`)
        const result = await api.redeemCoupon('tachiya-95', 18, 'coupon-jwt-token')

        assert.deepEqual(result, { balance: 24, voucher_code: 'REAL-VOUCHER-24' })
        assert.deepEqual(
          requests.map(({ method, url, authorization, body }) => ({
            method,
            url,
            authorization,
            body,
          })),
          [
            {
              method: 'POST',
              url: '/spend/redeem',
              authorization: 'Bearer coupon-jwt-token',
              body: { coupon_id: 'tachiya-95', amount: 18 },
            },
          ],
        )
      } finally {
        if (originalBaseUrl === undefined) {
          delete process.env.VITE_TACHIGO_API_URL
        } else {
          process.env.VITE_TACHIGO_API_URL = originalBaseUrl
        }
      }
    },
  )
})

async function createTestI18n() {
  const instance = i18next.createInstance()
  await instance.init({
    lng: 'en',
    fallbackLng: 'en',
    interpolation: { escapeValue: false },
    resources: {
      en: {
        common: {
          common: {
            error: 'Something went wrong.',
          },
          coupon: {
            title: 'Coupon Shop',
            header: 'Coupon Shop',
            entry: 'Shop',
            back: 'Back',
            balanceLabel: 'Balance',
            subtitle: 'Redeem your rewards.',
            featured: 'Featured',
            listTitle: 'Available',
            cost: 'Costs {{amount}} TCG',
            redeem: 'Redeem',
            redeemed: 'OWNED',
            claimedCode: 'CODE {{code}} READY',
            insufficientBalance: 'NOT ENOUGH TCG',
            alreadyRedeemed: 'Already redeemed',
            items: {
              tachiya95: {
                brand: 'TACHIYA',
                title: '95% Voucher',
                description: 'Discount for creator goods',
                tag: 'HOT',
              },
              freeShip: {
                brand: 'TACHI MART',
                title: 'Free Shipping',
                description: 'Shipping reward',
                tag: 'SHIP',
              },
              bundle120: {
                brand: 'CREATOR DROP',
                title: '$120 Off',
                description: 'Bundle discount',
                tag: 'DROP',
              },
            },
          },
        },
      },
    },
  })
  return instance
}

test('CouponShopPanel renders the claimed voucher code when one exists', async () => {
  const i18n = await createTestI18n()
  const t = i18n.getFixedT('en', 'common')
  const html = renderToStaticMarkup(renderCouponRedeemStatus({
    error: '',
    isRedeemed: true,
    voucherCode: 'REAL-VOUCHER-24',
    t,
  }))

  assert.match(html, /CODE REAL-VOUCHER-24 READY/)
  assert.doesNotMatch(html, /TACHIYA95/)
})

test('CouponShopPanel does not render the mock coupon code when the voucher code is missing', async () => {
  const i18n = await createTestI18n()
  const t = i18n.getFixedT('en', 'common')
  const html = renderToStaticMarkup(renderCouponRedeemStatus({
    error: '',
    isRedeemed: true,
    voucherCode: '',
    t,
  }))

  assert.match(html, /OWNED/)
  assert.doesNotMatch(html, /TACHIYA95/)
})

test('executeCouponRedeem updates balance, voucher codes, and redeemed ids on success', async () => {
  let balance = 18
  let voucherCodes: Record<string, string> = {}
  let redeemedIds = ['free-ship']

  const redeemedCouponIdsRef = { current: [...redeemedIds] }

  const outcome = await executeCouponRedeem({
    couponId: 'tachiya-95',
    cost: 18,
    jwt: 'coupon-jwt-token',
    redeemedCouponIdsRef,
    setTcgBalance: (nextBalance) => {
      balance = nextBalance
    },
    setVoucherCodes: (updater) => {
      voucherCodes = updater(voucherCodes)
    },
    setRedeemedCouponIds: (nextRedeemedIds) => {
      redeemedIds = nextRedeemedIds
    },
    redeemCouponFn: async () => ({
      balance: 24,
      voucher_code: 'REAL-VOUCHER-24',
    }),
  })

  assert.equal(outcome, 'success')
  assert.equal(balance, 24)
  assert.deepEqual(voucherCodes, { 'tachiya-95': 'REAL-VOUCHER-24' })
  assert.deepEqual(redeemedIds, ['free-ship', 'tachiya-95'])
  assert.deepEqual(redeemedCouponIdsRef.current, ['free-ship', 'tachiya-95'])
})

test('executeCouponRedeem returns insufficient when the backend signals insufficient balance', async () => {
  const outcome = await executeCouponRedeem({
    couponId: 'tachiya-95',
    cost: 18,
    jwt: 'coupon-jwt-token',
    redeemedCouponIdsRef: { current: [] },
    setTcgBalance: () => {
      throw new Error('should not update balance on insufficient')
    },
    setVoucherCodes: () => {
      throw new Error('should not update voucher codes on insufficient')
    },
    setRedeemedCouponIds: () => {
      throw new Error('should not update redeemed ids on insufficient')
    },
    redeemCouponFn: async () => {
      throw new Error('Failed to redeem coupon (402): insufficient balance')
    },
  })

  assert.equal(outcome, 'insufficient')
})

test('executeCouponRedeem returns error when JWT is missing', async () => {
  const outcome = await executeCouponRedeem({
    couponId: 'tachiya-95',
    cost: 18,
    jwt: '',
    redeemedCouponIdsRef: { current: [] },
    setTcgBalance: () => {
      throw new Error('should not update balance without JWT')
    },
    setVoucherCodes: () => {
      throw new Error('should not update voucher codes without JWT')
    },
    setRedeemedCouponIds: () => {
      throw new Error('should not update redeemed ids without JWT')
    },
    redeemCouponFn: async () => ({
      balance: 24,
      voucher_code: 'REAL-VOUCHER-24',
    }),
  })

  assert.equal(outcome, 'error')
})

test('executeCouponRedeem returns error on non-insufficient backend failures', async () => {
  const outcome = await executeCouponRedeem({
    couponId: 'tachiya-95',
    cost: 18,
    jwt: 'coupon-jwt-token',
    redeemedCouponIdsRef: { current: [] },
    setTcgBalance: () => {
      throw new Error('should not update balance on generic error')
    },
    setVoucherCodes: () => {
      throw new Error('should not update voucher codes on generic error')
    },
    setRedeemedCouponIds: () => {
      throw new Error('should not update redeemed ids on generic error')
    },
    redeemCouponFn: async () => {
      throw new Error('Failed to redeem coupon (500): internal error')
    },
  })

  assert.equal(outcome, 'error')
})
