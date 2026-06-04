import client from '@/services/api'

type ApiResponse<T> = { success: boolean; data: T }

export type RaffleStatus = 'draft' | 'active' | 'completed'
export type RaffleMode = 'public' | 'subscribers_only'

export interface Raffle {
  id: string
  user_id: string
  title: string
  status: RaffleStatus
  source?: 'csv' | 'twitch_api'
  mode?: RaffleMode
  entry_open?: boolean
  created_at: string
  updated_at: string
}

export interface RaffleEntryStats {
  eligible_count: number
  ineligible_count: number
  ineligible_reasons: Record<string, number>
  total_joined: number
}

export interface RaffleEntry {
  id: string
  raffle_id: string
  twitch_login: string
  display_name: string
  created_at: string
}

export interface RafflePrizeTier {
  id: string
  raffle_id: string
  name: string
  prize_description: string
  winner_count: number
  drawn_count: number
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
  prize_tier_id?: string
  prize_tier?: RafflePrizeTier
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

export async function listDraws(raffleId: string): Promise<RaffleDraw[]> {
  const { data } = await client.get<ApiResponse<{ draws: RaffleDraw[] }>>(
    `/api/v1/dashboard/raffles/${raffleId}/draws`,
  )
  return data.data.draws
}

export async function drawNext(raffleId: string): Promise<RaffleDraw> {
  const { data } = await client.post<ApiResponse<{ draw: RaffleDraw }>>(
    `/api/v1/dashboard/raffles/${raffleId}/draws`,
    undefined,
  )
  return data.data.draw
}

export async function importCSV(
  raffleId: string,
  file: File,
): Promise<{ imported: number; skipped: number }> {
  const form = new FormData()
  form.append('file', file)
  const { data } = await client.post<ApiResponse<{ imported: number; skipped: number }>>(
    `/api/v1/dashboard/raffles/${raffleId}/entries/import-csv`,
    form,
  )
  return data.data
}

export async function activateRaffle(raffleId: string): Promise<Raffle> {
  const { data } = await client.post<ApiResponse<{ raffle: Raffle }>>(
    `/api/v1/dashboard/raffles/${raffleId}/activate`,
    undefined,
  )
  return data.data.raffle
}

export async function setRaffleMode(
  raffleId: string,
  mode: RaffleMode,
): Promise<Raffle> {
  const { data } = await client.patch<ApiResponse<{ raffle: Raffle }>>(
    `/api/v1/dashboard/raffles/${raffleId}/mode`,
    { mode },
  )
  return data.data.raffle
}

export async function setRaffleEntryOpen(
  raffleId: string,
  entryOpen: boolean,
): Promise<Raffle> {
  const { data } = await client.patch<ApiResponse<{ raffle: Raffle }>>(
    `/api/v1/dashboard/raffles/${raffleId}/entry-open`,
    { entry_open: entryOpen },
  )
  return data.data.raffle
}

export async function getRaffleEntryStats(
  raffleId: string,
  signal?: AbortSignal,
): Promise<RaffleEntryStats> {
  const { data } = await client.get<ApiResponse<{ stats: RaffleEntryStats }>>(
    `/api/v1/dashboard/raffles/${raffleId}/entry-stats`,
    { signal },
  )
  return data.data.stats
}

export async function completeRaffle(raffleId: string): Promise<void> {
  await client.post(
    `/api/v1/dashboard/raffles/${raffleId}/complete`,
    undefined,
  )
}

export async function listPrizeTiers(raffleId: string): Promise<RafflePrizeTier[]> {
  const { data } = await client.get<ApiResponse<{ tiers: RafflePrizeTier[] }>>(
    `/api/v1/dashboard/raffles/${raffleId}/prize-tiers`,
  )
  return data.data.tiers
}

export async function createPrizeTier(
  raffleId: string,
  payload: { name: string; prize_description: string; winner_count: number },
): Promise<RafflePrizeTier> {
  const { data } = await client.post<ApiResponse<{ tier: RafflePrizeTier }>>(
    `/api/v1/dashboard/raffles/${raffleId}/prize-tiers`,
    payload,
  )
  return data.data.tier
}

export async function deletePrizeTier(raffleId: string, tierId: string): Promise<void> {
  await client.delete(`/api/v1/dashboard/raffles/${raffleId}/prize-tiers/${tierId}`)
}

export async function drawFromTier(raffleId: string, tierId: string): Promise<RaffleDraw> {
  const { data } = await client.post<ApiResponse<{ draw: RaffleDraw }>>(
    `/api/v1/dashboard/raffles/${raffleId}/prize-tiers/${tierId}/draws`,
    undefined,
  )
  return data.data.draw
}

export async function setDiscordWebhook(
  raffleId: string,
  webhookUrl: string,
): Promise<boolean> {
  const { data } = await client.patch<ApiResponse<{ discord_webhook_configured: boolean }>>(
    `/api/v1/dashboard/raffles/${raffleId}/discord-webhook`,
    { discord_webhook_url: webhookUrl },
  )
  return data.data.discord_webhook_configured
}
