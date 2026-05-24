import { useOne } from '@refinedev/core'
import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router'
import { Skeleton } from '@/components/ui/skeleton'
import { activateRaffle, completeRaffle, createPrizeTier, deletePrizeTier, drawFromTier, drawNext, importCSV, listDraws, listPrizeTiers, setDiscordWebhook } from '@/services/raffles'
import type { Raffle, RaffleDraw, RafflePrizeTier, RaffleStatus } from '@/services/raffles'
import gachaBg from '../assets/raffle-bg.jpg.png'

const OCEAN_KEYFRAMES = `
@keyframes oceanShake {
  0%,100%{transform:translateX(0)}
  15%{transform:translateX(-9px)}
  30%{transform:translateX(9px)}
  45%{transform:translateX(-6px)}
  60%{transform:translateX(6px)}
  75%{transform:translateX(-3px)}
  90%{transform:translateX(3px)}
}
@keyframes oceanPopout {
  0%{transform:translateX(-50%) translateY(0) scale(0);opacity:0}
  25%{transform:translateX(-50%) translateY(0) scale(1.4);opacity:1}
  60%{transform:translateX(-50%) translateY(38px) scale(1.1);opacity:1}
  85%{transform:translateX(-50%) translateY(70px) scale(.8);opacity:.6}
  100%{transform:translateX(-50%) translateY(100px) scale(0);opacity:0}
}
@keyframes oceanTwinkle {
  0%,100%{opacity:1;transform:scale(1)}
  50%{opacity:.3;transform:scale(.6)}
}
@keyframes oceanFloat {
  0%,100%{transform:translateY(0)}
  50%{transform:translateY(-6px)}
}
@keyframes oceanFadeSlideIn {
  from{opacity:0;transform:translateY(8px)}
  to{opacity:1;transform:translateY(0)}
}
@keyframes oceanNewMsg {
  from{opacity:0;transform:translateY(-6px)}
  to{opacity:1;transform:translateY(0)}
}
@keyframes oceanModalIn {
  from{opacity:0}
  to{opacity:1}
}
@keyframes oceanBoxBounce {
  from{transform:scale(.6)}
  to{transform:scale(1)}
}
@keyframes oceanLiveDot {
  0%,100%{opacity:1}
  50%{opacity:.25}
}
`

const glassStyle: React.CSSProperties = {
  background: 'rgba(4,14,52,.55)',
  border: '1px solid rgba(80,160,255,.22)',
  borderRadius: 14,
  backdropFilter: 'blur(14px)',
  WebkitBackdropFilter: 'blur(14px)',
}

const BALL_COLORS = [
  'linear-gradient(135deg,#93c5fd,#3b82f6)',
  'linear-gradient(135deg,#d8b4fe,#8b5cf6)',
  'linear-gradient(135deg,#86efac,#22c55e)',
  'linear-gradient(135deg,#fde68a,#f59e0b)',
  'linear-gradient(135deg,#f9a8d4,#ec4899)',
  'linear-gradient(135deg,#a5f3fc,#06b6d4)',
]

const statusLabel: Record<RaffleStatus, string> = {
  draft: '草稿',
  active: '進行中',
  completed: '已完成',
}

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr)
  if (Number.isNaN(date.getTime())) return ''

  const diffMs = Date.now() - date.getTime()
  const secs = Math.floor(diffMs / 1000)
  if (secs < 60) return '剛剛'

  const mins = Math.floor(secs / 60)
  if (mins < 60) return `${mins} 分鐘前`

  return date.toLocaleString('zh-TW')
}

function StatCard({
  label,
  value,
  colorClass,
}: {
  label: string
  value: string
  colorClass: string
}) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/[.04] p-3 text-center">
      <p className="mb-1 text-[10px] tracking-wide text-white/40">{label}</p>
      <p className={`text-3xl font-black leading-none ${colorClass}`}>{value}</p>
    </div>
  )
}

function WinnerList({ draws }: { draws: RaffleDraw[] }) {
  if (draws.length === 0) {
    return (
      <p data-testid="empty-winners" className="py-6 text-center text-sm text-white/30">
        目前還沒有抽出得獎者
      </p>
    )
  }

  const sorted = [...draws].sort(
    (a, b) => new Date(b.drawn_at).getTime() - new Date(a.drawn_at).getTime(),
  )

  return (
    <div className="flex flex-col gap-2">
      {sorted.map((draw, index) => (
        <div
          key={draw.id}
          data-testid="winner-item"
          className="flex items-center gap-3 rounded-xl border border-white/10 bg-white/[.04] px-4 py-3"
        >
          <span className="min-w-[28px] text-xs font-bold text-amber-400">
            #{draws.length - index}
          </span>
          <span className="flex-1 text-sm font-medium">
            {draw.prize_tier && (
              <span className="mr-1.5 rounded bg-blue-900/60 px-1.5 py-0.5 text-[10px] font-semibold text-blue-300">
                {draw.prize_tier.name}
              </span>
            )}
            {draw.entry.display_name || draw.entry.twitch_login}
          </span>
          <span className="text-[10px] text-white/30">{formatRelativeTime(draw.drawn_at)}</span>
        </div>
      ))}
    </div>
  )
}

function CsvUploadZone({
  raffleId,
  locked,
  onSuccess,
}: {
  raffleId: string
  locked: boolean
  onSuccess: (result: { imported: number; skipped: number }) => void
}) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [uploading, setUploading] = useState(false)
  const [result, setResult] = useState<{ imported: number; skipped: number } | null>(null)
  const [error, setError] = useState<string | null>(null)

  if (locked) {
    return (
      <div
        data-testid="csv-locked"
        className="rounded-xl border border-dashed border-white/10 bg-white/[.02] px-4 py-3 text-center"
      >
        <p className="text-sm text-white/30">名單已鎖定，無法再匯入</p>
      </div>
    )
  }

  async function handleFile(file: File) {
    setUploading(true)
    setError(null)
    try {
      const response = await importCSV(raffleId, file)
      setResult(response)
      onSuccess(response)
    } catch {
      setError('上傳失敗，請稍後再試')
    } finally {
      setUploading(false)
    }
  }

  return (
    <div
      role="button"
      aria-label="上傳 CSV"
      tabIndex={0}
      className="cursor-pointer rounded-xl border border-dashed border-amber-500/25 bg-amber-500/[.025] px-4 py-3 text-center transition hover:border-amber-500/50"
      onClick={() => inputRef.current?.click()}
      onKeyDown={(event) => { if (event.key === 'Enter' || event.key === ' ' || event.key === 'Spacebar' || event.code === 'Space') { event.preventDefault(); inputRef.current?.click() } }}
      onDrop={(event) => {
        event.preventDefault()
        const file = event.dataTransfer.files[0]
        if (file) void handleFile(file)
      }}
      onDragOver={(event) => event.preventDefault()}
    >
      <input
        ref={inputRef}
        data-testid="csv-input"
        type="file"
        accept=".csv"
        className="sr-only"
        onChange={(event) => {
          const file = event.target.files?.[0]
          if (file) void handleFile(file)
        }}
      />
      {uploading ? (
        <p className="text-sm text-amber-400/70">上傳中...</p>
      ) : (
        <>
          <p className="text-sm text-amber-400/80">點擊或拖曳 CSV 匯入參加者</p>
          <p className="text-[10px] text-white/30">欄位格式：twitch_login, display_name</p>
        </>
      )}
      {result && (
        <p data-testid="csv-success" className="mt-2 text-xs text-green-400">
          匯入成功：{result.imported} 人，略過 {result.skipped} 人
        </p>
      )}
      {error && (
        <p data-testid="csv-error" className="mt-2 text-xs text-red-400">
          {error}
        </p>
      )}
    </div>
  )
}

function DrawControls({
  status,
  exhausted,
  drawing,
  confirmEnd,
  ending,
  onDraw,
  onRequestEnd,
  onConfirmEnd,
  onCancelEnd,
}: {
  status: string
  exhausted: boolean
  drawing: boolean
  confirmEnd: boolean
  ending: boolean
  onDraw: () => void
  onRequestEnd: () => void
  onConfirmEnd: () => void
  onCancelEnd: () => void
}) {
  const isCompleted = status === 'completed'
  const drawDisabled = isCompleted || exhausted || drawing

  return (
    <div className="flex flex-col gap-2">
      <button
        data-testid="mgmt-draw-btn"
        disabled={drawDisabled}
        onClick={onDraw}
        className="w-full rounded-full bg-gradient-to-br from-amber-300 via-amber-500 to-amber-700 px-4 py-4 text-base font-black tracking-widest text-amber-950 shadow-lg shadow-amber-500/30 transition hover:-translate-y-0.5 disabled:cursor-not-allowed disabled:opacity-40"
      >
        {drawing ? '抽獎中...' : '抽出下一位'}
      </button>

      {!isCompleted && !confirmEnd && (
        <button
          data-testid="end-btn"
          onClick={onRequestEnd}
          className="w-full rounded-full border border-red-400/20 bg-transparent py-2 text-xs tracking-widest text-red-400/60 transition hover:border-red-400/50 hover:text-red-400/90"
        >
          結束活動
        </button>
      )}

      {confirmEnd && (
        <div
          data-testid="confirm-end"
          className="rounded-xl border border-amber-700/30 bg-amber-950/30 p-4 text-sm"
        >
          <p className="mb-3 font-medium text-amber-300">
            確定要結束活動嗎？結束後將無法繼續抽獎。
          </p>
          <div className="flex gap-2">
            <button
              data-testid="confirm-yes"
              disabled={ending}
              onClick={onConfirmEnd}
              className="flex-1 rounded-lg border border-red-500/40 bg-red-950/40 py-2 text-xs text-red-300 transition hover:bg-red-950/60 disabled:opacity-40"
            >
              {ending ? '結束中...' : '確定結束'}
            </button>
            <button
              data-testid="confirm-no"
              onClick={onCancelEnd}
              className="flex-1 rounded-lg border border-white/10 bg-white/5 py-2 text-xs text-white/50 transition hover:bg-white/10"
            >
              取消
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

function DiscordWebhookPanel({ raffleId }: { raffleId: string }) {
  const [url, setUrl] = useState('')
  const [configured, setConfigured] = useState<boolean | null>(null)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSave() {
    if (saving) return
    setSaving(true)
    setError(null)
    try {
      const result = await setDiscordWebhook(raffleId, url)
      setConfigured(result)
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setError(e?.response?.data?.error ?? '儲存失敗，請稍後再試')
    } finally {
      setSaving(false)
    }
  }

  async function handleClear() {
    if (saving) return
    setSaving(true)
    setError(null)
    try {
      const result = await setDiscordWebhook(raffleId, '')
      setConfigured(result)
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setError(e?.response?.data?.error ?? '清除失敗，請稍後再試')
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="rounded-xl border border-white/10 bg-white/[.04] p-4 space-y-3">
      <div className="flex items-center gap-2">
        <h2 className="text-[10px] uppercase tracking-widest text-white/30">Discord 通知</h2>
        {configured !== null && (
          <span
            data-testid="discord-webhook-status"
            className={`text-[10px] rounded-full px-2 py-0.5 ${configured ? 'bg-green-500/20 text-green-400' : 'bg-white/10 text-white/30'}`}
          >
            {configured ? '已設定' : '未設定'}
          </span>
        )}
      </div>
      <div className="flex gap-2">
        <input
          data-testid="discord-webhook-input"
          type="text"
          value={url}
          onInput={(e) => setUrl((e.target as HTMLInputElement).value)}
          onChange={(e) => setUrl(e.target.value)}
          placeholder="https://discord.com/api/webhooks/..."
          disabled={saving}
          className="flex-1 rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-xs text-white/80 placeholder:text-white/20 focus:outline-none focus:border-amber-500/50 disabled:opacity-40"
        />
        <button
          data-testid="discord-webhook-save"
          onClick={() => { void handleSave() }}
          disabled={saving}
          className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-400 transition hover:bg-amber-500/20 disabled:opacity-40"
        >
          {saving ? '...' : '儲存'}
        </button>
        <button
          data-testid="discord-webhook-clear"
          onClick={() => { void handleClear() }}
          disabled={saving}
          className="rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-xs text-white/40 transition hover:bg-white/10 disabled:opacity-40"
        >
          清除
        </button>
      </div>
      {error && (
        <p data-testid="discord-webhook-error" className="text-xs text-red-400">{error}</p>
      )}
    </section>
  )
}

interface OceanGachaMachineProps {
  shaking: boolean
  popBallColor: string | null
}

function OceanGachaMachine({ shaking, popBallColor }: OceanGachaMachineProps) {
  return (
    <div style={{ position: 'relative', width: '100%' }}>
      <img
        src={gachaBg}
        alt="扭蛋機"
        style={{
          width: '100%',
          height: 'auto',
          display: 'block',
          borderRadius: 12,
          filter: 'drop-shadow(0 0 50px rgba(30,100,255,.55)) drop-shadow(0 16px 60px rgba(0,0,0,.85))',
          animation: shaking ? 'oceanShake .5s ease-in-out' : undefined,
        }}
      />
      {popBallColor !== null && (
        <div
          style={{
            position: 'absolute',
            width: 44,
            height: 44,
            borderRadius: '50%',
            background: popBallColor,
            border: '2.5px solid rgba(255,255,255,.45)',
            boxShadow: '0 0 24px rgba(255,220,60,.95),inset 4px 4px 9px rgba(255,255,255,.3)',
            bottom: '22%',
            left: '50%',
            animation: 'oceanPopout 1.2s cubic-bezier(.22,.68,0,1.15) forwards',
            pointerEvents: 'none',
          }}
        />
      )}
    </div>
  )
}

const INITIAL_CHAT = [
  { emoji: '🐬', name: '小海豚', msg: '來啦來啦 🐬' },
  { emoji: '🏊', name: '海底探險家', msg: '抽我抽我！' },
  { emoji: '🐙', name: '章魚哥', msg: '希望中獎～' },
  { emoji: '🐚', name: '小貝殼', msg: '好期待 😄' },
  { emoji: '🪼', name: '水母寶寶', msg: '祝大家好運！' },
  { emoji: '🌊', name: '藍藍海', msg: '加油加油 💪' },
  { emoji: '🪸', name: '珊瑚小妹', msg: '衝衝衝！' },
]

const AUTO_CHAT = [
  { emoji: '🦀', name: '螃蟹哥', msg: '來啦！' },
  { emoji: '🐡', name: '河豚兒', msg: '好興奮～' },
  { emoji: '🦑', name: '魷魚妹', msg: '抽我抽我 🙏' },
  { emoji: '🐠', name: '熱帶魚', msg: '加油加油！' },
  { emoji: '🦈', name: '鯊魚君', msg: '必中！' },
  { emoji: '🐟', name: '小鯉魚', msg: '希望中獎～' },
]

interface ChatMsg { emoji: string; name: string; msg: string }

function ChatPanel() {
  const [messages, setMessages] = useState<ChatMsg[]>(INITIAL_CHAT)
  const autoIdx = useRef(0)

  useEffect(() => {
    const id = window.setInterval(() => {
      const next = AUTO_CHAT[autoIdx.current % AUTO_CHAT.length]
      autoIdx.current += 1
      setMessages(prev => [...prev.slice(-7), next])
    }, 2800)
    return () => window.clearInterval(id)
  }, [])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8, ...glassStyle, padding: '11px 13px', flex: 1, overflow: 'hidden' }}>
      <div style={{ fontSize: 12, fontWeight: 600, color: 'rgba(180,220,255,.8)', marginBottom: 4, display: 'flex', justifyContent: 'space-between' }}>
        聊天室 <span style={{ color: 'rgba(148,210,255,.3)' }}>›</span>
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 6, overflow: 'hidden' }}>
        {messages.map((m, i) => (
          <div key={i} style={{ display: 'flex', alignItems: 'flex-start', gap: 6, animation: i === messages.length - 1 ? 'oceanNewMsg .5s ease' : undefined }}>
            <div style={{ width: 22, height: 22, borderRadius: '50%', flexShrink: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 11, border: '1px solid rgba(255,255,255,.1)', background: 'rgba(100,200,255,.12)' }}>
              {m.emoji}
            </div>
            <div style={{ fontSize: 11, color: 'rgba(200,230,255,.85)', lineHeight: 1.4 }}>
              <span style={{ fontWeight: 700, color: 'rgba(147,210,255,.95)' }}>{m.name}：</span>
              {m.msg}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

interface WinnerModalProps {
  name: string | null
  onClose: () => void
}

function WinnerModal({ name, onClose }: WinnerModalProps) {
  if (name === null) return null
  return (
    <div
      onClick={onClose}
      style={{
        position: 'fixed', inset: 0, zIndex: 50,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        background: 'rgba(0,0,0,.65)',
        backdropFilter: 'blur(4px)',
        animation: 'oceanModalIn .4s ease forwards',
      }}
    >
      <div
        onClick={e => e.stopPropagation()}
        style={{
          background: 'rgba(4,20,70,.92)',
          border: '2px solid rgba(56,189,248,.4)',
          borderRadius: 20,
          padding: '32px 52px',
          textAlign: 'center',
          boxShadow: '0 0 60px rgba(14,165,233,.4)',
          animation: 'oceanBoxBounce .5s cubic-bezier(.22,.68,0,1.3)',
          color: '#fff',
        }}
      >
        <div style={{ fontSize: 52, marginBottom: 10 }}>🎉</div>
        <div style={{ fontSize: 13, color: 'rgba(148,210,255,.6)', letterSpacing: '.1em', marginBottom: 8 }}>恭喜中獎！</div>
        <div style={{ fontSize: 'clamp(24px,4vw,38px)', fontWeight: 900, textShadow: '0 0 20px rgba(56,189,248,.9)' }}>{name}</div>
        <button
          onClick={onClose}
          style={{
            marginTop: 18, padding: '8px 30px', borderRadius: 30,
            border: '1px solid rgba(56,189,248,.3)', background: 'transparent',
            color: 'rgba(148,210,255,.7)', fontSize: 12, cursor: 'pointer', letterSpacing: '.08em',
          }}
        >
          繼續抽獎
        </button>
      </div>
    </div>
  )
}


export default function RaffleDetailPage() {
  const { raffleId } = useParams()
  const navigate = useNavigate()
  const { query: { data, isLoading, isError } } = useOne<Raffle>({
    resource: 'raffles',
    id: raffleId,
    queryOptions: { enabled: Boolean(raffleId), retry: false },
  })
  const raffle = data?.data

  const [draws, setDraws] = useState<RaffleDraw[]>([])
  const [totalEntries, setTotalEntries] = useState<number | null>(null)
  const [exhausted, setExhausted] = useState(false)
  const [drawing, setDrawing] = useState(false)
  const [confirmEnd, setConfirmEnd] = useState(false)
  const [ending, setEnding] = useState(false)
  const [localCompleted, setLocalCompleted] = useState(false)
  const [localActivated, setLocalActivated] = useState(false)
  const [activating, setActivating] = useState(false)
  const [activateError, setActivateError] = useState<string | null>(null)
  const [shaking, setShaking] = useState(false)
  const [popBallColor, setPopBallColor] = useState<string | null>(null)
  const [modalWinner, setModalWinner] = useState<string | null>(null)
  const [tiers, setTiers] = useState<RafflePrizeTier[]>([])
  const [tierDrawing, setTierDrawing] = useState<Record<string, boolean>>({})
  const [tierExhausted, setTierExhausted] = useState<Record<string, boolean>>({})
  const [showAddTier, setShowAddTier] = useState(false)
  const [newTier, setNewTier] = useState({ name: '', prize_description: '', winner_count: 1 })
  const [addingTier, setAddingTier] = useState(false)
  const [addTierError, setAddTierError] = useState<string | null>(null)

  const pendingTimers = useRef<number[]>([])

  useEffect(() => {
    return () => { pendingTimers.current.forEach(id => window.clearTimeout(id)) }
  }, [])

  const effectiveStatus = localCompleted ? 'completed' : localActivated ? 'active' : (raffle?.status ?? '')

  const fetchTiers = useCallback(async () => {
    if (!raffleId) return
    try {
      setTiers(await listPrizeTiers(raffleId))
    } catch {
      setTiers([])
    }
  }, [raffleId])

  const fetchDraws = useCallback(async (): Promise<RaffleDraw[]> => {
    if (!raffleId) return []
    try {
      const result = await listDraws(raffleId)
      setDraws(result)
      return result
    } catch {
      setDraws([])
      return []
    }
  }, [raffleId])

  useEffect(() => {
    if (!raffleId) return
    const initialLoadId = window.setTimeout(() => {
      void fetchDraws()
      void fetchTiers()
    }, 0)

    if (effectiveStatus === 'completed') {
      return () => window.clearTimeout(initialLoadId)
    }

    const timerId = window.setInterval(() => {
      void fetchDraws()
    }, 5000)

    return () => {
      window.clearTimeout(initialLoadId)
      window.clearInterval(timerId)
    }
  }, [effectiveStatus, fetchDraws, fetchTiers, raffleId])

  async function handleDraw() {
    if (!raffleId || drawing) return
    setDrawing(true)

    setShaking(true)
    pendingTimers.current.push(window.setTimeout(() => setShaking(false), 550))

    pendingTimers.current.push(window.setTimeout(() => {
      const color = BALL_COLORS[Math.floor(Math.random() * BALL_COLORS.length)]
      setPopBallColor(color)
      pendingTimers.current.push(window.setTimeout(() => setPopBallColor(null), 1200))
    }, 400))

    try {
      await drawNext(raffleId)
      const result = await fetchDraws()
      const latest = result != null && result.length > 0
        ? [...result].sort((a, b) => new Date(b.drawn_at).getTime() - new Date(a.drawn_at).getTime())[0]
        : null
      if (latest) {
        const name = latest.entry.display_name || latest.entry.twitch_login
        pendingTimers.current.push(window.setTimeout(() => {
          setModalWinner(name)
          setDrawing(false)
        }, 900))
      } else {
        setDrawing(false)
      }
      setExhausted(false)
    } catch (error: unknown) {
      if (error && typeof error === 'object' && 'response' in error) {
        const response = (error as { response?: { status?: number } }).response
        if (response?.status === 409) setExhausted(true)
      }
      setDrawing(false)
    }
  }

  async function handleActivate() {
    if (!raffleId || activating) return
    setActivating(true)
    setActivateError(null)
    try {
      await activateRaffle(raffleId)
      setLocalActivated(true)
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setActivateError(e?.response?.data?.error ?? '啟動失敗，請稍後再試')
    } finally {
      setActivating(false)
    }
  }

  async function handleConfirmEnd() {
    if (!raffleId || ending) return

    setEnding(true)
    try {
      await completeRaffle(raffleId)
      setLocalCompleted(true)
      setConfirmEnd(false)
    } finally {
      setEnding(false)
    }
  }

  async function handleAddTier() {
    if (!raffleId || addingTier || !newTier.name.trim() || newTier.winner_count < 1) return
    setAddingTier(true)
    setAddTierError(null)
    try {
      await createPrizeTier(raffleId, newTier)
      await fetchTiers()
      setNewTier({ name: '', prize_description: '', winner_count: 1 })
      setShowAddTier(false)
    } catch (err: unknown) {
      const e = err as { response?: { data?: { error?: string } } }
      setAddTierError(e?.response?.data?.error ?? '新增失敗，請稍後再試')
    } finally {
      setAddingTier(false)
    }
  }

  async function handleDrawFromTier(tierId: string) {
    if (!raffleId || tierDrawing[tierId]) return
    setTierDrawing(prev => ({ ...prev, [tierId]: true }))
    try {
      await drawFromTier(raffleId, tierId)
      await Promise.all([fetchTiers(), fetchDraws()])
      setTierExhausted(prev => ({ ...prev, [tierId]: false }))
    } catch (error: unknown) {
      const res = (error as { response?: { status?: number } }).response
      if (res?.status === 409) setTierExhausted(prev => ({ ...prev, [tierId]: true }))
    } finally {
      setTierDrawing(prev => ({ ...prev, [tierId]: false }))
    }
  }

  async function handleDeleteTier(tierId: string) {
    if (!raffleId) return
    try {
      await deletePrizeTier(raffleId, tierId)
      await fetchTiers()
    } catch {
      // tier has draws — ignore silently
    }
  }

  if (isLoading) {
    return <Skeleton data-testid="skeleton" className="h-96 w-full" />
  }

  if (isError || !raffle) {
    return (
      <div className="rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
        無法載入抽獎活動
      </div>
    )
  }

  const entryCount = totalEntries
  const drawnCount = draws.length
  const remaining = entryCount === null ? null : Math.max(entryCount - drawnCount, 0)

  return (
    <div style={{ background: '#050e24', color: '#f0f9ff', minHeight: '100vh', fontFamily: 'var(--font-sans, sans-serif)' }}>
      <style>{OCEAN_KEYFRAMES}</style>

      {/* Management section (collapsible) */}
      <details style={{ borderBottom: '1px solid rgba(80,160,255,.15)' }}>
        <summary style={{ padding: '10px 20px', cursor: 'pointer', fontSize: 12, color: 'rgba(148,210,255,.6)', letterSpacing: '.08em', userSelect: 'none' }}>
          ⚙ 活動管理 — {raffle.title}
          <span style={{ marginLeft: 8, background: effectiveStatus === 'active' ? 'rgba(34,197,94,.12)' : 'rgba(148,163,184,.1)', color: effectiveStatus === 'active' ? '#4ade80' : 'rgba(148,163,184,.7)', padding: '2px 8px', borderRadius: 99, fontSize: 10, border: `1px solid ${effectiveStatus === 'active' ? 'rgba(34,197,94,.3)' : 'rgba(148,163,184,.2)'}` }}>
            {statusLabel[effectiveStatus as RaffleStatus] ?? effectiveStatus}
          </span>
        </summary>
        <div style={{ padding: '12px 20px 16px', display: 'flex', flexDirection: 'column', gap: 12 }}>
          <button onClick={() => navigate('/raffles')} style={{ alignSelf: 'flex-start', background: 'none', border: 'none', color: 'rgba(148,210,255,.5)', fontSize: 12, cursor: 'pointer' }}>
            ‹ 返回抽獎列表
          </button>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3,1fr)', gap: 8 }}>
            <StatCard label="匯入人數" value={entryCount?.toString() ?? '--'} colorClass="text-blue-400" />
            <StatCard label="已抽出" value={drawnCount.toString()} colorClass="text-green-400" />
            <StatCard label="剩餘" value={remaining?.toString() ?? '--'} colorClass="text-amber-400" />
          </div>

          <CsvUploadZone
            raffleId={raffle.id}
            locked={effectiveStatus !== 'draft'}
            onSuccess={(result) => { setTotalEntries(prev => (prev ?? 0) + result.imported); setExhausted(false) }}
          />

          {effectiveStatus === 'draft' && (
            <>
              <button
                data-testid="activate-btn"
                disabled={activating}
                onClick={() => { void handleActivate() }}
                style={{ width: '100%', padding: '9px', borderRadius: 8, border: '1px solid rgba(251,191,36,.35)', background: 'rgba(251,191,36,.08)', color: '#fbbf24', fontSize: 12, fontWeight: 600, cursor: 'pointer' }}
              >
                {activating ? '鎖定中...' : '開始抽獎（鎖定名單）'}
              </button>
              {activateError && <p data-testid="activate-error" style={{ fontSize: 12, color: '#f87171', textAlign: 'center' }}>{activateError}</p>}
            </>
          )}

          <DiscordWebhookPanel raffleId={raffle.id} />

          <DrawControls
            status={effectiveStatus}
            exhausted={exhausted || remaining === 0}
            drawing={drawing}
            confirmEnd={confirmEnd}
            ending={ending}
            onDraw={() => { void handleDraw() }}
            onRequestEnd={() => setConfirmEnd(true)}
            onConfirmEnd={() => { void handleConfirmEnd() }}
            onCancelEnd={() => setConfirmEnd(false)}
          />
        </div>
      </details>

      {/* Title */}
      <div style={{ textAlign: 'center', padding: '20px 20px 8px' }}>
        <div style={{ display: 'flex', justifyContent: 'center', gap: 7, marginBottom: 4 }}>
          {['★','★','★','★','★'].map((s, i) => (
            <span key={i} style={{ color: i === 2 ? '#93c5fd' : '#60a5fa', fontSize: i === 2 ? 20 : 14, filter: 'drop-shadow(0 0 5px rgba(96,165,250,.9))', animation: `oceanTwinkle 2s ease-in-out ${i * 0.15}s infinite` }}>{s}</span>
          ))}
        </div>
        <div style={{ fontSize: 'clamp(26px,4.5vw,50px)', fontWeight: 900, letterSpacing: '.05em', textShadow: '0 0 22px rgba(56,189,248,.95),0 0 55px rgba(56,189,248,.4)' }}>
          觀眾抽獎
        </div>
        <div style={{ display: 'inline-block', marginTop: 6, background: 'rgba(10,60,160,.5)', border: '1px solid rgba(56,189,248,.3)', borderRadius: 20, padding: '3px 16px', fontSize: 12, color: 'rgba(150,220,255,.9)', letterSpacing: '.06em' }}>
          扭蛋抽出幸運觀眾！
        </div>
      </div>

      {/* Three-column layout */}
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 14, padding: '0 16px 20px', maxWidth: 1300, margin: '0 auto' }}>

        {/* Left: Chat */}
        <div style={{ width: 160, flexShrink: 0, display: 'flex', flexDirection: 'column', gap: 10 }}>
          <div style={{ ...glassStyle, padding: '11px 13px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 7, fontSize: 12, fontWeight: 700, color: '#e0f2fe', marginBottom: 4 }}>
              <div style={{ width: 24, height: 24, borderRadius: '50%', background: 'rgba(56,189,248,.15)', border: '1px solid rgba(56,189,248,.3)', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 11 }}>
                👤
              </div>
              參加方式
            </div>
            <div style={{ fontSize: 11, color: 'rgba(148,210,255,.7)', lineHeight: 1.55 }}>在聊天室留言即可參加抽獎！</div>
          </div>
          <ChatPanel />
        </div>

        {/* Center: Gacha machine + draw button */}
        <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 14 }}>
          <OceanGachaMachine shaking={shaking} popBallColor={popBallColor} />
          <button
            data-testid="draw-btn"
            disabled={effectiveStatus === 'completed' || exhausted || remaining === 0 || drawing}
            onClick={() => { void handleDraw() }}
            style={{
              padding: '13px 60px', borderRadius: 50, border: 'none',
              background: 'linear-gradient(90deg,#0ea5e9,#2563eb)',
              color: '#e0f2fe', fontSize: 'clamp(15px,2.2vw,20px)', fontWeight: 900,
              letterSpacing: '.12em', cursor: 'pointer',
              boxShadow: '0 0 35px rgba(14,165,233,.75),0 0 70px rgba(14,165,233,.2),0 4px 16px rgba(0,0,0,.5)',
              opacity: (effectiveStatus === 'completed' || exhausted || remaining === 0 || drawing) ? .5 : 1,
            }}
          >
            {drawing ? '抽獎中...' : '開始抽獎'}
          </button>
          <div style={{ fontSize: 11, color: 'rgba(148,210,255,.5)', textAlign: 'center' }}>
            將從參加者中隨機抽出一位幸運觀眾！
          </div>
        </div>

        {/* Right: Winner list */}
        <div style={{ width: 160, flexShrink: 0 }}>
          <div style={{ ...glassStyle, padding: '13px', display: 'flex', flexDirection: 'column', alignItems: 'center', minHeight: 300 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 7, fontSize: 12, fontWeight: 700, color: '#e0f2fe', marginBottom: 14, letterSpacing: '.04em' }}>
              🐚 中獎名單 🐚
            </div>
            <WinnerList draws={draws} />
            <div style={{ marginTop: 'auto', fontSize: 24, opacity: .45, animation: 'oceanFloat 3s ease-in-out infinite' }}>
              🪼
            </div>
          </div>
        </div>
      </div>

      {/* Prize Tiers */}
      {effectiveStatus === 'active' && (
        <div style={{ maxWidth: 1300, margin: '0 auto 2rem', padding: '0 16px' }}>
          <div style={{ ...glassStyle, padding: '16px 20px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
              <span style={{ fontSize: 13, fontWeight: 700, color: '#e0f2fe', letterSpacing: '.04em' }}>🏆 獎項設定</span>
              <button
                onClick={() => setShowAddTier(v => !v)}
                style={{ background: 'rgba(56,189,248,.12)', border: '1px solid rgba(56,189,248,.3)', color: '#7dd3fc', borderRadius: 6, padding: '4px 12px', fontSize: 12, cursor: 'pointer' }}
              >
                {showAddTier ? '取消' : '+ 新增獎項'}
              </button>
            </div>

            {showAddTier && (
              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'flex-end', marginBottom: 12, padding: '10px 12px', background: 'rgba(56,189,248,.06)', borderRadius: 8, border: '1px solid rgba(56,189,248,.15)' }}>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
                  <label style={{ fontSize: 10, color: 'rgba(148,210,255,.6)' }}>獎項名稱</label>
                  <input value={newTier.name} onChange={e => setNewTier(p => ({ ...p, name: e.target.value }))} placeholder="例：一等獎" style={{ background: 'rgba(255,255,255,.07)', border: '1px solid rgba(80,160,255,.25)', borderRadius: 5, color: '#e0f2fe', fontSize: 12, padding: '5px 8px' }} />
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
                  <label style={{ fontSize: 10, color: 'rgba(148,210,255,.6)' }}>獎品描述</label>
                  <input value={newTier.prize_description} onChange={e => setNewTier(p => ({ ...p, prize_description: e.target.value }))} placeholder="例：Switch 主機" style={{ background: 'rgba(255,255,255,.07)', border: '1px solid rgba(80,160,255,.25)', borderRadius: 5, color: '#e0f2fe', fontSize: 12, padding: '5px 8px' }} />
                </div>
                <div style={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
                  <label style={{ fontSize: 10, color: 'rgba(148,210,255,.6)' }}>抽幾人</label>
                  <input type="number" min={1} value={newTier.winner_count} onChange={e => setNewTier(p => ({ ...p, winner_count: Number(e.target.value) }))} style={{ width: 60, background: 'rgba(255,255,255,.07)', border: '1px solid rgba(80,160,255,.25)', borderRadius: 5, color: '#e0f2fe', fontSize: 12, padding: '5px 8px' }} />
                </div>
                <button onClick={() => { void handleAddTier() }} disabled={addingTier || !newTier.name.trim()} style={{ background: 'rgba(34,197,94,.15)', border: '1px solid rgba(34,197,94,.3)', color: '#4ade80', borderRadius: 6, padding: '6px 14px', fontSize: 12, cursor: 'pointer', opacity: (addingTier || !newTier.name.trim()) ? .5 : 1 }}>
                  {addingTier ? '新增中...' : '確認新增'}
                </button>
                {addTierError && (
                  <p style={{ fontSize: 11, color: '#f87171', marginTop: 6, width: '100%' }}>{addTierError}</p>
                )}
              </div>
            )}

            {tiers.length === 0 && !showAddTier && (
              <p style={{ fontSize: 12, color: 'rgba(148,210,255,.4)', textAlign: 'center', padding: '8px 0' }}>尚未設定獎項</p>
            )}

            {tiers.map(tier => {
              const isDone = tier.drawn_count >= tier.winner_count
              const isDrawing = tierDrawing[tier.id] ?? false
              const isExhausted = tierExhausted[tier.id] ?? false
              return (
                <div key={tier.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 12px', borderRadius: 8, marginBottom: 6, background: isDone ? 'rgba(255,255,255,.03)' : 'rgba(56,189,248,.05)', border: `1px solid ${isDone ? 'rgba(255,255,255,.08)' : 'rgba(56,189,248,.18)'}` }}>
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                    <span style={{ fontSize: 13, fontWeight: 600, color: isDone ? 'rgba(148,210,255,.5)' : '#e0f2fe' }}>{tier.name}</span>
                    {tier.prize_description && <span style={{ fontSize: 11, color: 'rgba(148,210,255,.5)' }}>{tier.prize_description}</span>}
                    <span style={{ fontSize: 10, color: isDone ? '#4ade80' : 'rgba(148,210,255,.4)' }}>{tier.drawn_count} / {tier.winner_count} 人已抽出</span>
                  </div>
                  <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                    <button onClick={() => { void handleDrawFromTier(tier.id) }} disabled={isDone || isDrawing || isExhausted} style={{ background: 'rgba(14,165,233,.15)', border: '1px solid rgba(14,165,233,.3)', color: '#38bdf8', borderRadius: 6, padding: '5px 12px', fontSize: 11, cursor: 'pointer', opacity: (isDone || isDrawing || isExhausted) ? .4 : 1 }}>
                      {isDrawing ? '抽中...' : isExhausted ? '已抽完' : '抽一人'}
                    </button>
                    {tier.drawn_count === 0 && (
                      <button onClick={() => { void handleDeleteTier(tier.id) }} style={{ background: 'rgba(239,68,68,.1)', border: '1px solid rgba(239,68,68,.25)', color: '#f87171', borderRadius: 6, padding: '5px 10px', fontSize: 11, cursor: 'pointer' }}>✕</button>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      <WinnerModal name={modalWinner} onClose={() => setModalWinner(null)} />
    </div>
  )
}
