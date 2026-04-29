import assert from 'node:assert/strict'
import test from 'node:test'

async function importTypesModule() {
  return import(`./types.ts?test=${Date.now()}-${Math.random()}`)
}

test('sanitizeDemoState returns fresh default objects instead of shared references', async () => {
  const types = await importTypesModule()

  const first = types.sanitizeDemoState(null)
  first.hud.points = 99
  first.redeemedCouponIds.push('coupon-1')

  const second = types.sanitizeDemoState(null)

  assert.notEqual(first, second)
  assert.notEqual(first.hud, second.hud)
  assert.deepEqual(second, {
    screen: 'login',
    language: 'en',
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
  const result = types.sanitizeDemoState({ screen: 'raffle' })
  assert.equal(result.screen, 'raffle')
})

test('sanitizeDemoState falls back to login for unknown screen', async () => {
  const types = await importTypesModule()
  const result = types.sanitizeDemoState({ screen: 'unknown_screen' })
  assert.equal(result.screen, 'login')
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
