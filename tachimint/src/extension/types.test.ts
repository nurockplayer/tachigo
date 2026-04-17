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

test('default demo state keeps hud defaults isolated from exported hud baseline', async () => {
  const types = await importTypesModule()

  assert.notEqual(types.defaultDemoState.hud, types.defaultHudDemoState)
})

test('sanitizeHudDemoState clamps negative numeric values to zero', async () => {
  const types = await importTypesModule()

  assert.deepEqual(
    types.sanitizeHudDemoState({
      points: -10,
      totalPoints: -5,
      countdown: -1,
      isWatching: true,
      clickCount: -99,
    }),
    {
      points: 0,
      totalPoints: 0,
      countdown: 0,
      isWatching: true,
      clickCount: 0,
    },
  )
})
