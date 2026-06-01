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
  processEnv?.VITE_TACHIGO_API_URL

if (!BASE_URL) {
  throw new Error('Missing VITE_TACHIGO_API_URL for Tachigo API client.')
}

const client = axios.create({
  baseURL: BASE_URL,
  headers: { 'Content-Type': 'application/json' },
})

let extensionJwtForRecovery: string | null = null
const authRecoveryRefreshes = new Map<string, Promise<string | null>>()

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
export async function completeTPointTransaction(
  extensionJwt: string,
  transactionReceipt: string,
  sku: string,
): Promise<TachigoToken> {
  const { data } = await client.post<TachigoToken>('/api/v1/extension/t-point/complete', {
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

async function performAuthTokenRefreshFromExtensionJwt(extensionJwt: string): Promise<string | null> {
  const loginResult = await loginWithTwitchExtension(extensionJwt)
  const accessToken = extractAccessToken(loginResult)
  if (!accessToken) {
    if (extensionJwtForRecovery === extensionJwt) {
      clearAuthToken()
    }
    return null
  }

  if (extensionJwtForRecovery === extensionJwt) {
    setAuthToken(accessToken)
  }
  return accessToken
}

async function refreshAuthTokenFromExtensionJwt(): Promise<string | null> {
  const extensionJwt = extensionJwtForRecovery
  if (!extensionJwt) {
    return null
  }

  let refresh = authRecoveryRefreshes.get(extensionJwt)
  if (!refresh) {
    refresh = performAuthTokenRefreshFromExtensionJwt(extensionJwt).finally(() => {
      if (authRecoveryRefreshes.get(extensionJwt) === refresh) {
        authRecoveryRefreshes.delete(extensionJwt)
      }
    })
    authRecoveryRefreshes.set(extensionJwt, refresh)
  }

  return refresh
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

    const recoveredToken = await refreshAuthTokenFromExtensionJwt()
    if (!recoveredToken) {
      throw error
    }

    return execute({
      ...config,
      headers: {
        ...config?.headers,
        Authorization: `Bearer ${recoveredToken}`,
      },
    })
  }
}

interface HeartbeatResponse {
  balance: number
  cumulativeTotal: number | null
}

interface TachiBalanceResponse {
  tachiBalance: number
}

interface PointBalanceResponse {
  spendableBalance: number
  cumulativeTotal: number
}

export interface CurrentAccount {
  id: string
  username: string | null
  email: string | null
  role: string
  isActive: boolean | null
  emailVerified: boolean | null
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

function parsePointBalanceFromPayload(payload: unknown): PointBalanceResponse {
  if (!payload || typeof payload !== 'object') {
    throw new Error('Invalid point balance response')
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const raw = payload as any
  const legacy = raw.points_balance
  const spendable = raw.spendable_balance
  const nestedSpendable = raw.data?.spendable_balance
  const cumulative = raw.cumulative_total
  const nestedCumulative = raw.data?.cumulative_total

  const spendableBalance = [spendable, nestedSpendable, legacy].find(
    (candidate) => typeof candidate === 'number',
  )
  const cumulativeTotal = [cumulative, nestedCumulative].find(
    (candidate) => typeof candidate === 'number',
  )
  if (typeof spendableBalance !== 'number' || typeof cumulativeTotal !== 'number') {
    throw new Error('Point balance response missing cumulative_total or spendable_balance')
  }

  return { spendableBalance, cumulativeTotal }
}

function parseOptionalPointBalanceFromPayload(payload: unknown): PointBalanceResponse | null {
  try {
    return parsePointBalanceFromPayload(payload)
  } catch {
    return null
  }
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

function parseCurrentAccountFromPayload(payload: unknown): CurrentAccount {
  if (!payload || typeof payload !== 'object') {
    throw new Error('Invalid account response')
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const raw = payload as any
  const user = raw.data?.user ?? raw.user ?? raw.data
  if (!user || typeof user !== 'object') {
    throw new Error('Account response missing user')
  }

  const id = user.id
  const role = user.role
  if (typeof id !== 'string' || typeof role !== 'string') {
    throw new Error('Account response missing id or role')
  }

  return {
    id,
    username: typeof user.username === 'string' ? user.username : null,
    email: typeof user.email === 'string' ? user.email : null,
    role,
    isActive: typeof user.is_active === 'boolean' ? user.is_active : null,
    emailVerified: typeof user.email_verified === 'boolean' ? user.email_verified : null,
  }
}

interface ClickResponse {
  balance: number
  delta: number
}

async function ensureWatchSession(channelId: string) {
  await runWithAuthRecovery((config) => client.post('/api/v1/extension/watch/start', { channel_id: channelId }, config))
}

export async function getPointBalance(channelId: string): Promise<PointBalanceResponse> {
  const { data } = await runWithAuthRecovery((config) => client.get('/api/v1/users/me/points', {
    ...config,
    params: { channel_id: channelId },
  }))

  return parsePointBalanceFromPayload(data)
}

export async function getCurrentAccount(): Promise<CurrentAccount> {
  const { data } = await runWithAuthRecovery((config) => client.get('/api/v1/users/me', config))

  return parseCurrentAccountFromPayload(data)
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
    const { data } = await runWithAuthRecovery((config) =>
      client.post<{ success: boolean; data: RedeemCouponResponse }>(
        '/spend/redeem',
        { coupon_id: couponId, amount },
        {
          ...config,
          headers: client.defaults.headers.common.Authorization
            ? config?.headers
            : {
                ...config?.headers,
                Authorization: `Bearer ${token}`,
              },
        },
      ),
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

  const heartbeatBalance = parseOptionalPointBalanceFromPayload(heartbeatResponse.data)
  if (heartbeatBalance) {
    return {
      balance: heartbeatBalance.spendableBalance,
      cumulativeTotal: heartbeatBalance.cumulativeTotal,
    }
  }

  try {
    const pointBalance = await getPointBalance(channelId)
    return {
      balance: pointBalance.spendableBalance,
      cumulativeTotal: pointBalance.cumulativeTotal,
    }
  } catch {
    const pointsEarned = parsePointsEarnedFromPayload(heartbeatResponse.data)
    if (typeof previousBalance === 'number') {
      return {
        balance: previousBalance + Math.max(pointsEarned ?? 0, 0),
        cumulativeTotal: null,
      }
    }

    return {
      balance: Math.max(pointsEarned ?? 0, 0),
      cumulativeTotal: null,
    }
  }
}

export async function getRaffleResult(raffleId: string): Promise<RaffleResultDraw[]> {
  const { data } = await client.get<{ success: boolean; data: { draws: RaffleResultDraw[] } }>(
    `/api/v1/extension/raffles/${raffleId}/result`,
  )
  return data.data.draws
}
