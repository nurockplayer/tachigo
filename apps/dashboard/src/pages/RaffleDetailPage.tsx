import { useOne } from '@refinedev/core'
import { useParams } from 'react-router'
import { Skeleton } from '@/components/ui/skeleton'
import type { Raffle } from '@/services/raffles'

const statusLabel: Record<string, string> = {
  draft: '草稿',
  active: '進行中',
  completed: '已結束',
}

export default function RaffleDetailPage() {
  const { raffleId } = useParams()
  const { query: { data, isLoading, isError } } = useOne<Raffle>({
    resource: 'raffles',
    id: raffleId,
    queryOptions: { enabled: Boolean(raffleId), retry: false },
  })
  const raffle = data?.data

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-foreground">抽獎控制頁</h1>

      {isLoading ? (
        <Skeleton className="h-32 w-full" />
      ) : isError || !raffle ? (
        <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          無法載入抽獎活動。
        </div>
      ) : (
        <section className="space-y-3 rounded-lg border border-border bg-secondary/30 p-6">
          <div>
            <p className="text-sm text-muted-foreground">活動名稱</p>
            <p className="text-xl font-semibold text-foreground">{raffle.title}</p>
          </div>
          <div className="grid gap-3 md:grid-cols-3">
            <div>
              <p className="text-sm text-muted-foreground">狀態</p>
              <p className="font-medium text-foreground">{statusLabel[raffle.status] ?? raffle.status}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">建立時間</p>
              <p className="font-medium text-foreground">
                {new Date(raffle.created_at).toLocaleString('zh-TW')}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">更新時間</p>
              <p className="font-medium text-foreground">
                {new Date(raffle.updated_at).toLocaleString('zh-TW')}
              </p>
            </div>
          </div>
        </section>
      )}
    </div>
  )
}
