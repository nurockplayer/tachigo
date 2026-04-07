import client from '@/services/api'

type ApiResponse<T> = {
  success: boolean
  data: T
}

export interface Streamer {
  id: string
  user_id: string
  agency_user_id?: string
  channel_id: string
  display_name: string
}

export interface StreamerStats {
  current_session_seconds: number
  daily_seconds: number
  monthly_seconds: number
  yearly_seconds: number
  unique_miners: number
  avg_session_seconds: number
  total_token_minted: number
  spendable_in_circulation: number
}

export interface ChannelConfig {
  channel_id: string
  seconds_per_point: number
  multiplier: number
}

export interface StreamerStatsResponse {
  stats: StreamerStats
  channelId: string
}

export async function getStreamers(): Promise<Streamer[]> {
  const { data } = await client.get<ApiResponse<{ streamers: Streamer[] }>>(
    '/api/v1/dashboard/streamers',
  )
  return data.data.streamers
}

export async function getStreamerStats(streamerId: string): Promise<StreamerStatsResponse> {
  const { data } = await client.get<ApiResponse<{ stats: StreamerStats; channel_id: string }>>(
    `/api/v1/dashboard/streamers/${streamerId}/stats`,
  )
  return { stats: data.data.stats, channelId: data.data.channel_id }
}

export async function getChannelConfig(channelId: string): Promise<ChannelConfig> {
  const { data } = await client.get<ApiResponse<{ config: ChannelConfig }>>(
    `/api/v1/dashboard/channels/${channelId}/config`,
  )
  return data.data.config
}

export async function updateChannelConfig(
  channelId: string,
  multiplier: number,
): Promise<ChannelConfig> {
  const { data } = await client.put<ApiResponse<{ config: ChannelConfig }>>(
    `/api/v1/dashboard/channels/${channelId}/config`,
    { multiplier },
  )
  return data.data.config
}
