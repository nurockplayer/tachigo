import { fireEvent, render, screen } from '@testing-library/react'
import { vi } from 'vitest'

import '../i18n'
import type { DemoState } from '../extension/types'

const loadDemoStateMock = vi.fn<() => Promise<DemoState>>()
const saveDemoStateMock = vi.fn<(state: DemoState) => Promise<void>>()

vi.mock('../extension/storage', () => ({
  loadDemoState: () => loadDemoStateMock(),
  saveDemoState: (state: DemoState) => saveDemoStateMock(state),
}))

import App from './App'

const baseHud = {
  points: 0,
  totalPoints: 12847,
  countdown: 60,
  isWatching: true,
  clickCount: 0,
}

describe('App coupon shop flow', () => {
  beforeEach(() => {
    loadDemoStateMock.mockResolvedValue({
      screen: 'coupon',
      language: 'zh-TW',
      hud: baseHud,
      tcgBalance: 0,
      redeemedCouponIds: [],
    })
    saveDemoStateMock.mockResolvedValue(undefined)
  })

  it('shows insufficient balance before any claim', async () => {
    render(<App />)

    expect(await screen.findByText('Coupon 兌換商城')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '立即兌換' }))

    expect(screen.getByText('平台幣不足')).toBeInTheDocument()
  })

  it('deducts TCG when balance is sufficient', async () => {
    loadDemoStateMock.mockResolvedValue({
      screen: 'coupon',
      language: 'zh-TW',
      hud: baseHud,
      tcgBalance: 50,
      redeemedCouponIds: [],
    })

    render(<App />)

    expect(await screen.findByText('50.00')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '立即兌換' }))

    expect(screen.getByText('32.00')).toBeInTheDocument()
    expect(screen.getByText(/折扣碼 TACHIYA95 已入袋/)).toBeInTheDocument()
  })

  it('blocks duplicate redemption for the same coupon', async () => {
    loadDemoStateMock.mockResolvedValue({
      screen: 'coupon',
      language: 'zh-TW',
      hud: baseHud,
      tcgBalance: 50,
      redeemedCouponIds: [],
    })

    render(<App />)

    await screen.findByText('Coupon 兌換商城')

    fireEvent.click(screen.getByRole('button', { name: '立即兌換' }))
    expect(screen.getByText('32.00')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '立即兌換' }))
    expect(screen.getByText('此 Coupon 已兌換')).toBeInTheDocument()
  })
})
