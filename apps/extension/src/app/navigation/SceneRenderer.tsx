import type { Dispatch, SetStateAction } from 'react'

import { LoadingScreen } from '../components/LoadingScreen'
import { LoginScreen } from '../components/LoginScreen'
import { MarioHUD } from '../components/MarioHUD'
import type { HudDemoState } from '../../extension/types'
import { PlaceholderSurface } from './PlaceholderSurface'
import { useNavigation } from './useNavigation'

interface SceneRendererProps {
  hudState: HudDemoState
  onHudStateChange: Dispatch<SetStateAction<HudDemoState>>
}

export function SceneRenderer({ hudState, onHudStateChange }: SceneRendererProps) {
  const { state, goScene, pushOverlay } = useNavigation()

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
      return <LoginScreen onLogin={() => goScene('loading')} />
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
          onNavigate={(screen) => pushOverlay({ kind: screen === 'coupon' ? 'shop' : 'claim' })}
        />
      )
  }
}
