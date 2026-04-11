import assert from 'node:assert/strict'
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

test('sendHeartbeat starts a watch session, sends heartbeat, then refreshes balance', async () => {
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

      if (req.method === 'GET' && req.url === '/api/v1/extension/watch/balance?channel_id=channel-123') {
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(
          JSON.stringify({
            success: true,
            data: { spendable_balance: 42, cumulative_total: 42 },
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
        const api = await import(`./api.ts?heartbeat=${Date.now()}`)

        api.setAuthToken('tachigo-access-token')
        const result = await api.sendHeartbeat('channel-123')

        assert.equal(result.balance, 42)
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
            {
              method: 'GET',
              url: '/api/v1/extension/watch/balance?channel_id=channel-123',
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
        const api = await import(`./api.ts?click=${Date.now()}`)

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
        const api = await import(`./api.ts?claim=${Date.now()}`)

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

test('claimPoints maps backend claim errors into typed frontend errors', async () => {
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
        const amount = (body as { amount?: number } | null)?.amount

        if (amount === 1) {
          res.writeHead(422, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ success: false, error: 'insufficient spendable balance to claim' }))
          return
        }

        if (amount === 2) {
          res.writeHead(422, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ success: false, error: 'web3 wallet not linked' }))
          return
        }

        if (amount === 3) {
          res.writeHead(500, { 'Content-Type': 'application/json' })
          res.end(JSON.stringify({ success: false, error: 'claim_contract_config_error' }))
          return
        }
      }

      res.writeHead(404, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ success: false, error: 'not found' }))
    },
    async (baseUrl) => {
      const originalBaseUrl = process.env.VITE_TACHIGO_API_URL
      process.env.VITE_TACHIGO_API_URL = baseUrl

      try {
        const api = await import(`./api.ts?claim-errors=${Date.now()}`)

        api.setAuthToken('tachigo-access-token')

        await assert.rejects(
          () => api.claimPoints(1),
          (error: { code?: string }) => error.code === 'insufficientBalance',
        )

        await assert.rejects(
          () => api.claimPoints(2),
          (error: { code?: string }) => error.code === 'walletNotLinked',
        )

        await assert.rejects(
          () => api.claimPoints(3),
          (error: { code?: string }) => error.code === 'contractConfig',
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

      if (req.method === 'GET' && req.url === '/api/v1/extension/watch/balance?channel_id=channel-123') {
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
        const api = await import(`./api.ts?recover=${Date.now()}`)

        api.setExtensionJwtForRecovery('extension-jwt')
        api.setAuthToken('expired-access-token')
        const result = await api.sendHeartbeat('channel-123', 40)

        assert.equal(result.balance, 42)
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
              url: '/api/v1/extension/watch/balance?channel_id=channel-123',
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
