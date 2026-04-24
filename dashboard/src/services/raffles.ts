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

export async function getRaffle(id: string): Promise<Raffle> {
  const { data } = await client.get<ApiResponse<{ raffle: Raffle }>>(
    `/api/v1/dashboard/raffles/${id}`,
  )
  return data.data.raffle
}

export async function importCSV(raffleId: string, file: File): Promise<number> {
  const form = new FormData()
  form.append('file', file)
  const { data } = await client.post<ApiResponse<{ imported: number }>>(
    `/api/v1/dashboard/raffles/${raffleId}/entries/import-csv`,
    form,
    { headers: { 'Content-Type': 'multipart/form-data' } },
  )
  return data.data.imported
}

export async function drawNext(raffleId: string): Promise<RaffleDraw> {
  const { data } = await client.post<ApiResponse<{ draw: RaffleDraw }>>(
    `/api/v1/dashboard/raffles/${raffleId}/draws`,
  )
  return data.data.draw
}

export async function listDraws(raffleId: string): Promise<RaffleDraw[]> {
  const { data } = await client.get<ApiResponse<{ draws: RaffleDraw[] }>>(
    `/api/v1/dashboard/raffles/${raffleId}/draws`,
  )
  return data.data.draws
}

export async function completeRaffle(raffleId: string): Promise<Raffle> {
  const { data } = await client.post<ApiResponse<{ raffle: Raffle }>>(
    `/api/v1/dashboard/raffles/${raffleId}/complete`,
  )
  return data.data.raffle
}
