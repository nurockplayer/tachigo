import { useList } from '@refinedev/core'
import { Skeleton } from '@/components/ui/skeleton'
import type { Streamer } from '@/services/channels'

type Transaction = {
  id?: string
  type?: string
  source?: string
  amount?: number
  delta?: number
  balance_after?: number
  note?: string
  created_at?: string
}

function formatTransactionAmount(transaction: Transaction) {
  const value = transaction.amount ?? transaction.delta
  return typeof value === 'number' ? value.toLocaleString() : '—'
}

export default function TransactionsPage() {
  const channelsResult = useList<Streamer>({
    resource: 'streamer-channels',
    queryOptions: { retry: false },
  })
  const channelsQuery = channelsResult.query
  const channelId = channelsQuery.data?.data[0]?.channel_id
  const { query: { data, isLoading: transactionsLoading, isError: transactionsError } } = useList<Transaction>({
    resource: 'transactions',
    meta: { params: { channel_id: channelId } },
    queryOptions: { enabled: Boolean(channelId), retry: false },
  })
  const isLoading = channelsQuery.isLoading || (Boolean(channelId) && transactionsLoading)
  const isError = channelsQuery.isError || transactionsError
  const transactions: Transaction[] = data?.data ?? []

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-foreground">交易紀錄</h1>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <Skeleton key={index} className="h-11 w-full" />
          ))}
        </div>
      ) : isError ? (
        <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          無法載入交易紀錄。
        </div>
      ) : transactions.length === 0 ? (
        <div className="rounded-lg border border-border bg-secondary/20 px-4 py-8 text-center text-sm text-muted-foreground">
          尚無交易紀錄。
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-secondary/50">
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">類型</th>
                <th className="px-4 py-3 text-right font-medium text-muted-foreground">數量</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">備註</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">時間</th>
              </tr>
            </thead>
            <tbody>
              {transactions.map((transaction, index) => (
                <tr
                  key={transaction.id ?? `${transaction.created_at ?? '?'}-${index}`}
                  className={`border-b border-border last:border-0 ${index % 2 === 0 ? '' : 'bg-secondary/20'}`}
                >
                  <td className="px-4 py-3 font-medium text-foreground">
                    {transaction.type ?? transaction.source ?? '—'}
                  </td>
                  <td className="px-4 py-3 text-right text-muted-foreground">
                    {formatTransactionAmount(transaction)}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">{transaction.note ?? '—'}</td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {transaction.created_at
                      ? new Date(transaction.created_at).toLocaleString('zh-TW')
                      : '—'}
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
