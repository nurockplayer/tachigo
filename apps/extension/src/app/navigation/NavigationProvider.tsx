import {
  createContext,
  useContext,
  useMemo,
  useReducer,
  type Dispatch,
  type ReactNode,
} from 'react'

import {
  defaultNavigationFlags,
  type NavigationFlags,
  type NavState,
  type Overlay,
  type OverlayEntry,
  type Scene,
} from './types'

export { defaultNavigationFlags }
export type { NavigationFlags, NavState, Overlay, OverlayEntry, Scene } from './types'

export type NavigationAction =
  | { type: 'goScene'; scene: Scene }
  | { type: 'pushOverlay'; entry: OverlayEntry }
  | { type: 'popOverlay' }
  | { type: 'closeAllOverlays' }
  | { type: 'setFlag'; key: keyof NavigationFlags; value: NavigationFlags[keyof NavigationFlags] }

export function createInitialNavState(flags: Partial<NavigationFlags> = {}): NavState {
  const mergedFlags = {
    ...defaultNavigationFlags,
    ...flags,
  }

  return {
    scene: mergedFlags.hasCompletedLogin ? 'loading' : 'entry',
    overlayStack: [],
    flags: mergedFlags,
  }
}

function areOverlayEntriesEqual(left: OverlayEntry, right: OverlayEntry) {
  if (left.kind !== right.kind) {
    return false
  }

  if (left.kind !== 'raffle-result' && right.kind !== 'raffle-result') {
    return true
  }

  if (left.kind === 'raffle-result' && right.kind === 'raffle-result') {
    return left.params.raffleId === right.params.raffleId
  }

  return false
}

export function navigationReducer(state: NavState, action: NavigationAction): NavState {
  switch (action.type) {
    case 'goScene':
      return {
        ...state,
        scene: action.scene,
        overlayStack: [],
      }
    case 'pushOverlay': {
      const topEntry = state.overlayStack.at(-1)

      if (topEntry && areOverlayEntriesEqual(topEntry, action.entry)) {
        return state
      }

      return {
        ...state,
        overlayStack: [...state.overlayStack, action.entry],
      }
    }
    case 'popOverlay':
      return {
        ...state,
        overlayStack: state.overlayStack.slice(0, -1),
      }
    case 'closeAllOverlays':
      return {
        ...state,
        overlayStack: [],
      }
    case 'setFlag':
      return {
        ...state,
        flags: {
          ...state.flags,
          [action.key]: action.value,
        },
      }
    default:
      return state
  }
}

interface NavigationContextValue {
  state: NavState
  dispatch: Dispatch<NavigationAction>
  goScene: (scene: Scene) => void
  pushOverlay: (entry: OverlayEntry) => void
  popOverlay: () => void
  closeAllOverlays: () => void
  setFlag: <K extends keyof NavigationFlags>(key: K, value: NavigationFlags[K]) => void
}

const NavigationContext = createContext<NavigationContextValue | null>(null)

interface NavigationProviderProps {
  children: ReactNode
  initialFlags?: Partial<NavigationFlags>
}

export function NavigationProvider({ children, initialFlags }: NavigationProviderProps) {
  const [state, dispatch] = useReducer(navigationReducer, createInitialNavState(initialFlags))

  const value = useMemo<NavigationContextValue>(
    () => ({
      state,
      dispatch,
      goScene: (scene) => dispatch({ type: 'goScene', scene }),
      pushOverlay: (entry) => dispatch({ type: 'pushOverlay', entry }),
      popOverlay: () => dispatch({ type: 'popOverlay' }),
      closeAllOverlays: () => dispatch({ type: 'closeAllOverlays' }),
      setFlag: (key, value) => dispatch({ type: 'setFlag', key, value }),
    }),
    [state],
  )

  return <NavigationContext.Provider value={value}>{children}</NavigationContext.Provider>
}

export function useNavigation() {
  const context = useContext(NavigationContext)

  if (!context) {
    throw new Error('useNavigation must be used within NavigationProvider')
  }

  return context
}
