import client from '@/services/api'

type ApiResponse<T> = { success: boolean; data: T }

export type RaffleStatus = 'draft' | 'active' | 'completed'

export interface Raffle {
  id: string
  user_id: string
  title: string
  status: RaffleStatus
  created_at: string
  updated_at: string
}

export interface RaffleEntry {
  id: string
  raffle_id: string
  twitch_login: string
  display_name: string
  created_at: string
}

export interface RaffleDraw {
  id: string
  raffle_id: string
  entry_id: string
  claim_token: string
  claim_expires_at: string
  drawn_at: string
  entry: RaffleEntry
}

export async function listRaffles(): Promise<Raffle[]> {
  const { data } = await client.get<ApiResponse<{ raffles: Raffle[] }>>(
    '/api/v1/dashboard/raffles',
  )
  return data.data.raffles
}

export async function createRaffle(title: string): Promise<Raffle> {
  const { data } = await client.post<ApiResponse<{ raffle: Raffle }>>(
    '/api/v1/dashboard/raffles',
    { title },
  )
  return data.data.raffle
}

