import {
  defaultNavigationFlags,
  type NavigationFlags,
  type NavState,
  type OverlayEntry,
  type Scene,
} from './types'

type SetNavigationFlagAction = {
  [K in keyof NavigationFlags]: { type: 'setFlag'; key: K; value: NavigationFlags[K] }
}[keyof NavigationFlags]

export type NavigationAction =
  | { type: 'goScene'; scene: Scene }
  | { type: 'pushOverlay'; entry: OverlayEntry }
  | { type: 'popOverlay' }
  | { type: 'closeAllOverlays' }
  | SetNavigationFlagAction

export function createSetNavigationFlagAction<K extends keyof NavigationFlags>(
  key: K,
  value: NavigationFlags[K],
): SetNavigationFlagAction {
  return { type: 'setFlag', key, value } as SetNavigationFlagAction
}

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
  }
}
