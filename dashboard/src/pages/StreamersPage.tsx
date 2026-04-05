import { useEffect, useState } from 'react'
import { usePermissions } from '@refinedev/core'
import { useNavigate } from 'react-router'
import { Skeleton } from '@/components/ui/skeleton'
import { getStreamerChannels, type ChannelListItem } from '@/services/channels'

function formatHours(seconds?: number) {
  if (seconds === undefined) return '—'
  return `${(seconds / 3600).toFixed(1)} 小時`
}

function formatNumber(value?: number) {
  if (value === undefined) return '—'
  return value.toLocaleString()
}

export default function StreamersPage() {
  const navigate = useNavigate()
  const { data: role } = usePermissions<string>({})
  const [channels, setChannels] = useState<ChannelListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)

  useEffect(() => {
    let mounted = true

    getStreamerChannels()
      .then((data) => {
        if (!mounted) return
        setChannels(data)
      })
      .catch(() => {
        if (!mounted) return
        setError(true)
      })
      .finally(() => {
        if (!mounted) return
        setLoading(false)
      })

    return () => {
      mounted = false
    }
  }, [])

  useEffect(() => {
    if (loading || role !== 'streamer') return
    const firstChannel = channels[0]
    if (!firstChannel) return
    navigate(`/streamers/${firstChannel.channel_id}`, { replace: true })
  }, [channels, loading, navigate, role])

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-foreground">實況主管理</h1>

      {loading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <Skeleton key={index} className="h-11 w-full" />
          ))}
        </div>
      ) : error ? (
        <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          無法載入實況主資料
        </div>
      ) : channels.length === 0 ? (
        <div className="rounded-lg border border-border bg-secondary/20 px-4 py-8 text-center text-sm text-muted-foreground">
          目前沒有可顯示的實況主資料
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-secondary/50">
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">實況主名稱</th>
                <th className="px-4 py-3 text-right font-medium text-muted-foreground">本日開台</th>
                <th className="px-4 py-3 text-right font-medium text-muted-foreground">挖礦觀眾</th>
                <th className="px-4 py-3 text-right font-medium text-muted-foreground">總產出點數</th>
              </tr>
            </thead>
            <tbody>
              {channels.map((channel, index) => (
                <tr
                  key={channel.id}
                  className={`cursor-pointer border-b border-border transition-colors last:border-0 hover:bg-accent/30 ${index % 2 === 0 ? '' : 'bg-secondary/20'}`}
                  onClick={() => navigate(`/streamers/${channel.channel_id}`)}
                >
                  <td className="px-4 py-3 font-medium text-foreground">{channel.display_name}</td>
                  <td className="px-4 py-3 text-right text-muted-foreground">
                    {formatHours(channel.daily_seconds)}
                  </td>
                  <td className="px-4 py-3 text-right text-muted-foreground">
                    {formatNumber(channel.unique_miners)}
                  </td>
                  <td className="px-4 py-3 text-right text-muted-foreground">
                    {formatNumber(channel.total_token_minted)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
