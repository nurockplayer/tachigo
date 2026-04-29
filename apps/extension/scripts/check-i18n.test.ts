import assert from 'node:assert/strict'
import test from 'node:test'

async function importCheckI18nModule() {
  return import(`./check-i18n.ts?test=${Date.now()}-${Math.random()}`)
}

test('extractInterpolationTokens returns sorted unique tokens', async () => {
  const { extractInterpolationTokens } = await importCheckI18nModule()

  assert.deepEqual(
    extractInterpolationTokens('{{amount}} {{ code }} {{amount}} {{count}}'),
    ['amount', 'code', 'count'],
  )
})

test('checkLocaleParity fails when a locale omits a required string value', async () => {
  const { checkLocaleParity } = await importCheckI18nModule()

  assert.throws(
    () =>
      checkLocaleParity({
        'en/common.json': {
          coupon: {
            cost: 'Cost {{amount}}',
          },
        },
        'zh-TW/common.json': {
          coupon: {},
        },
      }),
    /zh-TW\/common\.json should provide a string value for coupon\.cost/,
  )
})

test('checkLocaleParity fails on interpolation token mismatch in the real parity path', async () => {
  const { checkLocaleParity } = await importCheckI18nModule()

  assert.throws(
    () =>
      checkLocaleParity({
        'en/common.json': {
          coupon: {
            claimedCode: 'Code: {{code}} / {{amount}}',
          },
        },
        'zh-TW/common.json': {
          coupon: {
            claimedCode: '序號：{{code}}',
          },
        },
      }),
    /zh-TW\/common\.json has mismatched interpolation tokens for coupon\.claimedCode/,
  )
})
