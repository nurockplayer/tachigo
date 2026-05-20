import assert from 'node:assert/strict'
import { test } from 'vitest'

import {
  createInitialNavState,
  navigationReducer,
} from './reducer'
import { defaultNavigationFlags, type NavState } from './types'

function createState(overrides: Partial<NavState> = {}): NavState {
  return {
    scene: 'mining',
    overlayStack: [],
    flags: { ...defaultNavigationFlags },
    ...overrides,
  }
}

test('createInitialNavState starts first-time users at entry', () => {
  assert.deepEqual(createInitialNavState({ hasCompletedLogin: false }), {
    scene: 'entry',
    overlayStack: [],
    flags: {
      hasCompletedLogin: false,
      onboardingVersion: 0,
      selectedCharacterOnce: false,
    },
  })
})

test('createInitialNavState routes returning users to loading', () => {
  assert.equal(createInitialNavState({ hasCompletedLogin: true }).scene, 'loading')
})

test('goScene clears active overlays', () => {
  const state = createState({
    overlayStack: [
      { kind: 'shop' },
      { kind: 'raffle-result', params: { raffleId: 'raffle-a' } },
    ],
  })

  assert.deepEqual(navigationReducer(state, { type: 'goScene', scene: 'login' }), {
    ...state,
    scene: 'login',
    overlayStack: [],
  })
})

test('pushOverlay deduplicates only an identical top entry', () => {
  const state = navigationReducer(createState(), { type: 'pushOverlay', entry: { kind: 'menu' } })

  assert.deepEqual(navigationReducer(state, { type: 'pushOverlay', entry: { kind: 'menu' } }), state)
})

test('pushOverlay keeps same overlay kind when params differ', () => {
  const first = navigationReducer(createState(), {
    type: 'pushOverlay',
    entry: { kind: 'raffle-result', params: { raffleId: 'raffle-a' } },
  })

  assert.deepEqual(
    navigationReducer(first, {
      type: 'pushOverlay',
      entry: { kind: 'raffle-result', params: { raffleId: 'raffle-b' } },
    }).overlayStack,
    [
      { kind: 'raffle-result', params: { raffleId: 'raffle-a' } },
      { kind: 'raffle-result', params: { raffleId: 'raffle-b' } },
    ],
  )
})

test('setFlag supports clearing completed login after auth failure', () => {
  const state = createState({
    flags: {
      ...defaultNavigationFlags,
      hasCompletedLogin: true,
    },
  })

  assert.equal(
    navigationReducer(state, {
      type: 'setFlag',
      key: 'hasCompletedLogin',
      value: false,
    }).flags.hasCompletedLogin,
    false,
  )
})
