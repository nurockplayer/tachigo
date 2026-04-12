import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  getChannelConfig,
  getStreamerStats,
  type ChannelConfig,
  type StreamerStats,
} from '@/services/channels'

function formatHours(seconds?: number) {
  if (seconds === undefined) return '—'
  return `${(seconds / 3600).toFixed(1)} 小時`
}

function formatMinutes(seconds?: number) {
  if (seconds === undefined) return '—'
  return `${Math.round(seconds / 60)} 分`
}

function formatNumber(value?: number, unit?: string) {
  if (value === undefined) return '—'
  return `${value.toLocaleString()}${unit ? ` ${unit}` : ''}`
}

function calcPerMinute(config: ChannelConfig | null) {
  if (!config || config.seconds_per_point === 0) return '—'
  return `${((60 / config.seconds_per_point) * config.multiplier).toFixed(1)} 點`
}

function TimeCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex-1 rounded-lg border border-border bg-secondary/30 p-4 text-center">
      <p className="mb-1 text-xs text-muted-foreground">{label}</p>
      <p className="text-lg font-bold text-foreground">{value}</p>
    </div>
  )
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex-1 space-y-1 rounded-lg border border-border bg-secondary/30 p-5">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-2xl font-bold text-foreground">{value}</p>
    </div>
  )
}

export default function StreamerDetailPage() {
  const { streamerId } = useParams()
  const navigate = useNavigate()
  const [stats, setStats] = useState<StreamerStats | null>(null)
  const [config, setConfig] = useState<ChannelConfig | null>(null)
  const [resolvedStreamerId, setResolvedStreamerId] = useState<string | null>(null)
  const [failedStreamerId, setFailedStreamerId] = useState<string | null>(null)

  useEffect(() => {
    if (!streamerId) return

    let mounted = true

    getStreamerStats(streamerId)
      .then(async ({ stats, channelId }) => {
        if (!mounted) return
        setStats(stats)

        try {
          const cfg = await getChannelConfig(channelId)
          if (!mounted) return
          setConfig(cfg)
        } catch {
          if (!mounted) return
          setConfig(null)
        }

        if (!mounted) return
        setFailedStreamerId(null)
        setResolvedStreamerId(streamerId)
      })
      .catch(() => {
        if (!mounted) return
        setFailedStreamerId(streamerId)
      })

    return () => {
      mounted = false
    }
  }, [streamerId])

  const loading = Boolean(streamerId) && resolvedStreamerId !== streamerId && failedStreamerId !== streamerId
  const error = failedStreamerId === streamerId

  const timeCards = [
    { label: '本次', value: formatHours(stats?.current_session_seconds) },
    { label: '本日', value: formatHours(stats?.daily_seconds) },
    { label: '本月', value: formatHours(stats?.monthly_seconds) },
    { label: '年度', value: formatHours(stats?.yearly_seconds) },
  ]

  const metricCards = [
    { label: '挖礦參與人數', value: formatNumber(stats?.unique_miners, '人') },
    { label: '觀眾平均停留', value: formatMinutes(stats?.avg_session_seconds) },
    { label: '總產出點數', value: formatNumber(stats?.total_token_minted, '點') },
    { label: '可用點數總量', value: formatNumber(stats?.spendable_in_circulation, '點') },
  ]

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate('/streamers')}
            className="text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            ← 返回列表
          </button>
          <h1 className="text-2xl font-bold text-foreground">{streamerId}</h1>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" disabled>
            空投
          </Button>
          <Button variant="outline" disabled>
            調整倍率
          </Button>
        </div>
      </div>

      {loading ? (
        <div className="space-y-4">
          <Skeleton className="h-28 w-full" />
          <div className="grid gap-4 md:grid-cols-4">
            <Skeleton className="h-28 w-full" />
            <Skeleton className="h-28 w-full" />
            <Skeleton className="h-28 w-full" />
            <Skeleton className="h-28 w-full" />
          </div>
          <Skeleton className="h-40 w-full" />
        </div>
      ) : error || !stats || !streamerId ? (
        <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          無法載入頻道詳細資料
        </div>
      ) : (
        <>
          <section className="space-y-3">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
              開台時數
            </h2>
            <div className="grid gap-3 md:grid-cols-4">
              {timeCards.map((card) => (
                <TimeCard key={card.label} label={card.label} value={card.value} />
              ))}
            </div>
          </section>

          <div className="grid gap-3 md:grid-cols-4">
            {metricCards.map((card) => (
              <MetricCard key={card.label} label={card.label} value={card.value} />
            ))}
          </div>

          <section className="space-y-4 rounded-lg border border-border bg-secondary/30 p-6">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
              挖礦倍率設定
            </h2>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">每秒點數基準</span>
                <span className="font-medium text-foreground">
                  {config ? `${config.seconds_per_point} 秒 / 點` : '—'}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">目前倍率</span>
                <span className="font-medium text-foreground">
                  {config ? `${config.multiplier}x` : '—'}
                </span>
              </div>
              <div className="mt-2 flex justify-between border-t border-border pt-2">
                <span className="text-muted-foreground">每分鐘產出</span>
                <span className="font-semibold text-primary">{calcPerMinute(config)}</span>
              </div>
            </div>
          </section>
        </>
      )}
    </div>
  )
}
