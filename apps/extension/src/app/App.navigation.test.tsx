// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'

import App from './App'
import { createDefaultDemoState, type HudDemoState } from '../extension/types'

const storageMock = vi.hoisted(() => ({
  loadDemoState: vi.fn(),
  saveDemoState: vi.fn(),
}))

vi.mock('../extension/storage', () => storageMock)

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    i18n: {
      language: 'en',
      resolvedLanguage: 'en',
      changeLanguage: vi.fn().mockResolvedValue(undefined),
    },
  }),
}))

vi.mock('../hooks/useTwitch', () => ({
  useTwitch: () => ({ jwt: 'test-jwt' }),
}))

vi.mock('./components/LoadingScreen', () => ({
  LoadingScreen: ({ onComplete }: { onComplete: () => void }) => (
    <button type="button" onClick={onComplete}>
      finish loading
    </button>
  ),
}))

vi.mock('./components/MarioHUD', () => ({
  MarioHUD: ({ state, onNavigate }: { state: HudDemoState; onNavigate?: (screen: 'claim' | 'coupon') => void }) => (
    <div>
      <output aria-label="hud points">{state.points}</output>
      <button type="button" onClick={() => onNavigate?.('claim')}>
        open claim
      </button>
    </div>
  ),
}))

vi.mock('./components/ClaimPanel', () => ({
  ClaimPanel: ({
    cpcBalance,
    tcgBalance,
    onClaim,
  }: {
    cpcBalance: number
    tcgBalance: number
    onClaim: (cpcAmount: number) => void
  }) => (
    <div>
      <output aria-label="claim balances">
        {cpcBalance}:{tcgBalance}
      </output>
      <button
        type="button"
        onClick={() => {
          onClaim(10)
          onClaim(10)
        }}
      >
        claim twice
      </button>
    </div>
  ),
}))

afterEach(() => {
  cleanup()
  storageMock.loadDemoState.mockReset()
  storageMock.saveDemoState.mockReset()
})

test('claim flow clamps rapid duplicate claims to the latest CPC balance', async () => {
  const state = createDefaultDemoState()
  state.flags = {
    hasCompletedLogin: true,
    onboardingVersion: 1,
    selectedCharacterOnce: false,
  }
  state.hud = {
    points: 10,
    totalPoints: 10,
    countdown: 60,
    isWatching: true,
    clickCount: 0,
  }
  state.tcgBalance = 0
  storageMock.loadDemoState.mockResolvedValue(state)
  storageMock.saveDemoState.mockResolvedValue(undefined)

  render(<App />)

  fireEvent.click(await screen.findByRole('button', { name: 'finish loading' }))
  fireEvent.click(screen.getByRole('button', { name: 'open claim' }))
  screen.getByText('10:0')

  fireEvent.click(screen.getByRole('button', { name: 'claim twice' }))

  await waitFor(() => {
    expect(screen.getByLabelText('claim balances').textContent).toBe('0:1')
  })
})
