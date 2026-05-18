import { useMemo, useReducer, type ReactNode } from 'react'

import { NavigationContext, type NavigationContextValue } from './NavigationContext'
import { createInitialNavState, navigationReducer } from './reducer'
import type { NavigationFlags } from './types'

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
