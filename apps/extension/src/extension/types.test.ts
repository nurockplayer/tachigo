import assert from 'node:assert/strict'
import { test, vi } from 'vitest'

async function importTypesModule() {
  vi.resetModules()
  return import('./types.ts')
}

test('sanitizeDemoState returns fresh default objects instead of shared references', async () => {
  const types = await importTypesModule()

  const first = types.sanitizeDemoState(null)
  first.hud.points = 99
  first.redeemedCouponIds.push('coupon-1')
  first.flags.hasCompletedLogin = true

  const second = types.sanitizeDemoState(null)

  assert.notEqual(first, second)
  assert.notEqual(first.hud, second.hud)
  assert.notEqual(first.flags, second.flags)
  assert.deepEqual(second, {
    language: 'en',
    flags: {
      hasCompletedLogin: false,
      onboardingVersion: 0,
      selectedCharacterOnce: false,
    },
    hud: {
      points: 0,
      totalPoints: 12847,
      countdown: 60,
      isWatching: true,
      clickCount: 0,
    },
    tcgBalance: 0,
    redeemedCouponIds: [],
  })
})

test('sanitizeHudDemoState rejects non-finite numeric values', async () => {
  const types = await importTypesModule()

  assert.deepEqual(
    types.sanitizeHudDemoState({
      points: Number.NaN,
      totalPoints: Number.POSITIVE_INFINITY,
      countdown: Number.NEGATIVE_INFINITY,
      isWatching: false,
      clickCount: Number.NaN,
    }),
    {
      points: 0,
      totalPoints: 12847,
      countdown: 60,
      isWatching: false,
      clickCount: 0,
    },
  )
})

test('sanitizeDemoState accepts raffle as a valid screen', async () => {
  const types = await importTypesModule()
  const result = types.sanitizeDemoState({ flags: { hasCompletedLogin: true } })
  assert.equal(result.flags.hasCompletedLogin, true)
})

test('sanitizeDemoState ignores legacy screen state', async () => {
  const types = await importTypesModule()
  const result = types.sanitizeDemoState({ screen: 'raffle' })
  assert.equal('screen' in result, false)
})

test('sanitizeHudDemoState normalizes negative zero to positive zero', async () => {
  const types = await importTypesModule()

  const sanitized = types.sanitizeHudDemoState({
    points: -0,
    totalPoints: -0,
    countdown: -0,
    isWatching: true,
    clickCount: -0,
  })

  assert.equal(Object.is(sanitized.points, -0), false)
  assert.equal(Object.is(sanitized.totalPoints, -0), false)
  assert.equal(Object.is(sanitized.countdown, -0), false)
  assert.equal(Object.is(sanitized.clickCount, -0), false)
})
