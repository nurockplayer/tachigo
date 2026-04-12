import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router'
import { Skeleton } from '@/components/ui/skeleton'
import { getMyChannels, getStreamers, type Streamer } from '@/services/channels'

export default function StreamersPage() {
  const navigate = useNavigate()
  const [streamers, setStreamers] = useState<Streamer[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [shouldAutoRedirect, setShouldAutoRedirect] = useState(false)

  function openStreamer(streamerId: string) {
    navigate(`/streamers/${streamerId}`)
  }

  useEffect(() => {
    setError(false)
    setLoading(true)
    setStreamers([])
    setShouldAutoRedirect(false)

    let mounted = true

    getMyChannels()
      .then((data) => {
        if (!mounted) return
        setStreamers(data)
        setShouldAutoRedirect(true)
      })
      .catch(async () => {
        try {
          const data = await getStreamers()
          if (!mounted) return
          setStreamers(data)
        } catch {
          if (!mounted) return
          setError(true)
        }
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
    if (loading || error || !shouldAutoRedirect) return
    const first = streamers[0]
    if (!first) return
    navigate(`/streamers/${first.id}`, { replace: true })
  }, [streamers, loading, error, navigate, shouldAutoRedirect])

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
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                  Channel ID
                </th>
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
                  <td className="px-4 py-3 text-muted-foreground">{streamer.channel_id}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
