export type Scene = 'entry' | 'login' | 'loading' | 'character-select' | 'mining'

export type Overlay =
  | 'claim'
  | 'shop'
  | 'raffle-result'
  | 'menu'
  | 'account'
  | 'settings'
  | 'character-switch'
  | 'collection'
  | 'missions'
  | 'equipment'
  | 'onboarding'

export type OverlayEntry =
  | { kind: 'raffle-result'; params: { raffleId: string } }
  | { kind: Exclude<Overlay, 'raffle-result'>; params?: undefined }

export interface NavigationFlags {
  hasCompletedLogin: boolean
  onboardingVersion: number
  selectedCharacterOnce: boolean
}

export interface NavState {
  scene: Scene
  overlayStack: OverlayEntry[]
  flags: NavigationFlags
}

export const defaultNavigationFlags: NavigationFlags = {
  hasCompletedLogin: false,
  onboardingVersion: 0,
  selectedCharacterOnce: false,
}
