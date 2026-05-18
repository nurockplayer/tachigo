import { createContext, type Dispatch } from 'react'

import type { NavigationAction } from './reducer'
import type { NavigationFlags, NavState, OverlayEntry, Scene } from './types'

export interface NavigationContextValue {
  state: NavState
  dispatch: Dispatch<NavigationAction>
  goScene: (scene: Scene) => void
  pushOverlay: (entry: OverlayEntry) => void
  popOverlay: () => void
  closeAllOverlays: () => void
  setFlag: <K extends keyof NavigationFlags>(key: K, value: NavigationFlags[K]) => void
}

export const NavigationContext = createContext<NavigationContextValue | null>(null)
