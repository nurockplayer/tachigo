// @vitest-environment jsdom
import { afterEach, beforeAll, expect, test, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'

import { defaultHudDemoState, defaultSettingsState } from '../../extension/types'
import '../../i18n'
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

vi.mock('../components/MarioHUD', () => ({
  MarioHUD: () => <div data-testid="mock-mario-hud">mock hud</div>,
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

function MiningProbe() {
  const { goScene } = useNavigation()

  return (
    <button type="button" onClick={() => goScene('mining')}>
      go mining
    </button>
  )
}

afterEach(() => {
  cleanup()
})

beforeAll(() => {
  window.localStorage.clear()
})

test('renders entry scene and enters login on press', () => {
  render(
    <NavigationProvider initialFlags={{ hasCompletedLogin: false }}>
      <SceneRenderer hudState={defaultHudDemoState} onHudStateChange={() => undefined} />
      <StateProbe />
    </NavigationProvider>,
  )

  expect(screen.getByRole('heading', { name: 'TACHIGO' })).toBeTruthy()
  expect(screen.getByText('PRESS ANYWHERE TO DIVE IN')).toBeTruthy()
  expect(screen.getByTestId('entry-motion-layer')).toBeTruthy()
  expect(screen.getAllByTestId('entry-motion-bubble')).toHaveLength(5)
  expect(screen.getByTestId('entry-current-ribbon')).toBeTruthy()

  fireEvent.click(screen.getByRole('button', { name: /PRESS ANYWHERE TO DIVE IN/i }))

  expect(screen.getByLabelText('navigation state').textContent).toBe('login:false')
  expect(screen.getByRole('button', { name: 'mock login' })).toBeTruthy()
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

test('hides the mining hud when hudVisible is disabled', () => {
  render(
    <NavigationProvider>
      <SceneRenderer
        hudState={defaultHudDemoState}
        settings={{ ...defaultSettingsState, hudVisible: false }}
        onHudStateChange={() => undefined}
      />
      <MiningProbe />
    </NavigationProvider>,
  )

  fireEvent.click(screen.getByRole('button', { name: 'go mining' }))

  expect(screen.queryByTestId('mock-mario-hud')).toBeNull()
  expect(screen.getByText('HUD Hidden')).toBeTruthy()
})
