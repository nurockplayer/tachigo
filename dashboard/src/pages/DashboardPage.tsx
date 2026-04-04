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

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-foreground">總覽</h1>

      <div className="grid gap-4 sm:grid-cols-2">
        {/* 使用者資訊卡 */}
        <div className="space-y-3 rounded-lg border border-border p-6">
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

        {/* 系統狀態卡 */}
        <div className="space-y-3 rounded-lg border border-border p-6">
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
    </div>
  )
}
