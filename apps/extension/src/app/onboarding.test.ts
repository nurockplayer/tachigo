import assert from 'node:assert/strict'
import { test } from 'vitest'

import { markOnboardingComplete, shouldShowOnboarding } from './onboarding'
import type { NavigationFlags } from './navigation/types'

const freshFlags: NavigationFlags = {
  hasCompletedLogin: true,
  onboardingVersion: 0,
  selectedCharacterOnce: false,
}

test('shouldShowOnboarding only opens the tour for first-time mining viewers', () => {
  assert.equal(shouldShowOnboarding('hud', freshFlags), true)
  assert.equal(shouldShowOnboarding('login', freshFlags), false)
  assert.equal(shouldShowOnboarding('claim', freshFlags), false)
  assert.equal(
    shouldShowOnboarding('hud', {
      ...freshFlags,
      onboardingVersion: 1,
    }),
    false,
  )
})

test('markOnboardingComplete stores the current onboarding version without lowering newer versions', () => {
  assert.deepEqual(markOnboardingComplete(freshFlags), {
    ...freshFlags,
    onboardingVersion: 1,
  })
  assert.deepEqual(
    markOnboardingComplete({
      ...freshFlags,
      onboardingVersion: 3,
    }),
    {
      ...freshFlags,
      onboardingVersion: 3,
    },
  )
})
