import assert from 'node:assert/strict'
import { test } from 'vitest'

import {
  CHARACTER_DEFINITIONS,
  CHARACTER_IDS,
  calculateCharacterBuff,
  calculateFamiliarityMultiplier,
} from './characters'

test('exports the expected character ids in phase 1 order', () => {
  assert.deepEqual(CHARACTER_IDS, ['crab', 'dolphin', 'turtle', 'whale', 'capybara'])
})

test('defines unlock costs, stage thresholds, and dev-only flags', () => {
  assert.deepEqual(CHARACTER_DEFINITIONS.crab, {
    id: 'crab',
    displayName: 'Crab',
    unlockCost: 0,
    stages: [1, 2, 3],
    evolutionXpThresholds: { stage2: 1000, stage3: 10000 },
    devOnly: false,
  })

  assert.equal(CHARACTER_DEFINITIONS.dolphin.unlockCost, 50)
  assert.equal(CHARACTER_DEFINITIONS.turtle.unlockCost, 1500)
  assert.equal(CHARACTER_DEFINITIONS.whale.unlockCost, null)
  assert.equal(CHARACTER_DEFINITIONS.capybara.devOnly, true)
})

test('calculateFamiliarityMultiplier follows the phase 1 curve without decay for 30 days', () => {
  assert.equal(calculateFamiliarityMultiplier(0), 0.1)
  assert.equal(calculateFamiliarityMultiplier(60 * 60), 0.5)
  assert.equal(calculateFamiliarityMultiplier(3 * 60 * 60), 0.75)
  assert.equal(calculateFamiliarityMultiplier(5 * 60 * 60), 1)
  assert.equal(calculateFamiliarityMultiplier(5 * 60 * 60, 30), 1)
})

test('calculateFamiliarityMultiplier decays after 30 days and floors at 0.1', () => {
  assert.equal(calculateFamiliarityMultiplier(5 * 60 * 60, 31), 0.99)
  assert.equal(calculateFamiliarityMultiplier(0, 365), 0.1)
})

test('calculateCharacterBuff returns crab click bonus only after 10 effective clicks', () => {
  assert.equal(calculateCharacterBuff('crab', { effectiveClicksInWindow: 9 }), 1)
  assert.equal(calculateCharacterBuff('crab', { effectiveClicksInWindow: 10 }), 1.5)
  assert.equal(calculateCharacterBuff('crab', { effectiveClicksInWindow: 17 }), 1.5)
})

test('calculateCharacterBuff returns capped dolphin chat bonus tiers', () => {
  assert.equal(calculateCharacterBuff('dolphin', { chatCount: 0 }), 1)
  assert.equal(calculateCharacterBuff('dolphin', { chatCount: 1 }), 1.4)
  assert.equal(calculateCharacterBuff('dolphin', { chatCount: 2 }), 1.6)
  assert.equal(calculateCharacterBuff('dolphin', { chatCount: 3 }), 1.8)
  assert.equal(calculateCharacterBuff('dolphin', { chatCount: 999 }), 1.8)
})

test('calculateCharacterBuff returns turtle continuous watch bonus tiers', () => {
  assert.equal(calculateCharacterBuff('turtle', { continuousWatchSeconds: 29 * 60 }), 1)
  assert.equal(calculateCharacterBuff('turtle', { continuousWatchSeconds: 30 * 60 }), 1.1)
  assert.equal(calculateCharacterBuff('turtle', { continuousWatchSeconds: 60 * 60 }), 1.2)
  assert.equal(calculateCharacterBuff('turtle', { continuousWatchSeconds: 90 * 60 }), 1.35)
})

test('calculateCharacterBuff leaves whale and capybara at neutral phase 1 buffs', () => {
  assert.equal(calculateCharacterBuff('whale', {}), 1)
  assert.equal(calculateCharacterBuff('capybara', {}), 1)
})

test('calculateCharacterBuff preserves S1 bonuses for evolved stages until phase 2 hooks land', () => {
  assert.equal(calculateCharacterBuff('crab', { effectiveClicksInWindow: 20, stage: 2 }), 1.5)
  assert.equal(calculateCharacterBuff('dolphin', { chatCount: 5, stage: 3 }), 1.8)
  assert.equal(calculateCharacterBuff('turtle', { continuousWatchSeconds: 90 * 60, stage: 2 }), 1.35)
})
