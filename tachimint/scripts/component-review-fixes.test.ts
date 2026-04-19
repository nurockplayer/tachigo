import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'

import { parseCpcAmount } from '../src/app/components/claimAmount.ts'

function readComponentSource(name: string) {
  return readFileSync(new URL(`../src/app/components/${name}`, import.meta.url), 'utf8')
}

test('parseCpcAmount rejects strings with trailing non-numeric characters', () => {
  assert.equal(parseCpcAmount('10abc'), null)
})

test('parseCpcAmount accepts decimal CPC amounts', () => {
  assert.equal(parseCpcAmount('12.5'), 12.5)
})

test('ClickableCapybara uses a keyboard-accessible button', () => {
  const source = readComponentSource('MarioHUD.tsx')
  const clickableCapybara = source.slice(
    source.indexOf('function ClickableCapybara'),
    source.indexOf('// ─── Spinning Coin'),
  )

  assert.match(clickableCapybara, /<button\s+type="button"/)
  assert.match(clickableCapybara, /aria-label=/)
  assert.doesNotMatch(clickableCapybara, /<div\s+onClick=/)
})

test('MarioHUD uses defaultHudDemoState for total points fallback', () => {
  const source = readComponentSource('MarioHUD.tsx')

  assert.match(source, /defaultHudDemoState/)
  assert.match(source, /state\?\.totalPoints\s+\?\?\s+defaultHudDemoState\.totalPoints/)
  assert.doesNotMatch(source, /state\?\.totalPoints\s+\?\?\s+12847/)
})

test('CouponShopPanel uses the shared demo coupon catalog', () => {
  const source = readComponentSource('CouponShopPanel.tsx')

  assert.match(source, /demoCouponMetas/)
  assert.match(source, /DemoCouponMeta/)
  assert.doesNotMatch(source, /interface CouponMeta/)
  assert.doesNotMatch(source, /const COUPON_METAS:\s*CouponMeta\[\]/)
})
