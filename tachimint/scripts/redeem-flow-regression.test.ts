import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import http from 'node:http'
import test from 'node:test'

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

test('CouponShopPanel renders redeemed fallback text instead of mock coupon code', async () => {
  const source = await readFile(new URL('../src/app/components/CouponShopPanel.tsx', import.meta.url), 'utf8')

  assert.match(
    source,
    /voucherCodes\[selectedCoupon\.id\]\s*\?\s*t\('coupon\.claimedCode',\s*\{\s*code:\s*voucherCodes\[selectedCoupon\.id\]\s*\}\)\s*:\s*t\('coupon\.redeemed'\)/s,
  )
  assert.doesNotMatch(source, /voucherCodes\[selectedCoupon\.id\]\s*\?\?\s*selectedCoupon\.code/)
  assert.doesNotMatch(source, /code:\s*voucherCodes\[selectedCoupon\.id\]\s*\?\?\s*selectedCoupon\.code/)
})
