import assert from 'node:assert/strict'
import http from 'node:http'
import { test, vi } from 'vitest'

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

test('sendHeartbeat returns heartbeat balance without refreshing point balance', async () => {
  await withApiServer(
    (requests) => async (req, res) => {
      const body = await readJsonBody(req)
      requests.push({
        method: req.method ?? 'GET',
        url: req.url ?? '/',
        authorization: req.headers.authorization,
        body,
      })

      if (req.method === 'POST' && req.url === '/api/v1/extension/watch/start') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { started: true } }))
        return
      }

      if (req.method === 'POST' && req.url === '/api/v1/extension/watch/heartbeat') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(
          JSON.stringify({
            success: true,
            data: {
              points_earned: 2,
              spendable_balance: 42,
              cumulative_total: 77,
            },
          }),
        )
        return
      }

      res.writeHead(404, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ success: false, error: 'not found' }))
    },
    async (baseUrl, requests) => {
      const originalBaseUrl = process.env.VITE_TACHIGO_API_URL
      process.env.VITE_TACHIGO_API_URL = baseUrl

      try {
        vi.resetModules()
        const api = await import('./api.ts')

        api.setAuthToken('tachigo-access-token')
        const result = await api.sendHeartbeat('channel-123')

        assert.deepEqual(result, { balance: 42, cumulativeTotal: 77 })
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
              url: '/api/v1/extension/watch/start',
              authorization: 'Bearer tachigo-access-token',
              body: { channel_id: 'channel-123' },
            },
            {
              method: 'POST',
              url: '/api/v1/extension/watch/heartbeat',
              authorization: 'Bearer tachigo-access-token',
              body: { channel_id: 'channel-123' },
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

test('sendHeartbeat falls back when heartbeat and point balance responses miss cumulative total', async () => {
  await withApiServer(
    (requests) => async (req, res) => {
      const body = await readJsonBody(req)
      requests.push({
        method: req.method ?? 'GET',
        url: req.url ?? '/',
        authorization: req.headers.authorization,
        body,
      })

      if (req.method === 'POST' && req.url === '/api/v1/extension/watch/start') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { started: true } }))
        return
      }

      if (req.method === 'POST' && req.url === '/api/v1/extension/watch/heartbeat') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { points_earned: 2 } }))
        return
      }

      if (req.method === 'GET' && req.url === '/api/v1/users/me/points?channel_id=channel-123') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { spendable_balance: 42 } }))
        return
      }

      res.writeHead(404, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ success: false, error: 'not found' }))
    },
    async (baseUrl, requests) => {
      const originalBaseUrl = process.env.VITE_TACHIGO_API_URL
      process.env.VITE_TACHIGO_API_URL = baseUrl

      try {
        vi.resetModules()
        const api = await import('./api.ts')

        api.setAuthToken('tachigo-access-token')
        const result = await api.sendHeartbeat('channel-123', 40)

        assert.deepEqual(result, { balance: 42, cumulativeTotal: null })
        assert.deepEqual(
          requests.map(({ method, url }) => ({ method, url })),
          [
            { method: 'POST', url: '/api/v1/extension/watch/start' },
            { method: 'POST', url: '/api/v1/extension/watch/heartbeat' },
            { method: 'GET', url: '/api/v1/users/me/points?channel_id=channel-123' },
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

test('sendClick ensures the watch session exists before sending click rewards', async () => {
  await withApiServer(
    (requests) => async (req, res) => {
      const body = await readJsonBody(req)
      requests.push({
        method: req.method ?? 'GET',
        url: req.url ?? '/',
        authorization: req.headers.authorization,
        body,
      })

      if (req.method === 'POST' && req.url === '/api/v1/extension/watch/start') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { started: true } }))
        return
      }

      if (req.method === 'POST' && req.url === '/api/v1/extension/watch/click') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { balance: 9, delta: 1 } }))
        return
      }

      res.writeHead(404, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ success: false, error: 'not found' }))
    },
    async (baseUrl, requests) => {
      const originalBaseUrl = process.env.VITE_TACHIGO_API_URL
      process.env.VITE_TACHIGO_API_URL = baseUrl

      try {
        vi.resetModules()
        const api = await import('./api.ts')

        api.setAuthToken('tachigo-access-token')
        const result = await api.sendClick('channel-123')

        assert.deepEqual(result, { balance: 9, delta: 1 })
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
              url: '/api/v1/extension/watch/start',
              authorization: 'Bearer tachigo-access-token',
              body: { channel_id: 'channel-123' },
            },
            {
              method: 'POST',
              url: '/api/v1/extension/watch/click',
              authorization: 'Bearer tachigo-access-token',
              body: { channel_id: 'channel-123' },
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

test('claimPoints claims viewer points then refreshes tachi balance', async () => {
  await withApiServer(
    (requests) => async (req, res) => {
      const body = await readJsonBody(req)
      requests.push({
        method: req.method ?? 'GET',
        url: req.url ?? '/',
        authorization: req.headers.authorization,
        body,
      })

      if (req.method === 'POST' && req.url === '/api/v1/users/me/points/claim') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { tachi_balance: 12 } }))
        return
      }

      if (req.method === 'GET' && req.url === '/api/v1/users/me/tachi/balance') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { tachi_balance: 12 } }))
        return
      }

      res.writeHead(404, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ success: false, error: 'not found' }))
    },
    async (baseUrl, requests) => {
      const originalBaseUrl = process.env.VITE_TACHIGO_API_URL
      process.env.VITE_TACHIGO_API_URL = baseUrl

      try {
        vi.resetModules()
        const api = await import('./api.ts')

        api.setAuthToken('tachigo-access-token')
        const claimed = await api.claimPoints()
        const balance = await api.getTachiBalance()

        assert.deepEqual(claimed, { tachiBalance: 12 })
        assert.equal(balance, 12)
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
              url: '/api/v1/users/me/points/claim',
              authorization: 'Bearer tachigo-access-token',
              body: { amount: 0 },
            },
            {
              method: 'GET',
              url: '/api/v1/users/me/tachi/balance',
              authorization: 'Bearer tachigo-access-token',
              body: null,
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

test('getCurrentAccount fetches the current account profile through auth recovery', async () => {
  let accountReads = 0

  await withApiServer(
    (requests) => async (req, res) => {
      const body = await readJsonBody(req)
      requests.push({
        method: req.method ?? 'GET',
        url: req.url ?? '/',
        authorization: req.headers.authorization,
        body,
      })

      if (req.method === 'POST' && req.url === '/api/v1/extension/auth/login') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { tokens: { access_token: 'refreshed-access-token' } } }))
        return
      }

      if (req.method === 'GET' && req.url === '/api/v1/users/me') {
        accountReads += 1
        if (accountReads === 1) {
          res.writeHead(401, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ success: false, error: 'expired' }))
          return
        }

        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(
          JSON.stringify({
            success: true,
            data: {
              user: {
                id: 'user-1',
                username: 'mika',
                email: 'mika@example.com',
                role: 'streamer',
                is_active: true,
                email_verified: false,
              },
            },
          }),
        )
        return
      }

      res.writeHead(404, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ success: false, error: 'not found' }))
    },
    async (baseUrl, requests) => {
      const originalBaseUrl = process.env.VITE_TACHIGO_API_URL
      process.env.VITE_TACHIGO_API_URL = baseUrl

      try {
        vi.resetModules()
        const api = await import('./api.ts')

        api.setExtensionJwtForRecovery('extension-jwt')
        api.setAuthToken('expired-access-token')

        assert.deepEqual(await api.getCurrentAccount(), {
          id: 'user-1',
          username: 'mika',
          email: 'mika@example.com',
          role: 'streamer',
          isActive: true,
          emailVerified: false,
        })
        assert.deepEqual(
          requests.map(({ method, url, authorization, body }) => ({
            method,
            url,
            authorization,
            body,
          })),
          [
            {
              method: 'GET',
              url: '/api/v1/users/me',
              authorization: 'Bearer expired-access-token',
              body: null,
            },
            {
              method: 'POST',
              url: '/api/v1/extension/auth/login',
              authorization: 'Bearer expired-access-token',
              body: { extension_jwt: 'extension-jwt' },
            },
            {
              method: 'GET',
              url: '/api/v1/users/me',
              authorization: 'Bearer refreshed-access-token',
              body: null,
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

test('redeemCoupon refreshes the Tachigo token without sending the extension JWT as bearer auth', async () => {
  let redeemAttempts = 0

  await withApiServer(
    (requests) => async (req, res) => {
      const body = await readJsonBody(req)
      requests.push({
        method: req.method ?? 'GET',
        url: req.url ?? '/',
        authorization: req.headers.authorization,
        body,
      })

      if (req.method === 'POST' && req.url === '/api/v1/extension/auth/login') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { tokens: { access_token: 'tachigo-access-token' } } }))
        return
      }

      if (req.method === 'POST' && req.url === '/spend/redeem') {
        redeemAttempts += 1
        if (redeemAttempts === 1) {
          res.writeHead(401, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ success: false, error: 'missing tachigo token' }))
          return
        }

        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { balance: 90, voucher_code: 'ABC' } }))
        return
      }

      res.writeHead(404, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ success: false, error: 'not found' }))
    },
    async (baseUrl, requests) => {
      const originalBaseUrl = process.env.VITE_TACHIGO_API_URL
      process.env.VITE_TACHIGO_API_URL = baseUrl

      try {
        vi.resetModules()
        const api = await import('./api.ts')

        api.setExtensionJwtForRecovery('extension-jwt')
        const result = await api.redeemCoupon('tachiya95', 10)

        assert.deepEqual(result, { balance: 90, voucher_code: 'ABC' })
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
              authorization: undefined,
              body: { coupon_id: 'tachiya95', amount: 10 },
            },
            {
              method: 'POST',
              url: '/api/v1/extension/auth/login',
              authorization: undefined,
              body: { extension_jwt: 'extension-jwt' },
            },
            {
              method: 'POST',
              url: '/spend/redeem',
              authorization: 'Bearer tachigo-access-token',
              body: { coupon_id: 'tachiya95', amount: 10 },
            },
          ],
        )
        assert.equal(
          requests.some(({ authorization }) => authorization === 'Bearer extension-jwt'),
          false,
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

test('sendHeartbeat re-authenticates after 401 and falls back to previous balance when balance read fails', async () => {
  let heartbeatAttempts = 0

  await withApiServer(
    (requests) => async (req, res) => {
      const body = await readJsonBody(req)
      requests.push({
        method: req.method ?? 'GET',
        url: req.url ?? '/',
        authorization: req.headers.authorization,
        body,
      })

      if (req.method === 'POST' && req.url === '/api/v1/extension/watch/start') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { started: true } }))
        return
      }

      if (req.method === 'POST' && req.url === '/api/v1/extension/auth/login') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { tokens: { access_token: 'refreshed-access-token' } } }))
        return
      }

      if (req.method === 'POST' && req.url === '/api/v1/extension/watch/heartbeat') {
        heartbeatAttempts += 1
        if (heartbeatAttempts === 1) {
          res.writeHead(401, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ success: false, error: 'expired' }))
          return
        }

        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { points_earned: 2 } }))
        return
      }

      if (req.method === 'GET' && req.url === '/api/v1/users/me/points?channel_id=channel-123') {
        res.writeHead(503, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: false, error: 'temporary unavailable' }))
        return
      }

      res.writeHead(404, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ success: false, error: 'not found' }))
    },
    async (baseUrl, requests) => {
      const originalBaseUrl = process.env.VITE_TACHIGO_API_URL
      process.env.VITE_TACHIGO_API_URL = baseUrl

      try {
        vi.resetModules()
        const api = await import('./api.ts')

        api.setExtensionJwtForRecovery('extension-jwt')
        api.setAuthToken('expired-access-token')
        const result = await api.sendHeartbeat('channel-123', 40)

        assert.deepEqual(result, { balance: 42, cumulativeTotal: null })
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
              url: '/api/v1/extension/watch/start',
              authorization: 'Bearer expired-access-token',
              body: { channel_id: 'channel-123' },
            },
            {
              method: 'POST',
              url: '/api/v1/extension/watch/heartbeat',
              authorization: 'Bearer expired-access-token',
              body: { channel_id: 'channel-123' },
            },
            {
              method: 'POST',
              url: '/api/v1/extension/auth/login',
              authorization: 'Bearer expired-access-token',
              body: { extension_jwt: 'extension-jwt' },
            },
            {
              method: 'POST',
              url: '/api/v1/extension/watch/heartbeat',
              authorization: 'Bearer refreshed-access-token',
              body: { channel_id: 'channel-123' },
            },
            {
              method: 'GET',
              url: '/api/v1/users/me/points?channel_id=channel-123',
              authorization: 'Bearer refreshed-access-token',
              body: null,
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

test('concurrent 401 responses share one extension JWT refresh', async () => {
  let expiredPointReads = 0
  let loginAttempts = 0

  async function waitForConcurrent401s(timeoutMs = 1000) {
    const deadline = Date.now() + timeoutMs

    while (expiredPointReads < 2) {
      if (Date.now() > deadline) {
        throw new Error(`Timed out waiting for concurrent 401s; expiredPointReads=${expiredPointReads}`)
      }

      await new Promise((resolve) => setTimeout(resolve, 0))
    }
  }

  await withApiServer(
    (requests) => async (req, res) => {
      const body = await readJsonBody(req)
      requests.push({
        method: req.method ?? 'GET',
        url: req.url ?? '/',
        authorization: req.headers.authorization,
        body,
      })

      if (req.method === 'POST' && req.url === '/api/v1/extension/auth/login') {
        loginAttempts += 1
        await waitForConcurrent401s()
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: true, data: { tokens: { access_token: 'refreshed-access-token' } } }))
        return
      }

      if (req.method === 'GET' && req.url === '/api/v1/users/me/points?channel_id=channel-123') {
        if (req.headers.authorization === 'Bearer expired-access-token') {
          expiredPointReads += 1
          res.writeHead(401, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ success: false, error: 'expired' }))
          return
        }

        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(
          JSON.stringify({
            success: true,
            data: { spendable_balance: 42, cumulative_total: 77 },
          }),
        )
        return
      }

      res.writeHead(404, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ success: false, error: 'not found' }))
    },
    async (baseUrl, requests) => {
      const originalBaseUrl = process.env.VITE_TACHIGO_API_URL
      process.env.VITE_TACHIGO_API_URL = baseUrl

      try {
        vi.resetModules()
        const api = await import('./api.ts')

        api.setExtensionJwtForRecovery('extension-jwt')
        api.setAuthToken('expired-access-token')

        const [firstResult, secondResult] = await Promise.all([
          api.getPointBalance('channel-123'),
          api.getPointBalance('channel-123'),
        ])

        assert.deepEqual(firstResult, { spendableBalance: 42, cumulativeTotal: 77 })
        assert.deepEqual(secondResult, { spendableBalance: 42, cumulativeTotal: 77 })
        assert.equal(expiredPointReads, 2)
        assert.equal(loginAttempts, 1)
        const pointReads = requests.filter(
          ({ method, url }) =>
            method === 'GET' && url === '/api/v1/users/me/points?channel_id=channel-123',
        )
        const loginRequests = requests.filter(
          ({ method, url }) =>
            method === 'POST' && url === '/api/v1/extension/auth/login',
        )

        assert.equal(loginRequests.length, 1)
        assert.deepEqual(loginRequests[0]?.body, { extension_jwt: 'extension-jwt' })
        assert.equal(
          pointReads.filter(({ authorization }) => authorization === 'Bearer expired-access-token').length,
          2,
        )
        assert.equal(
          pointReads.filter(({ authorization }) => authorization === 'Bearer refreshed-access-token').length,
          2,
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

test('401 recovery starts a new refresh when the extension JWT changes while an old refresh is in flight', async () => {
  let oldLoginStartedResolve!: () => void
  let releaseOldLogin!: () => void
  const oldLoginStarted = new Promise<void>((resolve) => {
    oldLoginStartedResolve = resolve
  })
  const oldLoginRelease = new Promise<void>((resolve) => {
    releaseOldLogin = resolve
  })

  async function waitForNewJwtLogin(requests: RecordedRequest[], timeoutMs = 1000) {
    const deadline = Date.now() + timeoutMs

    while (
      !requests.some(
        ({ method, url, body }) =>
          method === 'POST' &&
          url === '/api/v1/extension/auth/login' &&
          JSON.stringify(body) === JSON.stringify({ extension_jwt: 'new-extension-jwt' }),
      )
    ) {
      if (Date.now() > deadline) {
        throw new Error('Timed out waiting for new-extension-jwt login')
      }

      await new Promise((resolve) => setTimeout(resolve, 0))
    }
  }

  await withApiServer(
    (requests) => async (req, res) => {
      const body = await readJsonBody(req)
      requests.push({
        method: req.method ?? 'GET',
        url: req.url ?? '/',
        authorization: req.headers.authorization,
        body,
      })

      if (req.method === 'POST' && req.url === '/api/v1/extension/auth/login') {
        if (JSON.stringify(body) === JSON.stringify({ extension_jwt: 'old-extension-jwt' })) {
          oldLoginStartedResolve()
          await oldLoginRelease
          res.writeHead(200, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ success: true, data: { tokens: { access_token: 'old-refreshed-token' } } }))
          return
        }

        if (JSON.stringify(body) === JSON.stringify({ extension_jwt: 'new-extension-jwt' })) {
          res.writeHead(200, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ success: true, data: { tokens: { access_token: 'new-refreshed-token' } } }))
          return
        }
      }

      if (req.method === 'GET' && req.url === '/api/v1/users/me/points?channel_id=old-channel') {
        if (req.headers.authorization === 'Bearer expired-old-token') {
          res.writeHead(401, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ success: false, error: 'expired' }))
          return
        }

        if (req.headers.authorization === 'Bearer old-refreshed-token') {
          res.writeHead(200, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ success: true, data: { spendable_balance: 10, cumulative_total: 11 } }))
          return
        }
      }

      if (req.method === 'GET' && req.url === '/api/v1/users/me/points?channel_id=new-channel') {
        if (req.headers.authorization === 'Bearer expired-new-token') {
          res.writeHead(401, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ success: false, error: 'expired' }))
          return
        }

        if (req.headers.authorization === 'Bearer new-refreshed-token') {
          res.writeHead(200, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ success: true, data: { spendable_balance: 20, cumulative_total: 21 } }))
          return
        }
      }

      if (req.method === 'GET' && req.url === '/api/v1/users/me/points?channel_id=default-channel') {
        if (req.headers.authorization === 'Bearer new-refreshed-token') {
          res.writeHead(200, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ success: true, data: { spendable_balance: 30, cumulative_total: 31 } }))
          return
        }

        res.writeHead(401, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({ success: false, error: 'stale token' }))
        return
      }

      res.writeHead(404, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ success: false, error: 'not found' }))
    },
    async (baseUrl, requests) => {
      const originalBaseUrl = process.env.VITE_TACHIGO_API_URL
      process.env.VITE_TACHIGO_API_URL = baseUrl

      try {
        vi.resetModules()
        const api = await import('./api.ts')

        api.setExtensionJwtForRecovery('old-extension-jwt')
        api.setAuthToken('expired-old-token')
        const oldRequest = api.getPointBalance('old-channel')

        await oldLoginStarted
        api.setExtensionJwtForRecovery('new-extension-jwt')
        api.setAuthToken('expired-new-token')
        const newRequest = api.getPointBalance('new-channel')

        let newJwtLoginError: unknown
        try {
          await waitForNewJwtLogin(requests)
        } catch (error) {
          newJwtLoginError = error
        } finally {
          releaseOldLogin()
        }

        const [oldResult, newResult] = await Promise.allSettled([oldRequest, newRequest])
        if (newJwtLoginError) {
          throw newJwtLoginError
        }

        assert.deepEqual(oldResult, {
          status: 'fulfilled',
          value: { spendableBalance: 10, cumulativeTotal: 11 },
        })
        assert.deepEqual(newResult, {
          status: 'fulfilled',
          value: { spendableBalance: 20, cumulativeTotal: 21 },
        })
        assert.deepEqual(await api.getPointBalance('default-channel'), {
          spendableBalance: 30,
          cumulativeTotal: 31,
        })

        const loginRequests = requests.filter(
          ({ method, url }) =>
            method === 'POST' && url === '/api/v1/extension/auth/login',
        )
        assert.deepEqual(
          loginRequests.map(({ body }) => body),
          [{ extension_jwt: 'old-extension-jwt' }, { extension_jwt: 'new-extension-jwt' }],
        )
        assert.equal(
          requests.some(
            ({ method, url, authorization }) =>
              method === 'GET' &&
              url === '/api/v1/users/me/points?channel_id=new-channel' &&
              authorization === 'Bearer new-refreshed-token',
          ),
          true,
        )
      } finally {
        releaseOldLogin()
        if (originalBaseUrl === undefined) {
          delete process.env.VITE_TACHIGO_API_URL
        } else {
          process.env.VITE_TACHIGO_API_URL = originalBaseUrl
        }
      }
    },
  )
})
