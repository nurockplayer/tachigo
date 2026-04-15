import { beforeEach, describe, expect, it, vi } from 'vitest'
import client from '@/services/api'
import {
  getChannelConfig,
  getMyChannels,
  getStreamerStats,
  getStreamers,
} from '@/services/channels'

vi.mock('@/services/api', () => ({
  default: {
    get: vi.fn(),
  },
}))

describe('channels service', () => {
  beforeEach(() => {
    vi.mocked(client.get).mockReset()
  })

  it('查詢 streamers 列表並回傳資料', async () => {
    const streamers = [
      { id: 'uuid-1', user_id: 'user-1', channel_id: 'channel-1', display_name: 'Alice' },
    ]

    vi.mocked(client.get).mockResolvedValueOnce({
      data: {
        data: {
          streamers,
        },
      },
    })

    const result = await getStreamers()

    expect(client.get).toHaveBeenCalledWith('/api/v1/dashboard/streamers')
    expect(result).toEqual(streamers)
  })

  it('查詢我的 channels 並回傳資料', async () => {
    const channels = [
      { id: 'uuid-1', user_id: 'user-1', channel_id: 'channel-1', display_name: 'Alice' },
    ]

    vi.mocked(client.get).mockResolvedValueOnce({
      data: {
        data: {
          channels,
        },
      },
    })

    const result = await getMyChannels()

    expect(client.get).toHaveBeenCalledWith('/api/v1/dashboard/streamers/channels')
    expect(result).toEqual(channels)
  })

  it('對 streamerId 做 URL encoding 後再查詢 stats', async () => {
    const stats = {
      current_session_seconds: 0,
      daily_seconds: 0,
      monthly_seconds: 0,
      yearly_seconds: 0,
      unique_miners: 0,
      avg_session_seconds: 0,
      total_token_minted: 0,
      spendable_in_circulation: 0,
    }

    vi.mocked(client.get).mockResolvedValue({
      data: {
        data: {
          stats,
          channel_id: 'channel-1',
        },
      },
    })

    const result = await getStreamerStats('streamer/id')

    expect(client.get).toHaveBeenCalledWith('/api/v1/dashboard/streamers/streamer%2Fid/stats')
    expect(result).toEqual({
      stats,
      channelId: 'channel-1',
    })
  })

  it('對 channelId 做 URL encoding 後再查詢 config', async () => {
    const config = {
      channel_id: 'channel/1',
      seconds_per_point: 60,
      multiplier: 2,
    }

    vi.mocked(client.get).mockResolvedValue({
      data: {
        data: {
          config,
        },
      },
    })

    const result = await getChannelConfig('channel/1')

    expect(client.get).toHaveBeenCalledWith('/api/v1/dashboard/channels/channel%2F1/config')
    expect(result).toEqual(config)
  })
})
