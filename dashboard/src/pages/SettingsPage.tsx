import { useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'
import { ROLE_LABELS, ROLE_BADGE_STYLES, type UserRole } from '@/lib/user'
import client from '@/services/api'

interface User {
  id: string
  username: string | null
  email: string | null
  avatar_url: string | null
  role: UserRole
  is_active: boolean
  email_verified: boolean
  created_at: string
  updated_at: string
}

function formatDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return '－'
  }

  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')

  return `${year}/${month}/${day}`
}

export default function SettingsPage() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [username, setUsername] = useState('')
  const [avatarUrl, setAvatarUrl] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  async function fetchUser() {
    const { data: body } = await client.get<{ success: boolean; data: { user: User } }>(
      '/api/v1/users/me',
    )
    const nextUser = body.data.user
    setUser(nextUser)
    setUsername(nextUser.username ?? '')
    setAvatarUrl(nextUser.avatar_url ?? '')
  }

  useEffect(() => {
    fetchUser()
      .catch(() => {
        setUser(null)
      })
      .finally(() => setLoading(false))
  }, [])

  async function handleSave() {
    setSaving(true)
    setError(null)
    setSuccess(false)

    try {
      await client.put('/api/v1/users/me', {
        username: username.trim() || null,
        avatar_url: avatarUrl.trim() || null,
      })
      await fetchUser()
      setSuccess(true)
      setTimeout(() => setSuccess(false), 3000)
    } catch {
      setError('儲存失敗，請確認使用者名稱未被使用')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <p className="text-muted-foreground">載入中...</p>
  }

  if (!user) {
    return <p className="text-destructive">無法載入帳號資訊</p>
  }

  return (
    <div className="max-w-2xl space-y-8">
      <h1 className="text-2xl font-bold text-foreground">設定</h1>

      <section className="space-y-4 rounded-lg border border-border p-6">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
          帳號資訊
        </h2>
        <InfoRow label="Email" value={user.email ?? '－'} />
        <InfoRow
          label="角色"
          value={
            <span
              className={cn(
                'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                ROLE_BADGE_STYLES[user.role],
              )}
            >
              {ROLE_LABELS[user.role]}
            </span>
          }
        />
        <InfoRow label="信箱驗證" value={user.email_verified ? '已驗證' : '未驗證'} />
        <InfoRow label="帳號建立時間" value={formatDate(user.created_at)} />
      </section>

      <section className="space-y-4 rounded-lg border border-border p-6">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
          編輯個人資料
        </h2>
        <form onSubmit={(e) => { e.preventDefault(); void handleSave() }} className="space-y-4">
          <Field
            label="使用者名稱"
            value={username}
            onChange={setUsername}
            placeholder="留空則移除"
          />
          <Field
            label="大頭貼 URL"
            value={avatarUrl}
            onChange={setAvatarUrl}
            placeholder="留空則移除"
          />
          {error && <p className="text-sm text-destructive">{error}</p>}
          {success && <p className="text-sm text-green-600">已儲存</p>}
          <Button type="submit" disabled={saving}>
            {saving ? '儲存中...' : '儲存'}
          </Button>
        </form>
      </section>
    </div>
  )
}

function InfoRow({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4 text-sm">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-right font-medium text-foreground">{value}</span>
    </div>
  )
}

function Field({
  label,
  value,
  onChange,
  placeholder,
}: {
  label: string
  value: string
  onChange: (value: string) => void
  placeholder?: string
}) {
  const inputId = `settings-field-${label === '使用者名稱' ? 'username' : 'avatar-url'}`

  return (
    <div className="space-y-1.5">
      <Label htmlFor={inputId} className="text-muted-foreground">
        {label}
      </Label>
      <Input
        id={inputId}
        type="text"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
      />
    </div>
  )
}
