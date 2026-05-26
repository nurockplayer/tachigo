import { useList, useOne } from '@refinedev/core'
import { useNavigate } from 'react-router'
import { Skeleton } from '@/components/ui/skeleton'
import type { Streamer } from '@/services/channels'
import type { Raffle, RaffleStatus } from '@/services/raffles'

type StreamerStats = {
  monthly_seconds?: number
  unique_miners?: number
  total_token_minted?: number
}

const statusLabel: Record<RaffleStatus, string> = {
  draft: '草稿',
  active: '進行中',
  completed: '已結束',
}

const statusClass: Record<RaffleStatus, string> = {
  draft: 'bg-secondary text-muted-foreground',
  active: 'bg-green-500/10 text-green-700',
  completed: 'bg-destructive/10 text-destructive',
}

function fmtHours(seconds?: number) {
  if (seconds === undefined) return undefined
  return `${(seconds / 3600).toFixed(1)} 小時`
}

function fmtNumber(value?: number) {
  if (value === undefined) return undefined
  return value.toLocaleString()
}

function StatCard({ label, value }: { label: string; value?: string }) {
  return (
    <div className="flex-1 rounded-lg border border-border bg-secondary/30 p-5">
      <p className="mb-1 text-xs text-muted-foreground">{label}</p>
      {value !== undefined
        ? <p className="text-2xl font-bold text-foreground">{value}</p>
        : <p className="text-2xl font-bold text-muted-foreground" data-testid="stat-empty">—</p>
      }
    </div>
  )
}

export default function DashboardPage() {
  const navigate = useNavigate()

  const { query: channelsQuery } = useList<Streamer>({
    resource: 'streamer-channels',
    queryOptions: { retry: false },
  })
  const channelId = channelsQuery.data?.data[0]?.id

  const { query: statsQuery } = useOne<StreamerStats>({
    resource: 'streamer-stats',
    id: channelId ?? '',
    queryOptions: { enabled: Boolean(channelId), retry: false },
  })
  const stats = statsQuery.data?.data

  const { query: rafflesQuery } = useList<Raffle>({
    resource: 'raffles',
    queryOptions: { retry: false },
  })
  const recentRaffles = (rafflesQuery.data?.data ?? []).slice(0, 3)

  const statsLoading = channelsQuery.isLoading || (Boolean(channelId) && statsQuery.isLoading)
  const rafflesLoading = rafflesQuery.isLoading

  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-bold text-foreground">總覽</h1>

      {/* Stats */}
      {statsLoading
        ? (
            <div className="flex gap-4" data-testid="stats-skeleton">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-24 flex-1 rounded-lg" />
              ))}
            </div>
          )
        : (
            <div className="flex gap-4">
              <StatCard label="本月開台" value={fmtHours(stats?.monthly_seconds)} />
              <StatCard label="挖礦觀眾" value={stats?.unique_miners !== undefined ? `${fmtNumber(stats.unique_miners)} 人` : undefined} />
              <StatCard label="總產出點數" value={stats?.total_token_minted !== undefined ? `${fmtNumber(stats.total_token_minted)} 點` : undefined} />
            </div>
          )
      }

      {/* Recent Raffles */}
      <div>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-foreground">最近抽獎</h2>
          <button
            onClick={() => navigate('/raffles')}
            className="text-xs text-primary hover:underline"
          >
            查看全部 →
          </button>
        </div>

        {rafflesLoading
          ? (
              <div className="space-y-2">
                {Array.from({ length: 3 }).map((_, i) => (
                  <Skeleton key={i} className="h-11 w-full rounded-lg" />
                ))}
              </div>
            )
          : recentRaffles.length === 0
            ? (
                <div className="rounded-lg border border-border bg-secondary/20 px-4 py-8 text-center text-sm text-muted-foreground">
                  尚無抽獎活動。
                </div>
              )
            : (
                <div className="overflow-hidden rounded-lg border border-border">
                  {recentRaffles.map((raffle, i) => (
                    <button
                      key={raffle.id}
                      onClick={() => navigate(`/raffles/${raffle.id}`)}
                      className={`flex w-full items-center justify-between border-b border-border px-4 py-3 text-left text-sm last:border-0 hover:bg-accent/30 ${i % 2 !== 0 ? 'bg-secondary/20' : ''}`}
                    >
                      <span className="font-medium text-foreground">{raffle.title}</span>
                      <span className="flex items-center gap-3">
                        <span className={`rounded-full px-2 py-0.5 text-xs ${statusClass[raffle.status]}`}>
                          {statusLabel[raffle.status]}
                        </span>
                        <span className="text-xs text-muted-foreground">
                          {new Date(raffle.created_at).toLocaleDateString('zh-TW')}
                        </span>
                      </span>
                    </button>
                  ))}
                </div>
              )
        }
      </div>
    </div>
  )
}
