import { useList } from '@refinedev/core'
import { Skeleton } from '@/components/ui/skeleton'

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

export default function TransactionsPage() {
  const { query: { data, isLoading, isError } } = useList<Transaction>({
    resource: 'transactions',
    queryOptions: { retry: false },
  })
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
                  key={transaction.id ?? `${transaction.created_at}-${index}`}
                  className={`border-b border-border last:border-0 ${index % 2 === 0 ? '' : 'bg-secondary/20'}`}
                >
                  <td className="px-4 py-3 font-medium text-foreground">
                    {transaction.type ?? transaction.source ?? '—'}
                  </td>
                  <td className="px-4 py-3 text-right text-muted-foreground">
                    {(transaction.amount ?? transaction.delta ?? 0).toLocaleString()}
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
