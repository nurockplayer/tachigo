// @vitest-environment jsdom
import '../../i18n'

import { useState } from 'react'
import { afterEach, expect, test, vi } from 'vitest'
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'

import { NavigationProvider } from './NavigationProvider'
import { OverlayHost } from './OverlayHost'
import { useNavigation } from './useNavigation'
import { getCurrentAccount } from '../../services/api'
import type { SettingsState } from '../../extension/types'
import type { AppLanguage } from '../../i18n'

vi.mock('../../services/api', () => ({
  getCurrentAccount: vi.fn(),
}))

const getCurrentAccountMock = vi.mocked(getCurrentAccount)

const defaultSettings: SettingsState = {
  soundEnabled: true,
  effectsEnabled: true,
  hudVisible: true,
  screenMode: 'compact',
}

interface HarnessProps {
  currentLanguage?: AppLanguage
  onChangeLanguage?: (language: AppLanguage) => void
  settings?: SettingsState
  onSettingsChange?: (settings: SettingsState) => void
}

function Harness({
  currentLanguage = 'en',
  onChangeLanguage = () => undefined,
  settings = defaultSettings,
  onSettingsChange = () => undefined,
}: HarnessProps) {
  const { state, pushOverlay } = useNavigation()
  const [language, setLanguage] = useState<AppLanguage>(currentLanguage)
  const [currentSettings, setCurrentSettings] = useState(settings)
  const topOverlay = state.overlayStack.at(-1)?.kind ?? 'none'
  const handleLanguageChange = (nextLanguage: AppLanguage) => {
    setLanguage(nextLanguage)
    onChangeLanguage(nextLanguage)
  }
  const handleSettingsChange = (nextSettings: SettingsState) => {
    setCurrentSettings(nextSettings)
    onSettingsChange(nextSettings)
  }

  return (
    <>
      <button type="button" onClick={() => pushOverlay({ kind: 'menu' })}>
        open menu
      </button>
      <output aria-label="top overlay">{topOverlay}</output>
      <OverlayHost
        cpcBalance={0}
        currentLanguage={language}
        tcgBalance={0}
        redeemedCouponIds={[]}
        voucherCodes={{}}
        settings={currentSettings}
        onChangeLanguage={handleLanguageChange}
        onClaim={() => undefined}
        onCouponRedeem={async () => 'error'}
        onSettingsChange={handleSettingsChange}
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

test('settings panel toggles persisted sound and hud preferences', () => {
  const changes: SettingsState[] = []

  render(
    <NavigationProvider>
      <Harness
        settings={defaultSettings}
        onSettingsChange={(nextSettings) => {
          changes.push(nextSettings)
        }}
      />
    </NavigationProvider>,
  )

  fireEvent.click(screen.getByRole('button', { name: 'open menu' }))
  fireEvent.click(screen.getByRole('button', { name: 'Settings' }))

  expect(screen.getByRole('heading', { name: 'Settings' })).toBeTruthy()

  fireEvent.click(screen.getByRole('checkbox', { name: 'Sound' }))
  fireEvent.click(screen.getByRole('checkbox', { name: 'HUD' }))

  expect(changes).toEqual([
    {
      ...defaultSettings,
      soundEnabled: false,
    },
    {
      ...defaultSettings,
      soundEnabled: false,
      hudVisible: false,
    },
  ])
})

test('settings panel changes language through the persisted language path', () => {
  const languageChanges: AppLanguage[] = []

  render(
    <NavigationProvider>
      <Harness
        currentLanguage="en"
        onChangeLanguage={(nextLanguage) => {
          languageChanges.push(nextLanguage)
        }}
      />
    </NavigationProvider>,
  )

  fireEvent.click(screen.getByRole('button', { name: 'open menu' }))
  fireEvent.click(screen.getByRole('button', { name: 'Settings' }))
  fireEvent.click(screen.getByRole('radio', { name: '繁體中文' }))

  expect(languageChanges).toEqual(['zh-TW'])
})

test('settings panel switches screen mode', () => {
  const changes: SettingsState[] = []

  render(
    <NavigationProvider>
      <Harness
        settings={defaultSettings}
        onSettingsChange={(nextSettings) => {
          changes.push(nextSettings)
        }}
      />
    </NavigationProvider>,
  )

  fireEvent.click(screen.getByRole('button', { name: 'open menu' }))
  fireEvent.click(screen.getByRole('button', { name: 'Settings' }))
  fireEvent.click(screen.getByRole('radio', { name: 'Focus' }))

  expect(changes).toEqual([
    {
      ...defaultSettings,
      screenMode: 'focus',
    },
  ])
})
