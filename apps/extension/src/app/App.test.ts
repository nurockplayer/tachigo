import { expect, test } from 'vitest'

import { defaultHudDemoState } from '../extension/types'
import { claimFromHudState } from './claimState'

test('claimFromHudState clamps rapid duplicate claims to the latest CPC balance', () => {
  const first = claimFromHudState({ ...defaultHudDemoState, points: 10 }, 10)
  const second = claimFromHudState(first.nextHudState, 10)

  expect(first.claimable).toBe(10)
  expect(first.nextHudState.points).toBe(0)
  expect(second.claimable).toBe(0)
  expect(second.nextHudState.points).toBe(0)
})
