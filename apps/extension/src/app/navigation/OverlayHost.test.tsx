// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'

import '../../i18n'
import { NavigationProvider } from './NavigationProvider'
import { OverlayHost } from './OverlayHost'
import { useNavigation } from './useNavigation'
import { getCurrentAccount } from '../../services/api'

vi.mock('../../services/api', () => ({
  getCurrentAccount: vi.fn(),
}))

const getCurrentAccountMock = vi.mocked(getCurrentAccount)

function Harness() {
  const { state, pushOverlay } = useNavigation()
  const topOverlay = state.overlayStack.at(-1)?.kind ?? 'none'

  return (
    <>
      <button type="button" onClick={() => pushOverlay({ kind: 'menu' })}>
        open menu
      </button>
      <button type="button" onClick={() => pushOverlay({ kind: 'shop' })}>
        open shop
      </button>
      <output aria-label="top overlay">{topOverlay}</output>
      <OverlayHost
        cpcBalance={0}
        tcgBalance={0}
        redeemedCouponIds={[]}
        voucherCodes={{}}
        onClaim={() => undefined}
        onCouponRedeem={async () => 'error'}
      />
    </>
  )
}

afterEach(() => {
  cleanup()
  getCurrentAccountMock.mockReset()
})

test('menu hub exposes all MVP destination buttons and opens a panel', () => {
  render(
    <NavigationProvider>
      <Harness />
    </NavigationProvider>,
  )

  fireEvent.click(screen.getByRole('button', { name: 'open menu' }))

  for (const label of ['Account', 'Settings', 'Character', 'Collection', 'Missions', 'Equipment']) {
    expect(screen.getByRole('button', { name: label })).toBeTruthy()
  }

  fireEvent.click(screen.getByRole('button', { name: 'Settings' }))

  expect(screen.getByLabelText('top overlay').textContent).toBe('settings')
})

test('account overlay renders current account details and closes with back', async () => {
  getCurrentAccountMock.mockResolvedValue({
    id: 'user-1',
    username: 'mika',
    email: 'mika@example.com',
    role: 'streamer',
    isActive: true,
    emailVerified: false,
  })

  render(
    <NavigationProvider>
      <Harness />
    </NavigationProvider>,
  )

  fireEvent.click(screen.getByRole('button', { name: 'open menu' }))
  fireEvent.click(screen.getByRole('button', { name: 'Account' }))

  expect(screen.getAllByRole('status').some((node) => node.textContent?.includes('Loading account'))).toBe(true)
  expect(await screen.findByText('mika')).toBeTruthy()
  expect(screen.getByText('mika@example.com')).toBeTruthy()
  expect(screen.getByText('streamer')).toBeTruthy()
  expect(screen.getByText('Active')).toBeTruthy()
  expect(screen.getByText('Email not verified')).toBeTruthy()

  fireEvent.click(screen.getByRole('button', { name: 'Back' }))

  await waitFor(() => {
    expect(screen.getByLabelText('top overlay').textContent).toBe('menu')
  })
})

test('account overlay renders an error state when the current account fetch fails', async () => {
  getCurrentAccountMock.mockRejectedValue(new Error('network down'))

  render(
    <NavigationProvider>
      <Harness />
    </NavigationProvider>,
  )

  fireEvent.click(screen.getByRole('button', { name: 'open menu' }))
  fireEvent.click(screen.getByRole('button', { name: 'Account' }))

  expect((await screen.findByRole('alert')).textContent).toContain('Could not load account')
  expect(screen.getAllByRole('button', { name: 'Close' }).length).toBeGreaterThan(0)
})

test('shop opens category cards before entering the coupon market', () => {
  render(
    <NavigationProvider>
      <Harness />
    </NavigationProvider>,
  )

  fireEvent.click(screen.getByRole('button', { name: 'open shop' }))

  expect(screen.getByLabelText('top overlay').textContent).toBe('shop')
  expect(screen.getByRole('heading', { name: 'Choose a reward shelf' })).toBeTruthy()
  expect(screen.queryByRole('button', { name: 'REDEEM' })).toBeNull()

  fireEvent.click(screen.getByRole('button', { name: 'Tachiya Coupon' }))

  expect(screen.getByText('COUPON MARKET')).toBeTruthy()
  expect(screen.getByRole('button', { name: 'REDEEM' })).toBeTruthy()

  fireEvent.click(screen.getByRole('button', { name: '‹ BACK' }))

  expect(screen.getByLabelText('top overlay').textContent).toBe('none')
})
