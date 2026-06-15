# Raffle Phase 2 — 逐輪抽獎模式 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 Dashboard 抽獎頁面新增逐輪模式，讓主播依 PrizeTier position 順序一輪一輪手動推進抽獎，每輪結束後展示得獎者，所有輪次結束後顯示完整名單。

**Architecture:** 純前端工作，後端 cross-tier 排除已完整（`raffle_service.go:1264`）。新增 `RaffleDrawSession` 元件管理逐輪狀態機；`RaffleDetailPage` 在 raffle 狀態為 `active` 且有 PrizeTier 時，以 `RaffleDrawSession` 取代現有的逐顆「抽一人」按鈕區塊。

**Tech Stack:** React 18, TypeScript, Vitest + Testing Library, inline styles（沿用 ocean 主題）

---

## 檔案結構

| 動作 | 路徑 | 職責 |
|---|---|---|
| **新增** | `apps/dashboard/src/components/raffle/RaffleDrawSession.tsx` | 逐輪模式容器 + 子元件（TierRoundCard、DrawResultOverlay、FinalWinnerList） |
| **新增** | `apps/dashboard/src/components/raffle/__tests__/RaffleDrawSession.test.tsx` | 元件測試 |
| **修改** | `apps/dashboard/src/services/raffles.ts` | 補 `position` 欄位至 `RafflePrizeTier` interface |
| **修改** | `apps/dashboard/src/pages/RaffleDetailPage.tsx` | active 狀態下嵌入 `RaffleDrawSession` |

---

## Task 1：補 `RafflePrizeTier.position` 欄位

後端 `raffle_prize_tiers` 表有 `position` 欄位但前端 interface 缺少，需先補上。

**Files:**
- Modify: `apps/dashboard/src/services/raffles.ts:35-43`

- [ ] **Step 1：確認後端欄位存在**

```bash
grep "position" services/api/internal/models/raffle.go
```

預期輸出含：`Position int \`gorm:\"not null;default:0\" json:\"position\"\``

- [ ] **Step 2：補 interface 欄位**

在 `apps/dashboard/src/services/raffles.ts` 找到 `RafflePrizeTier` interface，加入 `position`:

```ts
export interface RafflePrizeTier {
  id: string
  raffle_id: string
  name: string
  prize_description: string
  winner_count: number
  drawn_count: number
  position: number
  created_at: string
}
```

- [ ] **Step 3：TypeScript 確認無錯誤**

```bash
cd apps/dashboard && npx tsc --noEmit
```

預期：無錯誤輸出。

- [ ] **Step 4：Commit**

```bash
git add apps/dashboard/src/services/raffles.ts
git commit -m "feat: add position field to RafflePrizeTier interface

refs #1043"
```

---

## Task 2：建立 `RaffleDrawSession` — 初始狀態與 round_ready

**Files:**
- Create: `apps/dashboard/src/components/raffle/RaffleDrawSession.tsx`
- Create: `apps/dashboard/src/components/raffle/__tests__/RaffleDrawSession.test.tsx`

- [ ] **Step 1：建立測試檔，寫第一個失敗測試**

建立 `apps/dashboard/src/components/raffle/__tests__/RaffleDrawSession.test.tsx`：

```tsx
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { RaffleDrawSession } from '../RaffleDrawSession'
import type { RafflePrizeTier } from '@/services/raffles'

const mockTiers: RafflePrizeTier[] = [
  { id: 't1', raffle_id: 'r1', name: '一等獎', prize_description: 'Switch', winner_count: 1, drawn_count: 0, position: 0, created_at: '' },
  { id: 't2', raffle_id: 'r1', name: '二等獎', prize_description: 'AirPods', winner_count: 2, drawn_count: 0, position: 1, created_at: '' },
]

describe('RaffleDrawSession', () => {
  it('顯示第一輪的獎項名稱與描述', () => {
    render(<RaffleDrawSession raffleId="r1" tiers={mockTiers} />)
    expect(screen.getByTestId('current-tier-name')).toHaveTextContent('一等獎')
    expect(screen.getByTestId('current-tier-description')).toHaveTextContent('Switch')
    expect(screen.getByTestId('draw-round-button')).toBeInTheDocument()
  })

  it('顯示輪次進度（第 1 / 2 輪）', () => {
    render(<RaffleDrawSession raffleId="r1" tiers={mockTiers} />)
    expect(screen.getByTestId('round-progress')).toHaveTextContent('第 1 / 2 輪')
  })
})
```

- [ ] **Step 2：確認測試失敗**

```bash
cd apps/dashboard && npx vitest run src/components/raffle/__tests__/RaffleDrawSession.test.tsx
```

預期：FAIL — `RaffleDrawSession` 不存在。

- [ ] **Step 3：建立元件檔，實作 round_ready 初始狀態**

建立 `apps/dashboard/src/components/raffle/RaffleDrawSession.tsx`：

```tsx
import { useState } from 'react'
import type { RaffleDraw, RafflePrizeTier } from '@/services/raffles'
import { drawFromTier } from '@/services/raffles'

type SessionPhase = 'round_ready' | 'drawing' | 'round_result' | 'session_complete'

interface Props {
  raffleId: string
  tiers: RafflePrizeTier[]
}

const glass: React.CSSProperties = {
  background: 'rgba(4,14,52,.55)',
  border: '1px solid rgba(80,160,255,.22)',
  borderRadius: 14,
  backdropFilter: 'blur(14px)',
  WebkitBackdropFilter: 'blur(14px)',
}

export function RaffleDrawSession({ raffleId, tiers }: Props) {
  const sorted = [...tiers].sort((a, b) => a.position - b.position)
  const [currentIndex, setCurrentIndex] = useState(0)
  const [phase, setPhase] = useState<SessionPhase>('round_ready')
  const [latestDraw, setLatestDraw] = useState<RaffleDraw | null>(null)
  const [error, setError] = useState<string | null>(null)

  const currentTier = sorted[currentIndex]

  async function handleDraw() {
    if (!currentTier) return
    setPhase('drawing')
    setError(null)
    try {
      const draw = await drawFromTier(raffleId, currentTier.id)
      setLatestDraw(draw)
      setPhase('round_result')
    } catch {
      setError('抽獎失敗，請再試一次')
      setPhase('round_ready')
    }
  }

  function handleNext() {
    const nextIndex = currentIndex + 1
    if (nextIndex >= sorted.length) {
      setPhase('session_complete')
    } else {
      setCurrentIndex(nextIndex)
      setPhase('round_ready')
    }
  }

  if (phase === 'session_complete') {
    return (
      <div data-testid="session-complete" style={{ ...glass, padding: '24px 28px', textAlign: 'center' }}>
        <p style={{ fontSize: 16, fontWeight: 700, color: '#86efac', marginBottom: 8 }}>🎉 所有輪次抽獎完成</p>
      </div>
    )
  }

  if (!currentTier) return null

  const totalRounds = sorted.length

  return (
    <div data-testid="raffle-draw-session" style={{ ...glass, padding: '20px 24px' }}>
      <p data-testid="round-progress" style={{ fontSize: 11, color: 'rgba(148,210,255,.5)', marginBottom: 12 }}>
        第 {currentIndex + 1} / {totalRounds} 輪
      </p>

      <p data-testid="current-tier-name" style={{ fontSize: 20, fontWeight: 800, color: '#e0f2fe', marginBottom: 4 }}>
        {currentTier.name}
      </p>
      <p data-testid="current-tier-description" style={{ fontSize: 13, color: 'rgba(148,210,255,.6)', marginBottom: 16 }}>
        {currentTier.prize_description}
      </p>
      <p style={{ fontSize: 11, color: 'rgba(148,210,255,.4)', marginBottom: 16 }}>
        {currentTier.drawn_count} / {currentTier.winner_count} 人已抽出
      </p>

      {phase === 'round_ready' && (
        <button
          data-testid="draw-round-button"
          onClick={() => { void handleDraw() }}
          style={{ background: 'rgba(14,165,233,.2)', border: '1px solid rgba(14,165,233,.4)', color: '#38bdf8', borderRadius: 8, padding: '10px 28px', fontSize: 14, fontWeight: 700, cursor: 'pointer' }}
        >
          抽這一輪
        </button>
      )}

      {phase === 'drawing' && (
        <p data-testid="drawing-indicator" style={{ fontSize: 14, color: '#7dd3fc' }}>抽獎中...</p>
      )}

      {phase === 'round_result' && latestDraw && (
        <div data-testid="round-result">
          <p style={{ fontSize: 13, color: 'rgba(148,210,255,.5)', marginBottom: 6 }}>得獎者</p>
          <p data-testid="winner-name" style={{ fontSize: 24, fontWeight: 800, color: '#fde68a', marginBottom: 20 }}>
            {latestDraw.entry.display_name || latestDraw.entry.twitch_login}
          </p>
          <button
            data-testid="next-round-button"
            onClick={handleNext}
            style={{ background: 'rgba(34,197,94,.15)', border: '1px solid rgba(34,197,94,.3)', color: '#4ade80', borderRadius: 8, padding: '10px 24px', fontSize: 13, fontWeight: 700, cursor: 'pointer' }}
          >
            {currentIndex + 1 >= totalRounds ? '完成抽獎' : '繼續下一輪'}
          </button>
        </div>
      )}

      {error && <p style={{ fontSize: 12, color: '#f87171', marginTop: 10 }}>{error}</p>}
    </div>
  )
}
```

- [ ] **Step 4：確認測試通過**

```bash
cd apps/dashboard && npx vitest run src/components/raffle/__tests__/RaffleDrawSession.test.tsx
```

預期：PASS（2 tests）。

- [ ] **Step 5：Commit**

```bash
git add apps/dashboard/src/components/raffle/RaffleDrawSession.tsx \
        apps/dashboard/src/components/raffle/__tests__/RaffleDrawSession.test.tsx
git commit -m "feat: add RaffleDrawSession round_ready state

refs #1043"
```

---

## Task 3：測試抽獎流程與輪次推進

**Files:**
- Modify: `apps/dashboard/src/components/raffle/__tests__/RaffleDrawSession.test.tsx`

- [ ] **Step 1：補充測試 — 抽獎成功顯示得獎者，以及繼續下一輪**

在測試檔 `describe` 區塊內補充：

```tsx
import { fireEvent, waitFor } from '@testing-library/react'

// 在 describe 頂層加入 mock
vi.mock('@/services/raffles', () => ({
  drawFromTier: vi.fn(),
}))

import * as rafflesService from '@/services/raffles'

// 在 describe 內加入：
it('按下「抽這一輪」後顯示得獎者名稱', async () => {
  vi.mocked(rafflesService.drawFromTier).mockResolvedValueOnce({
    id: 'd1', raffle_id: 'r1', entry_id: 'e1',
    claim_token: '', claim_expires_at: '', drawn_at: '',
    entry: { id: 'e1', raffle_id: 'r1', twitch_login: 'viewer1', display_name: 'Viewer One', created_at: '' },
    prize_tier_id: 't1',
  } as any)

  render(<RaffleDrawSession raffleId="r1" tiers={mockTiers} />)
  fireEvent.click(screen.getByTestId('draw-round-button'))

  await waitFor(() => {
    expect(screen.getByTestId('winner-name')).toHaveTextContent('Viewer One')
  })
  expect(screen.getByTestId('next-round-button')).toHaveTextContent('繼續下一輪')
})

it('按下「繼續下一輪」後推進到第二輪', async () => {
  vi.mocked(rafflesService.drawFromTier).mockResolvedValueOnce({
    id: 'd1', raffle_id: 'r1', entry_id: 'e1',
    claim_token: '', claim_expires_at: '', drawn_at: '',
    entry: { id: 'e1', raffle_id: 'r1', twitch_login: 'viewer1', display_name: 'Viewer One', created_at: '' },
    prize_tier_id: 't1',
  } as any)

  render(<RaffleDrawSession raffleId="r1" tiers={mockTiers} />)
  fireEvent.click(screen.getByTestId('draw-round-button'))
  await waitFor(() => screen.getByTestId('next-round-button'))
  fireEvent.click(screen.getByTestId('next-round-button'))

  expect(screen.getByTestId('current-tier-name')).toHaveTextContent('二等獎')
  expect(screen.getByTestId('round-progress')).toHaveTextContent('第 2 / 2 輪')
})

it('最後一輪結束後顯示完成畫面', async () => {
  const singleTier: RafflePrizeTier[] = [
    { id: 't1', raffle_id: 'r1', name: '唯一獎', prize_description: '', winner_count: 1, drawn_count: 0, position: 0, created_at: '' },
  ]
  vi.mocked(rafflesService.drawFromTier).mockResolvedValueOnce({
    id: 'd1', raffle_id: 'r1', entry_id: 'e1',
    claim_token: '', claim_expires_at: '', drawn_at: '',
    entry: { id: 'e1', raffle_id: 'r1', twitch_login: 'winner', display_name: 'Winner', created_at: '' },
    prize_tier_id: 't1',
  } as any)

  render(<RaffleDrawSession raffleId="r1" tiers={singleTier} />)
  fireEvent.click(screen.getByTestId('draw-round-button'))
  await waitFor(() => screen.getByTestId('next-round-button'))
  fireEvent.click(screen.getByTestId('next-round-button'))

  expect(screen.getByTestId('session-complete')).toBeInTheDocument()
})

it('winner_count: 2 的輪次：第一次抽完按鈕顯示「繼續抽本輪」，第二次抽完才顯示完成', async () => {
  const twoWinnerTier: RafflePrizeTier[] = [
    { id: 't1', raffle_id: 'r1', name: '一等獎', prize_description: 'Switch', winner_count: 2, drawn_count: 0, position: 0, created_at: '' },
  ]
  const mockEntry = { id: 'e1', raffle_id: 'r1', twitch_login: 'viewer1', display_name: 'Viewer One', created_at: '' }
  vi.mocked(rafflesService.drawFromTier)
    .mockResolvedValueOnce({
      id: 'd1', raffle_id: 'r1', entry_id: 'e1',
      claim_token: '', claim_expires_at: '', drawn_at: '',
      entry: mockEntry,
      prize_tier_id: 't1',
    } as any)
    .mockResolvedValueOnce({
      id: 'd2', raffle_id: 'r1', entry_id: 'e2',
      claim_token: '', claim_expires_at: '', drawn_at: '',
      entry: { ...mockEntry, id: 'e2', twitch_login: 'viewer2', display_name: 'Viewer Two' },
      prize_tier_id: 't1',
    } as any)

  render(<RaffleDrawSession raffleId="r1" tiers={twoWinnerTier} />)

  // 第一次抽
  fireEvent.click(screen.getByTestId('draw-round-button'))
  await waitFor(() => expect(screen.getByTestId('next-round-button').textContent).toContain('繼續抽本輪'))

  // 點「繼續抽本輪」→ 回到 round_ready
  fireEvent.click(screen.getByTestId('next-round-button'))
  await waitFor(() => expect(screen.getByTestId('draw-round-button')).not.toBeNull())

  // 第二次抽
  fireEvent.click(screen.getByTestId('draw-round-button'))
  await waitFor(() => expect(screen.getByTestId('next-round-button').textContent).toContain('完成抽獎'))

  // 點「完成抽獎」→ session_complete
  fireEvent.click(screen.getByTestId('next-round-button'))
  expect(screen.getByTestId('session-complete')).not.toBeNull()
})

it('API 失敗時顯示錯誤訊息，回到 round_ready', async () => {
  vi.mocked(rafflesService.drawFromTier).mockRejectedValueOnce(new Error('network error'))

  render(<RaffleDrawSession raffleId="r1" tiers={mockTiers} />)
  fireEvent.click(screen.getByTestId('draw-round-button'))

  await waitFor(() => {
    expect(screen.getByTestId('draw-round-button')).toBeInTheDocument()
    expect(screen.getByText('抽獎失敗，請再試一次')).toBeInTheDocument()
  })
})
```

注意：需在測試檔最頂層（import 之後，describe 之前）加上：
```tsx
vi.mock('@/services/raffles', () => ({
  drawFromTier: vi.fn(),
}))
```
並在 `beforeEach` 中加：
```tsx
beforeEach(() => {
  vi.clearAllMocks()
})
```

- [ ] **Step 2：確認測試通過**

```bash
cd apps/dashboard && npx vitest run src/components/raffle/__tests__/RaffleDrawSession.test.tsx
```

預期：PASS（7 tests）。

- [ ] **Step 3：Commit**

```bash
git add apps/dashboard/src/components/raffle/__tests__/RaffleDrawSession.test.tsx
git commit -m "test: add RaffleDrawSession draw flow and round progression tests

refs #1043"
```

---

## Task 4：整合進 `RaffleDetailPage`

active 狀態且有 PrizeTier 時，以 `RaffleDrawSession` 取代現有「逐顆抽」按鈕區塊。

**Files:**
- Modify: `apps/dashboard/src/pages/RaffleDetailPage.tsx`
- Modify: `apps/dashboard/src/pages/__tests__/RaffleDetailPage.test.tsx`

- [ ] **Step 1：在 `RaffleDetailPage.tsx` import `RaffleDrawSession`**

在檔案頂部 import 區加入：

```tsx
import { RaffleDrawSession } from '@/components/raffle/RaffleDrawSession'
```

- [ ] **Step 2：找到 prize-tiers-section，加入條件渲染**

找到 `data-testid="prize-tiers-section"` 的 `<div>`（約第 1044 行），在其**前方**插入：

```tsx
{effectiveStatus === 'active' && tiers.length > 0 && (
  <div style={{ maxWidth: 1300, margin: '0 auto 2rem', padding: '0 16px' }}>
    <RaffleDrawSession raffleId={raffleId ?? ''} tiers={tiers} />
  </div>
)}
```

不修改或刪除現有的 prize-tiers-section（draft 狀態仍需要管理 UI）。

- [ ] **Step 3：將現有 prize-tiers-section 改為只在 draft 狀態顯示**

由於 Step 2 已讓 `RaffleDrawSession` 在 `active` 狀態顯示，原本只在 `active` 顯示的 prize-tiers-section（獎項層管理 UI）改為只在 `draft` 狀態顯示，避免兩個區塊同時出現。

找到 `data-testid="prize-tiers-section"` 外層 `<div>` 的條件渲染，將：

```tsx
{effectiveStatus === 'active' && (
```

改為：

```tsx
{effectiveStatus === 'draft' && (
```

- [ ] **Step 4：確認 TypeScript 無錯誤**

```bash
cd apps/dashboard && npx tsc --noEmit
```

預期：無錯誤。

- [ ] **Step 5：在 `RaffleDetailPage.test.tsx` 補整合測試**

在測試檔找到現有 import 的 mock 區塊，確認 `@/services/raffles` mock 中包含 `drawFromTier: vi.fn()`。若無，補上。

在測試檔末尾加入：

```tsx
describe('RaffleDrawSession 整合', () => {
  it('raffle active 且有 tiers 時顯示 RaffleDrawSession', async () => {
    vi.mocked(rafflesService.listPrizeTiers).mockResolvedValue([
      { id: 't1', raffle_id: 'r1', name: '一等獎', prize_description: 'Switch', winner_count: 1, drawn_count: 0, position: 0, created_at: '' },
    ])

    render(
      <RaffleDetailPage />,
      { wrapper: createRefineWrapper({ raffle: { ...mockRaffle, status: 'active' } }) }
    )

    await waitFor(() => {
      expect(screen.getByTestId('raffle-draw-session')).toBeInTheDocument()
    })
  })

  it('raffle draft 時不顯示 RaffleDrawSession', async () => {
    vi.mocked(rafflesService.listPrizeTiers).mockResolvedValue([
      { id: 't1', raffle_id: 'r1', name: '一等獎', prize_description: '', winner_count: 1, drawn_count: 0, position: 0, created_at: '' },
    ])

    render(
      <RaffleDetailPage />,
      { wrapper: createRefineWrapper({ raffle: { ...mockRaffle, status: 'draft' } }) }
    )

    await waitFor(() => {
      expect(screen.getByTestId('prize-tiers-section')).toBeInTheDocument()
    })
    expect(screen.queryByTestId('raffle-draw-session')).not.toBeInTheDocument()
  })
})
```

> 若測試檔的 `createRefineWrapper` 或 mock raffle shape 與上方不符，請依照測試檔現有 pattern 調整 — 重點是 `status: 'active'` vs `status: 'draft'` 的渲染差異。

- [ ] **Step 6：跑全部 dashboard 測試**

```bash
cd apps/dashboard && npx vitest run
```

預期：全部 PASS，無新 failure。

- [ ] **Step 7：Commit**

```bash
git add apps/dashboard/src/pages/RaffleDetailPage.tsx \
        apps/dashboard/src/pages/__tests__/RaffleDetailPage.test.tsx
git commit -m "feat: integrate RaffleDrawSession into RaffleDetailPage for active state

refs #1043"
```

---

## 驗證 Checklist

完成所有 Task 後，手動確認：

- [ ] 建立含 3 個 PrizeTier 的 Raffle（位置 0/1/2），啟動後進入逐輪模式
- [ ] 第一輪顯示 position=0 的獎項，按「抽這一輪」出現得獎者
- [ ] 按「繼續下一輪」，第二輪正確顯示 position=1 的獎項
- [ ] 所有輪次抽完後顯示「所有輪次抽獎完成」
- [ ] draft 狀態下仍能新增 / 刪除 PrizeTier（管理 UI 不受影響）
- [ ] 同一人不可能在兩輪都得獎（後端保證）
