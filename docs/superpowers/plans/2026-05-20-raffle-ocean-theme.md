# RaffleDetailPage 海洋主題改版 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 將 `RaffleDetailPage` 改版為海洋主題三欄佈局，以插圖扭蛋機為中央主視覺，保留所有現有管理功能與測試覆蓋。

**Architecture:** 在同一個 `RaffleDetailPage.tsx` 檔案內重構 JSX 結構與 inline 樣式，不新增外部 CSS 檔。管理功能（CSV、啟動、統計、Discord、結束）移至頂部折疊區塊。主體改為三欄：左欄模擬聊天、中欄圖片主視覺 + 抽獎按鈕、右欄中獎名單。所有現有 `data-testid` 屬性完整保留，確保既有測試零改動通過。

**Tech Stack:** React 18, TypeScript, inline styles（延續現有慣例），`apps/dashboard/src/assets/raffle-bg.jpg.png`

---

## 檔案異動

| 檔案 | 動作 |
|---|---|
| `apps/dashboard/src/pages/RaffleDetailPage.tsx` | 主要改版 |
| `apps/dashboard/src/assets/raffle-bg.jpg.png` | 已存在，無需異動 |
| `apps/dashboard/src/pages/__tests__/RaffleDetailPage.test.tsx` | 不改動，全數通過為驗收標準 |

---

## Task 1：確認測試基線通過

在改版前確認所有現有測試通過，建立 baseline。

**Files:**
- Read: `apps/dashboard/src/pages/__tests__/RaffleDetailPage.test.tsx`

- [ ] **Step 1：執行現有測試**

```bash
docker compose run --no-deps --rm dashboard pnpm test --run apps/dashboard/src/pages/__tests__/RaffleDetailPage.test.tsx
```

預期：全部 PASS（約 20 個測試）。若有失敗，先修復再繼續。

- [ ] **Step 2：記錄通過的測試數量**

確認輸出包含類似：`Test Files 1 passed | Tests XX passed`

---

## Task 2：新增動畫 keyframe 常數與 CSS helper

在 `RaffleDetailPage.tsx` 頂部新增 CSS keyframe 字串與 glass panel style 常數，供後續 Task 使用。

**Files:**
- Modify: `apps/dashboard/src/pages/RaffleDetailPage.tsx`

- [ ] **Step 1：在檔案頂部（imports 之後、第一個 interface 之前）插入以下常數**

```tsx
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
```

- [ ] **Step 2：執行測試確認無 TypeScript 錯誤**

```bash
docker compose run --no-deps --rm dashboard pnpm test --run apps/dashboard/src/pages/__tests__/RaffleDetailPage.test.tsx
```

預期：全部 PASS（常數不影響行為）。

---

## Task 3：新增 `OceanGachaMachine` 元件（圖片主視覺）

替換現有純 CSS `GachaMachine` 元件，改為圖片 + 彈出球 overlay。

**Files:**
- Modify: `apps/dashboard/src/pages/RaffleDetailPage.tsx`

- [ ] **Step 1：在 `GachaMachine` 元件定義位置，將整個 `GachaMachine` function 替換為 `OceanGachaMachine`**

```tsx
import gachaBg from '../assets/raffle-bg.jpg.png'

const BALL_COLORS = [
  'linear-gradient(135deg,#93c5fd,#3b82f6)',
  'linear-gradient(135deg,#d8b4fe,#8b5cf6)',
  'linear-gradient(135deg,#86efac,#22c55e)',
  'linear-gradient(135deg,#fde68a,#f59e0b)',
  'linear-gradient(135deg,#f9a8d4,#ec4899)',
  'linear-gradient(135deg,#a5f3fc,#06b6d4)',
]

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
```

- [ ] **Step 2：執行測試確認無破壞**

```bash
docker compose run --no-deps --rm dashboard pnpm test --run apps/dashboard/src/pages/__tests__/RaffleDetailPage.test.tsx
```

預期：全部 PASS（`OceanGachaMachine` 尚未整合進主體）。

---

## Task 4：新增 `ChatPanel` 元件（模擬即時聊天）

**Files:**
- Modify: `apps/dashboard/src/pages/RaffleDetailPage.tsx`

- [ ] **Step 1：在 `OceanGachaMachine` 之後新增 `ChatPanel` 元件**

```tsx
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
  const [messages, setMessages] = React.useState<ChatMsg[]>(INITIAL_CHAT)
  const autoIdx = React.useRef(0)

  React.useEffect(() => {
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
```

- [ ] **Step 2：執行測試確認無破壞**

```bash
docker compose run --no-deps --rm dashboard pnpm test --run apps/dashboard/src/pages/__tests__/RaffleDetailPage.test.tsx
```

預期：全部 PASS。

---

## Task 5：新增 `WinnerModal` 元件（抽獎彈窗）

**Files:**
- Modify: `apps/dashboard/src/pages/RaffleDetailPage.tsx`

- [ ] **Step 1：在 `ChatPanel` 之後新增 `WinnerModal` 元件**

```tsx
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
```

- [ ] **Step 2：執行測試確認無破壞**

```bash
docker compose run --no-deps --rm dashboard pnpm test --run apps/dashboard/src/pages/__tests__/RaffleDetailPage.test.tsx
```

預期：全部 PASS。

---

## Task 6：重構 `RaffleDetailPage` 主體佈局

將主體 JSX 改為三欄海洋主題佈局，保留所有管理功能於頂部折疊區塊，保留全部 `data-testid`。

**Files:**
- Modify: `apps/dashboard/src/pages/RaffleDetailPage.tsx`

- [ ] **Step 1：在元件頂部新增 draw animation state（`shaking`, `popBallColor`, `modalWinner`）**

在 `RaffleDetailPage` function 內，現有 state 宣告之後新增：

```tsx
const [shaking, setShaking] = React.useState(false)
const [popBallColor, setPopBallColor] = React.useState<string | null>(null)
const [modalWinner, setModalWinner] = React.useState<string | null>(null)
```

- [ ] **Step 2：修改 `handleDraw` 加入動畫觸發**

將現有 `handleDraw` 替換為：

```tsx
async function handleDraw() {
  if (!raffleId || drawing) return
  setDrawing(true)

  // 震動動畫
  setShaking(true)
  window.setTimeout(() => setShaking(false), 550)

  // 彩球彈出
  window.setTimeout(() => {
    const color = BALL_COLORS[Math.floor(Math.random() * BALL_COLORS.length)]
    setPopBallColor(color)
    window.setTimeout(() => setPopBallColor(null), 1200)
  }, 400)

  try {
    await drawNext(raffleId)
    const result = await fetchDraws()
    // 取最新得獎者名字顯示於 modal
    const latest = result?.[0]
    if (latest) {
      const name = latest.entry.display_name || latest.entry.twitch_login
      window.setTimeout(() => setModalWinner(name), 900)
    }
    setExhausted(false)
  } catch (error: unknown) {
    if (error && typeof error === 'object' && 'response' in error) {
      const response = (error as { response?: { status?: number } }).response
      if (response?.status === 409) setExhausted(true)
    }
  } finally {
    setDrawing(false)
  }
}
```

注意：`fetchDraws` 需要回傳 draws 陣列，將現有 `fetchDraws` 修改為有回傳值：

```tsx
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
```

- [ ] **Step 3：將主體 return JSX 替換為三欄海洋主題佈局**

完整替換 `RaffleDetailPage` 的 return 區塊（從 `if (isLoading)` 判斷之後的 return）：

```tsx
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

    {/* ── 頂部管理區（折疊） ── */}
    <details style={{ borderBottom: '1px solid rgba(80,160,255,.15)' }}>
      <summary style={{ padding: '10px 20px', cursor: 'pointer', fontSize: 12, color: 'rgba(148,210,255,.6)', letterSpacing: '.08em', userSelect: 'none' }}>
        ⚙ 活動管理 — {raffle.title}
        <span style={{ marginLeft: 8, ...{ background: effectiveStatus === 'active' ? 'rgba(34,197,94,.12)' : 'rgba(148,163,184,.1)', color: effectiveStatus === 'active' ? '#4ade80' : 'rgba(148,163,184,.7)', padding: '2px 8px', borderRadius: 99, fontSize: 10, border: `1px solid ${effectiveStatus === 'active' ? 'rgba(34,197,94,.3)' : 'rgba(148,163,184,.2)'}` } }}>
          {statusLabel[effectiveStatus as RaffleStatus] ?? effectiveStatus}
        </span>
      </summary>
      <div style={{ padding: '12px 20px 16px', display: 'flex', flexDirection: 'column', gap: 12 }}>
        <button onClick={() => navigate('/raffles')} style={{ alignSelf: 'flex-start', background: 'none', border: 'none', color: 'rgba(148,210,255,.5)', fontSize: 12, cursor: 'pointer' }}>
          ‹ 返回抽獎列表
        </button>

        {/* StatCards */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3,1fr)', gap: 8 }}>
          <StatCard label="匯入人數" value={entryCount?.toString() ?? '--'} colorClass="text-blue-400" />
          <StatCard label="已抽出" value={drawnCount.toString()} colorClass="text-green-400" />
          <StatCard label="剩餘" value={remaining?.toString() ?? '--'} colorClass="text-amber-400" />
        </div>

        {/* CSV */}
        <CsvUploadZone
          raffleId={raffle.id}
          locked={effectiveStatus !== 'draft'}
          onSuccess={(result) => { setTotalEntries(prev => (prev ?? 0) + result.imported); setExhausted(false) }}
        />

        {/* Activate */}
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

        {/* Discord */}
        <DiscordWebhookPanel raffleId={raffle.id} />

        {/* End */}
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

    {/* ── 主視覺標題 ── */}
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

    {/* ── 三欄主體 ── */}
    <div style={{ display: 'flex', alignItems: 'flex-start', gap: 14, padding: '0 16px 20px', maxWidth: 1300, margin: '0 auto' }}>

      {/* 左欄：聊天室 */}
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

      {/* 中欄：扭蛋機 */}
      <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 14 }}>
        <OceanGachaMachine shaking={shaking} popBallColor={popBallColor} />

        {/* 抽獎按鈕（draw-btn testid 保留） */}
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

      {/* 右欄：中獎名單 */}
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

    {/* 中獎彈窗 */}
    <WinnerModal name={modalWinner} onClose={() => setModalWinner(null)} />
  </div>
)
```

- [ ] **Step 4：執行測試確認全部通過**

```bash
docker compose run --no-deps --rm dashboard pnpm test --run apps/dashboard/src/pages/__tests__/RaffleDetailPage.test.tsx
```

預期：全部 PASS。若有失敗，檢查 `data-testid` 是否遺漏。

- [ ] **Step 5：commit**

```bash
git add apps/dashboard/src/pages/RaffleDetailPage.tsx apps/dashboard/src/assets/raffle-bg.jpg.png
git commit -m "feat: redesign RaffleDetailPage with ocean theme

refs #232

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 7：新增抽獎彈窗測試

**Files:**
- Modify: `apps/dashboard/src/pages/__tests__/RaffleDetailPage.test.tsx`

- [ ] **Step 1：在檔案末尾新增 modal 測試 describe 區塊**

```tsx
describe('RaffleDetailPage — winner modal', () => {
  it('shows winner modal after draw completes', async () => {
    vi.mocked(rafflesService.drawNext).mockResolvedValue(mockDraw)
    vi.mocked(rafflesService.listDraws).mockResolvedValue([mockDraw])
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="draw-btn"]')).toBeTruthy())

    await act(async () => {
      container.querySelector('[data-testid="draw-btn"]')!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    // modal 在抽完後 900ms 出現，需等候
    await waitFor(() => expect(container.textContent).toContain('恭喜中獎'), { timeout: 2000 })
    expect(container.textContent).toContain('Viewer One')
    cleanup(root, container)
  })

  it('closes modal when 繼續抽獎 button clicked', async () => {
    vi.mocked(rafflesService.drawNext).mockResolvedValue(mockDraw)
    vi.mocked(rafflesService.listDraws).mockResolvedValue([mockDraw])
    const dp = createMockDataProvider({
      getOne: { raffles: vi.fn().mockResolvedValue(mockRaffle as BaseRecord) },
    })
    const { container, root } = await renderAt('r1', dp)
    await waitFor(() => expect(container.querySelector('[data-testid="draw-btn"]')).toBeTruthy())

    await act(async () => {
      container.querySelector('[data-testid="draw-btn"]')!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    await waitFor(() => expect(container.textContent).toContain('恭喜中獎'), { timeout: 2000 })

    const closeBtn = Array.from(container.querySelectorAll('button')).find(b => b.textContent?.includes('繼續抽獎'))
    await act(async () => {
      closeBtn!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    })
    expect(container.textContent).not.toContain('恭喜中獎')
    cleanup(root, container)
  })
})
```

- [ ] **Step 2：執行全部測試**

```bash
docker compose run --no-deps --rm dashboard pnpm test --run apps/dashboard/src/pages/__tests__/RaffleDetailPage.test.tsx
```

預期：原有 20 個測試 + 新增 2 個，共 22 個全部 PASS。

- [ ] **Step 3：commit**

```bash
git add apps/dashboard/src/pages/__tests__/RaffleDetailPage.test.tsx
git commit -m "test: add winner modal tests for ocean theme raffle page

refs #232

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## 自我審查

**Spec coverage:**
- [x] 三欄佈局（左 160px / 中 flex:1 / 右 160px）— Task 6
- [x] 扭蛋機圖片主視覺完整顯示 — Task 3
- [x] 震動 + 彈出球 + 彈窗 — Task 6
- [x] 聊天室自動滾動 — Task 4
- [x] 右欄中獎名單 — Task 6（使用現有 `WinnerList`）
- [x] 所有管理功能保留（CSV、鎖定、統計、Discord、結束）— Task 6 折疊區塊
- [x] TypeScript 型別無錯誤 — 每 Task 均執行測試

**Placeholder scan:** 無 TBD / TODO。

**Type consistency:**
- `OceanGachaMachineProps.shaking: boolean` ← `useState<boolean>` ✓
- `OceanGachaMachineProps.popBallColor: string | null` ← `useState<string | null>` ✓
- `WinnerModalProps.name: string | null` ← `useState<string | null>` ✓
- `fetchDraws` 回傳 `Promise<RaffleDraw[]>` ← Task 6 Step 2 使用 `result?.[0]` ✓
