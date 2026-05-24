// @vitest-environment jsdom
import { afterEach, expect, test, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'

import { defaultHudDemoState } from '../../extension/types'
import { NavigationProvider } from './NavigationProvider'
import { SceneRenderer } from './SceneRenderer'
import { useNavigation } from './useNavigation'

vi.mock('../components/LoginScreen', () => ({
  LoginScreen: ({ onLogin }: { onLogin: () => void }) => (
    <button type="button" onClick={onLogin}>
      mock login
    </button>
  ),
}))

function StateProbe() {
  const { state, goScene } = useNavigation()

  return (
    <>
      <button type="button" onClick={() => goScene('login')}>
        go login
      </button>
      <output aria-label="navigation state">
        {state.scene}:{String(state.flags.hasCompletedLogin)}
      </output>
    </>
  )
}

afterEach(() => {
  cleanup()
})

test('marks login as complete before entering loading scene', () => {
  render(
    <NavigationProvider initialFlags={{ hasCompletedLogin: false }}>
      <SceneRenderer hudState={defaultHudDemoState} onHudStateChange={() => undefined} />
      <StateProbe />
    </NavigationProvider>,
  )

  fireEvent.click(screen.getByRole('button', { name: 'go login' }))
  fireEvent.click(screen.getByRole('button', { name: 'mock login' }))

  expect(screen.getByLabelText('navigation state').textContent).toBe('loading:true')
})
