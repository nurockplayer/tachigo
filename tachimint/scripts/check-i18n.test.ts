import assert from 'node:assert/strict'
import test from 'node:test'

import { extractInterpolationTokens } from './check-i18n-helpers.ts'

test('extractInterpolationTokens deduplicates repeated tokens before sorting', () => {
  assert.deepEqual(
    extractInterpolationTokens('Count {{count}} / again {{ count }} / amount {{amount}}'),
    ['amount', 'count'],
  )
})

test('locale parity comparison fails fast when translated value is missing', () => {
  const localeStrings: Record<string, string> = {}

  assert.throws(() => {
    const translatedValue = localeStrings['coupon.cost']
    if (typeof translatedValue !== 'string') {
      assert.fail('zh-TW/common.json should provide a string value for coupon.cost')
    }
  }, /should provide a string value for coupon\.cost/)
})
