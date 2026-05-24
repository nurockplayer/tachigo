import type { NavigationFlags } from './navigation/types'
import type { Scene } from './navigation/types'

export const ONBOARDING_VERSION = 1

export function shouldShowOnboarding(scene: Scene, flags: NavigationFlags): boolean {
  return scene === 'mining' && flags.onboardingVersion < ONBOARDING_VERSION
}

export function markOnboardingComplete(flags: NavigationFlags): NavigationFlags {
  return {
    ...flags,
    onboardingVersion: Math.max(flags.onboardingVersion, ONBOARDING_VERSION),
  }
}
