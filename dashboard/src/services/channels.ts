import client from '@/services/api'

type ApiResponse<T> = {
  success: boolean
  data: T
}

export interface ChannelListItem {
  id: string
  channel_id: string
  display_name: string
  daily_seconds?: number
  unique_miners?: number
  total_token_minted?: number
}

export interface ChannelStats {
  current_session_seconds: number
  daily_seconds: number
  monthly_seconds: number
  yearly_seconds: number
  unique_miners?: number
  avg_session_seconds?: number
  total_token_minted?: number
}

export interface ChannelConfig {
  channel_id: string
  seconds_per_point: number
  multiplier?: number
}

export async function getStreamerChannels(): Promise<ChannelListItem[]> {
  const { data } = await client.get<ApiResponse<{ channels: ChannelListItem[] }>>(
    '/api/v1/dashboard/streamers/channels',
  )
  return data.data.channels
}

export async function getChannelStats(channelId: string): Promise<ChannelStats> {
  const { data } = await client.get<ApiResponse<{ stats: ChannelStats }>>(
    `/api/v1/dashboard/channels/${channelId}/stats`,
  )
  return data.data.stats
}

export async function getChannelConfig(channelId: string): Promise<ChannelConfig> {
  const { data } = await client.get<ApiResponse<{ config: ChannelConfig }>>(
    `/api/v1/dashboard/channels/${channelId}/config`,
  )
  return data.data.config
}
