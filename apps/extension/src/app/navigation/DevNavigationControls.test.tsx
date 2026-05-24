// @vitest-environment jsdom
import { useState } from 'react'
import { afterEach, expect, test } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'

import { DevNavigationControls } from './DevNavigationControls'
import { NavigationProvider } from './NavigationProvider'
import { useNavigation } from './useNavigation'
import type { AppLanguage } from '../../i18n'

function StateProbe() {
  const { state } = useNavigation()
  const topOverlay = state.overlayStack.at(-1)?.kind ?? 'none'

  return (
    <output aria-label="navigation state">
      {state.scene}:{topOverlay}
    </output>
  )
}

function Harness({ isDev }: { isDev: boolean }) {
  const [language, setLanguage] = useState<AppLanguage>('en')
  const [raffleId, setRaffleId] = useState('')

  return (
    <NavigationProvider>
      <DevNavigationControls
        isDev={isDev}
        isPopupMode={false}
        currentLanguage={language}
        raffleId={raffleId}
        onChangeLanguage={setLanguage}
        onChangeRaffleId={setRaffleId}
        onOpenPopupMode={() => undefined}
      />
      <StateProbe />
    </NavigationProvider>
  )
}

afterEach(() => {
  cleanup()
})

test('does not render demo navigation controls outside dev mode', () => {
  render(<Harness isDev={false} />)

  expect(screen.queryByRole('button', { name: 'LOGIN' })).toBeNull()
  screen.getByText('entry:none')
})

test('drives scene and overlay navigation in dev mode', () => {
  render(<Harness isDev />)

  fireEvent.click(screen.getByRole('button', { name: 'MINING' }))
  screen.getByText('mining:none')

  fireEvent.click(screen.getByRole('button', { name: 'SHOP' }))
  screen.getByText('mining:shop')

  fireEvent.click(screen.getByRole('button', { name: 'CLAIM' }))
  screen.getByText('mining:claim')

  fireEvent.click(screen.getByRole('button', { name: 'LOGIN' }))
  screen.getByText('login:none')
})
