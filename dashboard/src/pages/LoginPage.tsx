import { useState } from 'react'
import { useNavigate } from 'react-router'
import { isAxiosError } from 'axios'
import { login } from '@/services/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

const messages = {
  'zh-TW': {
    title: '登入',
    password: '密碼',
    submit: '登入',
    submitting: '登入中...',
    errorInvalidCredentials: '電子郵件或密碼錯誤',
    errorConnection: '連線失敗，請稍後再試',
  },
  'zh-CN': {
    title: '登录',
    password: '密码',
    submit: '登录',
    submitting: '登录中...',
    errorInvalidCredentials: '邮箱或密码错误',
    errorConnection: '连接失败，请稍后再试',
  },
  en: {
    title: 'Login',
    password: 'Password',
    submit: 'Login',
    submitting: 'Logging in...',
    errorInvalidCredentials: 'Invalid email or password',
    errorConnection: 'Connection failed, please try again later',
  },
  ja: {
    title: 'ログイン',
    password: 'パスワード',
    submit: 'ログイン',
    submitting: 'ログイン中...',
    errorInvalidCredentials: 'メールアドレスまたはパスワードが違います',
    errorConnection: '接続に失敗しました。後でもう一度お試しください',
  },
  ko: {
    title: '로그인',
    password: '비밀번호',
    submit: '로그인',
    submitting: '로그인 중...',
    errorInvalidCredentials: '이메일 또는 비밀번호가 올바르지 않습니다',
    errorConnection: '연결에 실패했습니다. 나중에 다시 시도해 주세요',
  },
  fr: {
    title: 'Connexion',
    password: 'Mot de passe',
    submit: 'Se connecter',
    submitting: 'Connexion...',
    errorInvalidCredentials: 'Email ou mot de passe invalide',
    errorConnection: 'Échec de la connexion, veuillez réessayer plus tard',
  },
  de: {
    title: 'Anmelden',
    password: 'Passwort',
    submit: 'Anmelden',
    submitting: 'Anmeldung...',
    errorInvalidCredentials: 'E-Mail oder Passwort ungültig',
    errorConnection: 'Verbindung fehlgeschlagen, bitte später erneut versuchen',
  },
  es: {
    title: 'Iniciar sesión',
    password: 'Contraseña',
    submit: 'Iniciar sesión',
    submitting: 'Iniciando sesión...',
    errorInvalidCredentials: 'Correo o contraseña incorrectos',
    errorConnection: 'Error de conexión, por favor inténtelo más tarde',
  },
  pt: {
    title: 'Entrar',
    password: 'Senha',
    submit: 'Entrar',
    submitting: 'Entrando...',
    errorInvalidCredentials: 'Email ou senha inválidos',
    errorConnection: 'Falha na conexão, tente novamente mais tarde',
  },
}

function getMessages() {
  const lang = navigator.language.toLowerCase()
  if (lang === 'zh-tw' || lang === 'zh-hk') return messages['zh-TW']
  if (lang.startsWith('zh')) return messages['zh-CN']
  if (lang.startsWith('ja')) return messages.ja
  if (lang.startsWith('ko')) return messages.ko
  if (lang.startsWith('fr')) return messages.fr
  if (lang.startsWith('de')) return messages.de
  if (lang.startsWith('es')) return messages.es
  if (lang.startsWith('pt')) return messages.pt
  return messages.en
}

export default function LoginPage() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const t = getMessages()

  async function handleSubmit(event: { preventDefault(): void }) {
    event.preventDefault()
    setIsLoading(true)
    setError('')

    try {
      await login(email, password)
      navigate('/')
    } catch (err) {
      if (isAxiosError(err) && err.response?.status === 401) {
        setError(t.errorInvalidCredentials)
      } else {
        setError(t.errorConnection)
      }
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="w-full max-w-sm rounded-lg border border-border bg-background p-8 shadow-sm">
        <h1 className="mb-6 text-2xl font-bold text-foreground">{t.title}</h1>
        {error && <p className="mb-4 text-sm text-destructive">{error}</p>}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1">
            <Label htmlFor="email">Email</Label>
            <Input
              id="email"
              type="email"
              placeholder="admin@example.com"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              required
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="password">{t.password}</Label>
            <Input
              id="password"
              type="password"
              placeholder="••••••••"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
            />
          </div>
          <Button type="submit" className="w-full" disabled={isLoading}>
            {isLoading ? t.submitting : t.submit}
          </Button>
        </form>
      </div>
    </div>
  )
}
