import { type FormEvent, type ReactNode, useEffect, useState } from 'react'
import { useParams } from 'react-router'
import { isAxiosError } from 'axios'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { restoreSession, isAuthenticated } from '@/services/auth'
import { apiBaseURL } from '@/services/api'
import { getClaim, submitClaim, type ClaimDraw, type ClaimInput } from '@/services/claim'

type ViewState =
  | { phase: 'loading' }
  | { phase: 'not_found' }
  | { phase: 'expired' }
  | { phase: 'forbidden' }
  | { phase: 'unauthenticated'; draw: ClaimDraw }
  | { phase: 'form'; draw: ClaimDraw }
  | { phase: 'submitted' }
  | { phase: 'load_error' }

const initialForm: ClaimInput = {
  recipient_name: '',
  phone: '',
  address_line1: '',
  address_line2: '',
  city: '',
  postal_code: '',
  country: '',
}

function Shell({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col items-center justify-center px-4 py-12">
      {children}
    </div>
  )
}

function NotFoundMessage() {
  return (
    <Shell>
      <div className="text-center space-y-2">
        <h1 className="text-2xl font-bold">找不到此領獎連結</h1>
        <p className="text-muted-foreground">連結可能有誤，請確認 Email 中的連結是否完整。</p>
      </div>
    </Shell>
  )
}

export default function ClaimPage() {
  const { token = '' } = useParams<{ token: string }>()
  const [state, setState] = useState<ViewState>({ phase: 'loading' })
  const [form, setForm] = useState<ClaimInput>(initialForm)
  const [submitting, setSubmitting] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) {
      return
    }
    let cancelled = false
    Promise.all([restoreSession(), getClaim(token)]).then(
      ([, draw]) => {
        if (cancelled) return
        setState(isAuthenticated() ? { phase: 'form', draw } : { phase: 'unauthenticated', draw })
      },
      (err) => {
        if (cancelled) return
        if (isAxiosError(err)) {
          const status = err.response?.status
          if (status === 404) setState({ phase: 'not_found' })
          else if (status === 410) setState({ phase: 'expired' })
          else if (status === 403) setState({ phase: 'forbidden' })
          else setState({ phase: 'load_error' })
        } else {
          setState({ phase: 'load_error' })
        }
      },
    )
    return () => {
      cancelled = true
    }
  }, [token])

  if (!token) {
    return <NotFoundMessage />
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (submitting) return
    setSubmitting(true)
    setFormError(null)
    try {
      await submitClaim(token, form)
      setState({ phase: 'submitted' })
    } catch (err) {
      if (isAxiosError(err)) {
        const status = err.response?.status
        if (status === 403) setFormError('此帳號非中獎者，無法領取獎品')
        else if (status === 409) setFormError('此獎品已完成領取')
        else setFormError('送出失敗，請稍後再試')
      } else {
        setFormError('送出失敗，請稍後再試')
      }
    } finally {
      setSubmitting(false)
    }
  }

  function field(key: keyof ClaimInput, label: string, required = false) {
    return (
      <div className="space-y-1">
        <Label htmlFor={key}>
          {label}
          {required && <span className="text-destructive ml-1">*</span>}
        </Label>
        <Input
          id={key}
          value={form[key]}
          onChange={e => setForm(f => ({ ...f, [key]: e.target.value }))}
          required={required}
        />
      </div>
    )
  }

  if (state.phase === 'loading') {
    return (
      <Shell>
        <p className="text-muted-foreground">載入中…</p>
      </Shell>
    )
  }

  if (state.phase === 'not_found') {
    return <NotFoundMessage />
  }

  if (state.phase === 'expired') {
    return (
      <Shell>
        <div className="text-center space-y-2">
          <h1 className="text-2xl font-bold">領獎連結已過期</h1>
          <p className="text-muted-foreground">領獎期限已過，請聯絡主辦單位。</p>
        </div>
      </Shell>
    )
  }

  if (state.phase === 'load_error') {
    return (
      <Shell>
        <div className="text-center space-y-2">
          <h1 className="text-2xl font-bold">載入失敗</h1>
          <p className="text-muted-foreground">請稍後再試，或聯絡主辦單位。</p>
        </div>
      </Shell>
    )
  }

  if (state.phase === 'forbidden') {
    return (
      <Shell>
        <div className="text-center space-y-2">
          <h1 className="text-2xl font-bold">無法領取此獎品</h1>
          <p className="text-muted-foreground">此帳號並非此獎項的中獎者。</p>
        </div>
      </Shell>
    )
  }

  if (state.phase === 'submitted') {
    return (
      <Shell>
        <div className="text-center space-y-2">
          <h1 className="text-2xl font-bold">領獎資料已送出！</h1>
          <p className="text-muted-foreground">我們將盡快安排寄送，感謝您的參與。</p>
        </div>
      </Shell>
    )
  }

  if (state.phase === 'unauthenticated') {
    const { draw } = state
    const name = draw.entry.display_name || draw.entry.twitch_login
    const expires = new Date(draw.claim_expires_at).toLocaleDateString('zh-TW')
    return (
      <Shell>
        <div className="text-center space-y-4 max-w-sm">
          <h1 className="text-3xl font-bold">恭喜中獎！</h1>
          <p className="text-lg">{name}，您已抽中獎品</p>
          <p className="text-sm text-muted-foreground">領獎截止：{expires}</p>
          <Button
            className="w-full"
            onClick={() => {
              const url = new URL('/api/v1/auth/twitch', apiBaseURL)
              url.searchParams.set('redirect_to', `/claim/${token}`)
              window.location.href = url.toString()
            }}
          >
            用 Twitch 帳號登入領獎
          </Button>
        </div>
      </Shell>
    )
  }

  // phase === 'form'
  const { draw } = state
  const name = draw.entry.display_name || draw.entry.twitch_login
  const expires = new Date(draw.claim_expires_at).toLocaleDateString('zh-TW')

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col items-center py-12 px-4">
      <div className="w-full max-w-md space-y-6">
        <div className="text-center space-y-1">
          <h1 className="text-2xl font-bold">填寫領獎資料</h1>
          <p className="text-sm text-muted-foreground">以 {name} 身份登入・領獎截止：{expires}</p>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4">
          {field('recipient_name', '收件人姓名', true)}
          {field('phone', '電話')}
          {field('address_line1', '地址第一行', true)}
          {field('address_line2', '地址第二行')}
          {field('city', '城市', true)}
          {field('postal_code', '郵遞區號')}
          {field('country', '國家')}
          {formError && <p className="text-sm text-destructive">{formError}</p>}
          <Button type="submit" className="w-full" disabled={submitting}>
            {submitting ? '送出中…' : '送出領獎資料'}
          </Button>
        </form>
      </div>
    </div>
  )
}
