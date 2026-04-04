import axios from 'axios'
import type { TachigoToken } from '../types/twitch'

const BASE_URL = import.meta.env.VITE_TACHIGO_API_URL ?? 'http://localhost:8080'

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

function parseBalanceFromHeartbeatResponse(payload: unknown): number {
  if (!payload || typeof payload !== 'object') {
    throw new Error('Invalid heartbeat response')
  }

  // Accept a few common API shapes to keep frontend resilient while backend evolves.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const raw = payload as any
  const direct = raw.balance
  const nested = raw.data?.balance
  const legacy = raw.points_balance

  const value = [direct, nested, legacy].find((candidate) => typeof candidate === 'number')
  if (typeof value !== 'number') {
    throw new Error('Heartbeat response missing balance')
  }

  return value
}

interface ClickResponse {
  pointsEarned: number
  newBalance: number
  cooldownRemainingMs: number
}

export async function sendClick(channelId: string): Promise<ClickResponse> {
  const { data } = await client.post('/api/v1/extension/click', { channel_id: channelId })
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const raw = (data?.data ?? data) as any
  return {
    pointsEarned: raw.points_earned ?? 0,
    newBalance: raw.new_balance ?? 0,
    cooldownRemainingMs: raw.cooldown_remaining_ms ?? 0,
  }
}

export async function sendHeartbeat(extensionJwt: string): Promise<HeartbeatResponse> {
  const { data } = await client.post('/api/v1/extension/heartbeat', {
    extension_jwt: extensionJwt,
  })

  return {
    balance: parseBalanceFromHeartbeatResponse(data),
  }
}
