import type { Dispatch, SetStateAction } from 'react'

import { LoadingScreen } from '../components/LoadingScreen'
import { LoginScreen } from '../components/LoginScreen'
import { MarioHUD } from '../components/MarioHUD'
import type { HudDemoState } from '../../extension/types'
import { PlaceholderSurface } from './PlaceholderSurface'
import { useNavigation } from './useNavigation'
import type { OverlayEntry } from './types'

interface SceneRendererProps {
  hudState: HudDemoState
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

export function SceneRenderer({ hudState, onHudStateChange }: SceneRendererProps) {
  const { state, goScene, pushOverlay, setFlag } = useNavigation()

  switch (state.scene) {
    case 'entry':
      return (
        <PlaceholderSurface
          eyebrow="Tachigo"
          title="Dive to Mine"
          body="Press anywhere to enter the extension flow."
          onClick={() => goScene('login')}
        />
      )
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
      return (
        <MarioHUD
          state={hudState}
          onStateChange={onHudStateChange}
          onNavigate={(screen) => pushOverlay(createHudOverlay(screen))}
        />
      )
  }
}
