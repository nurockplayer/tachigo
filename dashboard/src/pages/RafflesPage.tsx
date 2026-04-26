import { isAxiosError } from 'axios'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { listRaffles, createRaffle, type Raffle, type RaffleStatus } from '@/services/raffles'

const statusLabel: Record<RaffleStatus, string> = {
  draft: '草稿',
  active: '進行中',
  completed: '已結束',
}

const statusClass: Record<RaffleStatus, string> = {
  draft: 'bg-secondary text-muted-foreground',
  active: 'bg-green-100 text-green-800',
  completed: 'bg-destructive/10 text-destructive',
}

export default function RafflesPage() {
  const navigate = useNavigate()
  const [raffles, setRaffles] = useState<Raffle[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [title, setTitle] = useState('')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)
  useEffect(() => {
    let mounted = true
    listRaffles()
      .then((data) => { if (mounted) setRaffles(data) })
      .catch(() => { if (mounted) setError(true) })
      .finally(() => { if (mounted) setLoading(false) })
    return () => { mounted = false }
  }, [])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!title.trim()) return
    setCreating(true)
    setCreateError(null)
    try {
      const raffle = await createRaffle(title.trim())
      setRaffles((prev) => [raffle, ...prev])
      setTitle('')
    } catch (err) {
      const apiError = isAxiosError(err) ? err.response?.data?.error : undefined
      setCreateError(
        typeof apiError === 'string' ? apiError : '建立失敗',
      )
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-foreground">抽獎管理</h1>

      {/* 建立表單 */}
      <form onSubmit={handleCreate} className="flex items-end gap-3">
        <div className="flex-1 space-y-1">
          <label htmlFor="raffle-title" className="text-sm font-medium text-foreground">
            活動名稱
          </label>
          <Input
            id="raffle-title"
            name="title"
            placeholder="例：2026 春季觀眾抽獎"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            disabled={creating}
          />
        </div>
        <Button type="submit" disabled={!title.trim() || creating}>
          {creating ? '建立中...' : '建立活動'}
        </Button>
      </form>
      {createError && (
        <p className="text-sm text-destructive">{createError}</p>
      )}

      {/* 列表 */}
      {loading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} data-testid="skeleton" className="h-11 w-full" />
          ))}
        </div>
      ) : error ? (
        <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          無法載入抽獎活動。
        </div>
      ) : raffles.length === 0 ? (
        <div className="rounded-lg border border-border bg-secondary/20 px-4 py-8 text-center text-sm text-muted-foreground">
          尚無抽獎活動，請建立第一個活動。
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-secondary/50">
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">活動名稱</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">狀態</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">建立時間</th>
              </tr>
            </thead>
            <tbody>
              {raffles.map((raffle, index) => (
                <tr
                  key={raffle.id}
                  tabIndex={0}
                  role="button"
                  aria-label={`開啟 ${raffle.title} 的控制頁`}
                  className={`cursor-pointer border-b border-border transition-colors last:border-0 hover:bg-accent/30 focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-primary ${index % 2 === 0 ? '' : 'bg-secondary/20'}`}
                  onClick={() => navigate(`/raffles/${raffle.id}`)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault()
                      navigate(`/raffles/${raffle.id}`)
                    }
                  }}
                >
                  <td className="px-4 py-3 font-medium text-foreground">{raffle.title}</td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${statusClass[raffle.status]}`}>
                      {statusLabel[raffle.status]}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {new Date(raffle.created_at).toLocaleDateString('zh-TW')}
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
