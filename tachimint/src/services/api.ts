import axios from 'axios'
import type { AxiosError, AxiosRequestConfig } from 'axios'
import type { TachigoToken } from '../types/twitch'
import type { RaffleResultDraw } from '../extension/types'

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

let extensionJwtForRecovery: string | null = null

function extractAccessToken(payload: unknown): string | null {
  if (!payload || typeof payload !== 'object') {
    return null
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const raw = payload as any
  const nested = raw?.data?.tokens?.access_token
  const direct = raw?.tokens?.access_token
  return typeof nested === 'string' ? nested : typeof direct === 'string' ? direct : null
}

export function setAuthToken(token: string) {
  client.defaults.headers.common['Authorization'] = `Bearer ${token}`
}

export function clearAuthToken() {
  delete client.defaults.headers.common['Authorization']
}

export function setExtensionJwtForRecovery(token: string | null) {
  extensionJwtForRecovery = token
}

/**
 * Exchange a Twitch Extension JWT + transaction receipt for a tachigo token.
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

async function refreshAuthTokenFromExtensionJwt(): Promise<boolean> {
  if (!extensionJwtForRecovery) {
    return false
  }

  const loginResult = await loginWithTwitchExtension(extensionJwtForRecovery)
  const accessToken = extractAccessToken(loginResult)
  if (!accessToken) {
    clearAuthToken()
    return false
  }

  setAuthToken(accessToken)
  return true
}

async function runWithAuthRecovery<T>(
  execute: (config?: AxiosRequestConfig) => Promise<T>,
  config?: AxiosRequestConfig,
): Promise<T> {
  try {
    return await execute(config)
  } catch (error) {
    const status = (error as AxiosError)?.response?.status
    if (status !== 401) {
      throw error
    }

    const recovered = await refreshAuthTokenFromExtensionJwt()
    if (!recovered) {
      throw error
    }

    return execute(config)
  }
}

interface HeartbeatResponse {
  balance: number
}

interface TachiBalanceResponse {
  tachiBalance: number
}

export interface RedeemCouponResponse {
  balance: number
  voucher_code: string
}

function parsePointsEarnedFromPayload(payload: unknown): number | null {
  if (!payload || typeof payload !== 'object') {
    return null
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const raw = payload as any
  const direct = raw.points_earned
  const nested = raw.data?.points_earned
  const value = typeof direct === 'number' ? direct : typeof nested === 'number' ? nested : null
  return value
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

function parseTachiBalanceFromPayload(payload: unknown): number {
  if (!payload || typeof payload !== 'object') {
    throw new Error('Invalid tachi balance response')
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const raw = payload as any
  const direct = raw.tachi_balance
  const nested = raw.data?.tachi_balance
  const value = typeof direct === 'number' ? direct : typeof nested === 'number' ? nested : null
  if (typeof value !== 'number') {
    throw new Error('Tachi balance response missing tachi_balance')
  }

  return value
}

interface ClickResponse {
  balance: number
  delta: number
}

async function ensureWatchSession(channelId: string) {
  await runWithAuthRecovery((config) => client.post('/api/v1/extension/watch/start', { channel_id: channelId }, config))
}

async function getWatchBalance(channelId: string): Promise<number> {
  const { data } = await runWithAuthRecovery((config) => client.get('/api/v1/extension/watch/balance', {
    ...config,
    params: { channel_id: channelId },
  }))

  return parseBalanceFromPayload(data)
}

export async function sendClick(channelId: string): Promise<ClickResponse> {
  await ensureWatchSession(channelId)

  const { data } = await runWithAuthRecovery((config) =>
    client.post<{ success: boolean; data: ClickResponse }>(
      '/api/v1/extension/watch/click',
      { channel_id: channelId },
      config,
    ))
  return data.data
}

export async function getTachiBalance(): Promise<number> {
  const { data } = await runWithAuthRecovery((config) => client.get('/api/v1/users/me/tachi/balance', config))

  return parseTachiBalanceFromPayload(data)
}

export async function claimPoints(amount = 0): Promise<TachiBalanceResponse> {
  const { data } = await runWithAuthRecovery((config) =>
    client.post('/api/v1/users/me/points/claim', { amount }, config))

  return {
    tachiBalance: parseTachiBalanceFromPayload(data),
  }
}

export async function redeemCoupon(
  couponId: string,
  amount: number,
  token: string,
): Promise<RedeemCouponResponse> {
  try {
    const { data } = await client.post<{ success: boolean; data: RedeemCouponResponse }>(
      '/spend/redeem',
      { coupon_id: couponId, amount },
      {
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      },
    )
    return data.data
  } catch (error) {
    if (axios.isAxiosError(error)) {
      const message =
        typeof error.response?.data === 'object' && error.response?.data && 'error' in error.response.data
          ? String(error.response.data.error)
          : error.message
      throw new Error(`Failed to redeem coupon${error.response?.status ? ` (${error.response.status})` : ''}: ${message}`, { cause: error })
    }

    throw error instanceof Error ? error : new Error('Failed to redeem coupon')
  }
}

export async function sendHeartbeat(
  channelId: string,
  previousBalance?: number | null,
): Promise<HeartbeatResponse> {
  await ensureWatchSession(channelId)

  const heartbeatResponse = await runWithAuthRecovery((config) =>
    client.post('/api/v1/extension/watch/heartbeat', {
      channel_id: channelId,
    }, config))

  try {
    return {
      balance: await getWatchBalance(channelId),
    }
  } catch {
    const pointsEarned = parsePointsEarnedFromPayload(heartbeatResponse.data)
    if (typeof previousBalance === 'number') {
      return {
        balance: previousBalance + Math.max(pointsEarned ?? 0, 0),
      }
    }

    return {
      balance: Math.max(pointsEarned ?? 0, 0),
    }
  }
}

export async function getRaffleResult(raffleId: string): Promise<RaffleResultDraw[]> {
  const { data } = await client.get<{ success: boolean; data: { draws: RaffleResultDraw[] } }>(
    `/api/v1/extension/raffles/${raffleId}/result`,
  )
  return data.data.draws
}
