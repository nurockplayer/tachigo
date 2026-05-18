import type { NavigationFlags } from './navigation/types'
import type { DemoScreen } from '../extension/types'

export const ONBOARDING_VERSION = 1

export function shouldShowOnboarding(screen: DemoScreen, flags: NavigationFlags): boolean {
  return screen === 'hud' && flags.onboardingVersion < ONBOARDING_VERSION
}

export function markOnboardingComplete(flags: NavigationFlags): NavigationFlags {
  return {
    ...flags,
    onboardingVersion: Math.max(flags.onboardingVersion, ONBOARDING_VERSION),
  }
}
