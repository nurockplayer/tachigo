import type { Dispatch, SetStateAction } from 'react'

import { EntryScene } from '../components/EntryScene'
import { LoadingScreen } from '../components/LoadingScreen'
import { LoginScreen } from '../components/LoginScreen'
import { MarioHUD } from '../components/MarioHUD'
import { defaultSettingsState, type HudDemoState, type SettingsState } from '../../extension/types'
import { PlaceholderSurface } from './PlaceholderSurface'
import { useNavigation } from './useNavigation'
import type { OverlayEntry } from './types'

interface SceneRendererProps {
  hudState: HudDemoState
  settings?: SettingsState
  onHudStateChange: Dispatch<SetStateAction<HudDemoState>>
}

type HudNavigationTarget = NonNullable<Parameters<typeof MarioHUD>[0]['onNavigate']> extends (screen: infer Screen) => void
  ? Screen
  : never

function createHudOverlay(screen: HudNavigationTarget): OverlayEntry {
  switch (screen) {
    case 'claim':
      return { kind: 'claim' }
    case 'coupon':
      return { kind: 'shop' }
  }
}

export function SceneRenderer({ hudState, settings = defaultSettingsState, onHudStateChange }: SceneRendererProps) {
  const { state, goScene, pushOverlay, setFlag } = useNavigation()

  switch (state.scene) {
    case 'entry':
      return <EntryScene onEnter={() => goScene('login')} />
    case 'login':
      return (
        <LoginScreen
          onLogin={() => {
            setFlag('hasCompletedLogin', true)
            goScene('loading')
          }}
        />
      )
    case 'loading':
      return <LoadingScreen onComplete={() => goScene('mining')} />
    case 'character-select':
      return (
        <PlaceholderSurface
          eyebrow="Dev-only"
          title="Character Select"
          body="Reserved for the ocean character track."
        />
      )
    case 'mining':
      if (!settings.hudVisible) {
        return (
          <PlaceholderSurface
            eyebrow="Settings"
            title="HUD Hidden"
            body="The mining HUD is hidden by your display setting."
          />
        )
      }

      return (
        <MarioHUD
          state={hudState}
          effectsEnabled={settings.effectsEnabled}
          soundEnabled={settings.soundEnabled}
          onStateChange={onHudStateChange}
          onNavigate={(screen) => pushOverlay(createHudOverlay(screen))}
        />
      )
  }
}
