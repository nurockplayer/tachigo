import { useList } from '@refinedev/core'
import { isAxiosError } from 'axios'
import { useEffect, useMemo } from 'react'
import { useNavigate } from 'react-router'
import { Skeleton } from '@/components/ui/skeleton'
import type { Streamer } from '@/services/channels'

function shouldFallbackToGetStreamers(error: unknown) {
  if (!isAxiosError(error)) return false
  return error.response?.status === 401 || error.response?.status === 403
}

export default function StreamersPage() {
  const navigate = useNavigate()
  const myChannelsResult = useList<Streamer>({
    resource: 'streamer-channels',
    queryOptions: { retry: false },
  })
  const myChannelsQuery = myChannelsResult.query
  const shouldFallback = shouldFallbackToGetStreamers(myChannelsQuery.error)
  const streamersResult = useList<Streamer>({
    resource: 'streamers',
    queryOptions: { enabled: shouldFallback, retry: false },
  })
  const streamersQuery = streamersResult.query
  const streamers: Streamer[] = useMemo(
    () =>
      myChannelsQuery.data?.data
      ?? (shouldFallback ? streamersQuery.data?.data : undefined)
      ?? [],
    [myChannelsQuery.data?.data, shouldFallback, streamersQuery.data?.data],
  )
  const loading = myChannelsQuery.isLoading || (shouldFallback && streamersQuery.isLoading)
  const error = Boolean(
    (myChannelsQuery.error && !shouldFallback) || (shouldFallback && streamersQuery.error),
  )

  function openStreamer(streamerId: string) {
    navigate(`/streamers/${streamerId}`)
  }

  useEffect(() => {
    if (loading || error || !myChannelsQuery.isSuccess) return
    const first = streamers[0]
    if (!first) return
    navigate(`/streamers/${first.id}`, { replace: true })
  }, [streamers, loading, error, navigate, myChannelsQuery.isSuccess])

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-foreground">直播主列表</h1>

      {loading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <Skeleton key={index} className="h-11 w-full" />
          ))}
        </div>
      ) : error ? (
        <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          無法載入直播主資料。
        </div>
      ) : streamers.length === 0 ? (
        <div className="rounded-lg border border-border bg-secondary/20 px-4 py-8 text-center text-sm text-muted-foreground">
          尚無可顯示的直播主資料。
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-secondary/50">
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                  直播主名稱
                </th>
                <th className="px-4 py-3 text-right font-medium text-muted-foreground">
                  本日開台
                </th>
                <th className="px-4 py-3 text-right font-medium text-muted-foreground">
                  挖礦觀眾
                </th>
                <th className="px-4 py-3 text-right font-medium text-muted-foreground">
                  總產出點數
                </th>
                <th className="w-8" />
              </tr>
            </thead>
            <tbody>
              {streamers.map((streamer, index) => (
                <tr
                  key={streamer.id}
                  tabIndex={0}
                  role="button"
                  aria-label={`查看 ${streamer.display_name || streamer.channel_id} 的詳細資料`}
                  className={`cursor-pointer border-b border-border transition-colors last:border-0 hover:bg-accent/30 focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-primary ${index % 2 === 0 ? '' : 'bg-secondary/20'}`}
                  onClick={() => openStreamer(streamer.id)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault()
                      openStreamer(streamer.id)
                    }
                  }}
                >
                  <td className="px-4 py-3 font-medium text-foreground">
                    {streamer.display_name || streamer.channel_id}
                  </td>
                  <td className="px-4 py-3 text-right text-muted-foreground">
                    {streamer.daily_seconds !== undefined
                      ? `${(streamer.daily_seconds / 3600).toFixed(1)} hr`
                      : '—'}
                  </td>
                  <td className="px-4 py-3 text-right text-muted-foreground">
                    {streamer.unique_miners !== undefined
                      ? streamer.unique_miners.toLocaleString()
                      : '—'}
                  </td>
                  <td className="px-4 py-3 text-right text-muted-foreground">
                    {streamer.total_token_minted !== undefined
                      ? streamer.total_token_minted.toLocaleString()
                      : '—'}
                  </td>
                  <td className="px-4 py-3 text-right text-muted-foreground">→</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
