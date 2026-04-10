import { beforeEach, describe, expect, it, vi } from 'vitest'
import client from '@/services/api'
import { getChannelConfig, getStreamerStats } from '@/services/channels'

vi.mock('@/services/api', () => ({
  default: {
    get: vi.fn(),
  },
}))

describe('channels service', () => {
  beforeEach(() => {
    vi.mocked(client.get).mockReset()
  })

  it('對 streamerId 做 URL encoding 後再查詢 stats', async () => {
    vi.mocked(client.get).mockResolvedValue({
      data: {
        data: {
          stats: {
            current_session_seconds: 0,
            daily_seconds: 0,
            monthly_seconds: 0,
            yearly_seconds: 0,
            unique_miners: 0,
            avg_session_seconds: 0,
            total_token_minted: 0,
            spendable_in_circulation: 0,
          },
          channel_id: 'channel-1',
        },
      },
    })

    await getStreamerStats('streamer/id')

    expect(client.get).toHaveBeenCalledWith('/api/v1/dashboard/streamers/streamer%2Fid/stats')
  })

  it('對 channelId 做 URL encoding 後再查詢 config', async () => {
    vi.mocked(client.get).mockResolvedValue({
      data: {
        data: {
          config: {
            channel_id: 'channel/1',
            seconds_per_point: 60,
            multiplier: 2,
          },
        },
      },
    })

    await getChannelConfig('channel/1')

    expect(client.get).toHaveBeenCalledWith('/api/v1/dashboard/channels/channel%2F1/config')
  })
})
