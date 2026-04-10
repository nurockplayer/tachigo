import axios from 'axios'
import type { TachigoToken } from '../types/twitch'

const processEnv =
  typeof globalThis === 'object' && 'process' in globalThis
    ? (
        globalThis as {
          process?: {
            env?: Record<string, string | undefined>
          }
        }
      ).process?.env
    : undefined

const BASE_URL =
  import.meta.env?.VITE_TACHIGO_API_URL ??
  processEnv?.VITE_TACHIGO_API_URL ??
  'http://localhost:8080'

const client = axios.create({
  baseURL: BASE_URL,
  headers: { 'Content-Type': 'application/json' },
})

export function setAuthToken(token: string) {
  client.defaults.headers.common['Authorization'] = `Bearer ${token}`
}

/**
 * Exchange a Twitch Extension JWT + Bits transaction receipt for a tachigo token.
 */
export async function completeBitsTransaction(
  extensionJwt: string,
  transactionReceipt: string,
  sku: string,
): Promise<TachigoToken> {
  const { data } = await client.post<TachigoToken>('/api/v1/extension/bits/complete', {
    extension_jwt: extensionJwt,
    transaction_receipt: transactionReceipt,
    sku,
  })
  return data
}

/**
 * Login to tachigo via Twitch Extension JWT (viewer identity).
 */
export async function loginWithTwitchExtension(extensionJwt: string): Promise<TachigoToken> {
  const { data } = await client.post<TachigoToken>('/api/v1/extension/auth/login', {
    extension_jwt: extensionJwt,
  })
  return data
}

interface HeartbeatResponse {
  balance: number
}

function parseBalanceFromPayload(payload: unknown): number {
  if (!payload || typeof payload !== 'object') {
    throw new Error('Invalid balance response')
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const raw = payload as any
  const direct = raw.balance
  const nested = raw.data?.balance
  const legacy = raw.points_balance
  const spendable = raw.spendable_balance
  const nestedSpendable = raw.data?.spendable_balance

  const value = [direct, nested, legacy, spendable, nestedSpendable].find(
    (candidate) => typeof candidate === 'number',
  )
  if (typeof value !== 'number') {
    throw new Error('Balance response missing balance')
  }

  return value
}

interface ClickResponse {
  balance: number
  delta: number
}

async function ensureWatchSession(channelId: string) {
  await client.post('/api/v1/extension/watch/start', { channel_id: channelId })
}

async function getWatchBalance(channelId: string): Promise<number> {
  const { data } = await client.get('/api/v1/extension/watch/balance', {
    params: { channel_id: channelId },
  })

  return parseBalanceFromPayload(data)
}

export async function sendClick(channelId: string): Promise<ClickResponse> {
  await ensureWatchSession(channelId)

  const { data } = await client.post<{ success: boolean; data: ClickResponse }>(
    '/api/v1/extension/watch/click',
    { channel_id: channelId },
  )
  return data.data
}

export async function sendHeartbeat(channelId: string): Promise<HeartbeatResponse> {
  await ensureWatchSession(channelId)

  await client.post('/api/v1/extension/watch/heartbeat', {
    channel_id: channelId,
  })

  return {
    balance: await getWatchBalance(channelId),
  }
}
