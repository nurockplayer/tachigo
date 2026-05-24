import assert from 'node:assert/strict'
import { test } from 'vitest'

import { defaultHudDemoState } from '../extension/types'
import { CHAT_SENT_MESSAGE_TYPE, applyChatMessageToHudState } from './chatMessages'

test('applyChatMessageToHudState increments chatCount for CHAT_SENT messages', () => {
  assert.deepEqual(
    applyChatMessageToHudState(
      { ...defaultHudDemoState, chatCount: 1 },
      { type: CHAT_SENT_MESSAGE_TYPE },
    ),
    { ...defaultHudDemoState, chatCount: 2 },
  )
})

test('applyChatMessageToHudState leaves hud state unchanged for unrelated runtime messages', () => {
  const previous = { ...defaultHudDemoState, chatCount: 3 }

  assert.equal(applyChatMessageToHudState(previous, { type: 'PLAY_SOUND' }), previous)
  assert.equal(applyChatMessageToHudState(previous, null), previous)
})
