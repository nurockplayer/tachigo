// @vitest-environment jsdom
import { afterEach, expect, test } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'

import { NavigationProvider } from './NavigationProvider'
import { OverlayHost } from './OverlayHost'
import { useNavigation } from './useNavigation'

function Harness() {
  const { state, pushOverlay } = useNavigation()
  const topOverlay = state.overlayStack.at(-1)?.kind ?? 'none'

  return (
    <>
      <button type="button" onClick={() => pushOverlay({ kind: 'menu' })}>
        open menu
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
