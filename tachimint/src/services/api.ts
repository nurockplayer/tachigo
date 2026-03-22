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
