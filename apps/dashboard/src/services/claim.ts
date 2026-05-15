import client from '@/services/api'

type ApiResponse<T> = { success: boolean; data: T }

export interface ClaimEntry {
  id: string
  raffle_id: string
  user_id: string | null
  twitch_login: string
  display_name: string
  created_at: string
}

export interface ClaimDraw {
  id: string
  raffle_id: string
  entry_id: string
  claim_expires_at: string
  drawn_at: string
  entry: ClaimEntry
}

export interface ClaimInput {
  recipient_name: string
  phone: string
  address_line1: string
  address_line2: string
  city: string
  postal_code: string
  country: string
}

export async function getClaim(token: string): Promise<ClaimDraw> {
  const { data } = await client.get<ApiResponse<{ draw: ClaimDraw }>>(
    `/api/v1/claim/${token}`,
  )
  return data.data.draw
}

export async function submitClaim(token: string, input: ClaimInput): Promise<void> {
  await client.post(`/api/v1/claim/${token}`, input)
}
