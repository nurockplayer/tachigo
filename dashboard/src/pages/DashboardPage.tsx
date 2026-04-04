import { useEffect, useState } from 'react'
import { cn } from '@/lib/utils'
import { ROLE_LABELS, ROLE_BADGE_STYLES, type UserRole } from '@/lib/user'
import client from '@/services/api'

interface User {
  username: string | null
  email: string | null
  role: UserRole
  email_verified: boolean
}

const MOCK_STATS = {
  interactionRate: 34,
  collaborationCount: 1240,
  retentionRate: 61,
  conversionCount: 87,
  angelGateProgress: 68,
  angelGateTarget: 100,
}

function StatCard({
  label,
  value,
  unit,
  description,
}: {
  label: string
  value: number | string
  unit?: string
  description: string
}) {
  return (
    <div className="space-y-1 rounded-lg border border-border bg-secondary/30 p-5">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-2xl font-bold text-foreground">
        {value}
        {unit && <span className="ml-1 text-sm font-normal text-muted-foreground">{unit}</span>}
      </p>
      <p className="text-xs text-muted-foreground">{description}</p>
    </div>
  )
}

export default function DashboardPage() {
  const [user, setUser] = useState<User | null>(null)
  const [userLoading, setUserLoading] = useState(true)
  const [userError, setUserError] = useState(false)
  const [healthy, setHealthy] = useState<boolean | null>(null)

  useEffect(() => {
    client
      .get<{ success: boolean; data: { user: User } }>('/api/v1/users/me')
      .then(({ data: body }) => setUser(body.data.user))
      .catch(() => setUserError(true))
      .finally(() => setUserLoading(false))

    client
      .get<{ status: string }>('/health')
      .then(({ data }) => setHealthy(data.status === 'ok'))
      .catch(() => setHealthy(false))
  }, [])

  const displayName = user?.username ?? user?.email ?? '使用者'
  const progressPct = Math.min(
    100,
    Math.round((MOCK_STATS.angelGateProgress / MOCK_STATS.angelGateTarget) * 100),
  )

  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-bold text-foreground">總覽</h1>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-3 rounded-lg border border-border bg-secondary/30 p-6">
          <p className="text-sm text-muted-foreground">歡迎回來</p>
          {userLoading ? (
            <div className="space-y-2">
              <div className="h-6 w-32 animate-pulse rounded bg-muted" />
              <div className="h-4 w-20 animate-pulse rounded bg-muted" />
            </div>
          ) : userError ? (
            <p className="text-sm text-destructive">無法載入帳號資訊</p>
          ) : (
            <>
              <p className="text-xl font-semibold text-foreground">{displayName}</p>
              {user && (
                <div className="flex flex-wrap items-center gap-2">
                  <span
                    className={cn(
                      'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                      ROLE_BADGE_STYLES[user.role],
                    )}
                  >
                    {ROLE_LABELS[user.role]}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    {user.email_verified ? '信箱已驗證' : '信箱未驗證'}
                  </span>
                </div>
              )}
            </>
          )}
        </div>

        <div className="space-y-3 rounded-lg border border-border bg-secondary/30 p-6">
          <p className="text-sm text-muted-foreground">系統狀態</p>
          <div className="flex items-center gap-2">
            <span
              className={cn(
                'h-2.5 w-2.5 rounded-full',
                healthy === null
                  ? 'bg-muted-foreground'
                  : healthy
                    ? 'bg-green-500'
                    : 'bg-red-500',
              )}
            />
            <span className="text-sm font-medium text-foreground">
              {healthy === null ? '確認中...' : healthy ? '正常運作' : '服務異常'}
            </span>
          </div>
        </div>
      </div>

      <section className="space-y-3">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
          商業指標
          <span className="ml-2 text-xs font-normal normal-case">(示範資料)</span>
        </h2>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <StatCard
            label="互動率"
            value={MOCK_STATS.interactionRate}
            unit="%"
            description="每場直播點擊參與觀眾比例"
          />
          <StatCard
            label="協作率"
            value={MOCK_STATS.collaborationCount.toLocaleString()}
            unit="人"
            description="參與安琪拉之門的唯一觀眾數"
          />
          <StatCard
            label="7 日留存率"
            value={MOCK_STATS.retentionRate}
            unit="%"
            description="導入後 7 日回訪率"
          />
          <StatCard
            label="轉化數"
            value={MOCK_STATS.conversionCount}
            unit="件"
            description="任務完成 + NFT 鑄造合計"
          />
        </div>
      </section>

      <section className="space-y-3">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
          安琪拉之門 — 全服協作進度
          <span className="ml-2 text-xs font-normal normal-case">(示範資料)</span>
        </h2>
        <div className="space-y-4 rounded-lg border border-border bg-secondary/30 p-6">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">全服能量匯聚進度</span>
            <span className="font-semibold text-foreground">{progressPct}%</span>
          </div>
          <div className="h-3 w-full overflow-hidden rounded-full bg-muted">
            <div
              className="h-full rounded-full bg-primary transition-all duration-500"
              style={{ width: `${progressPct}%` }}
            />
          </div>
          <p className="text-xs text-muted-foreground">
            達成 100% 後觸發 E5，解鎖頻道專屬 Web3 NFT
          </p>
        </div>
      </section>
    </div>
  )
}
