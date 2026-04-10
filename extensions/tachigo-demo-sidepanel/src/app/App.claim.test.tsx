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

vi.mock('./components/ClaimPanel', () => ({
  ClaimPanel: ({
    cpcBalance,
    tcgBalance,
    onClaim,
  }: {
    cpcBalance: number
    tcgBalance: number
    onClaim: (amount: number) => void
  }) => (
    <div>
      <div data-testid="cpc-balance">{cpcBalance}</div>
      <div data-testid="tcg-balance">{tcgBalance}</div>
      <button onClick={() => onClaim(999)}>OVERCLAIM</button>
      <button onClick={() => onClaim(-5)}>NEGATIVE CLAIM</button>
      <button onClick={() => onClaim(10)}>VALID CLAIM</button>
    </div>
  ),
}))

import App from './App'

describe('App claim guards', () => {
  beforeEach(() => {
    loadDemoStateMock.mockResolvedValue({
      screen: 'claim',
      language: 'en',
      hud: {
        points: 50,
        totalPoints: 12847,
        countdown: 60,
        isWatching: true,
        clickCount: 0,
      },
    })
    saveDemoStateMock.mockResolvedValue(undefined)
  })

  it('caps claim rewards to the current CPC balance', async () => {
    render(<App />)

    expect(await screen.findByTestId('cpc-balance')).toHaveTextContent('50')
    expect(screen.getByTestId('tcg-balance')).toHaveTextContent('0')

    fireEvent.click(screen.getByRole('button', { name: 'OVERCLAIM' }))

    expect(screen.getByTestId('cpc-balance')).toHaveTextContent('0')
    expect(screen.getByTestId('tcg-balance')).toHaveTextContent('5')
  })

  it('ignores negative claim requests', async () => {
    render(<App />)

    expect(await screen.findByTestId('cpc-balance')).toHaveTextContent('50')
    expect(screen.getByTestId('tcg-balance')).toHaveTextContent('0')

    fireEvent.click(screen.getByRole('button', { name: 'NEGATIVE CLAIM' }))

    expect(screen.getByTestId('cpc-balance')).toHaveTextContent('50')
    expect(screen.getByTestId('tcg-balance')).toHaveTextContent('0')
  })
})
