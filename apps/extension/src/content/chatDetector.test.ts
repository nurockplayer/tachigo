import assert from 'node:assert/strict'
import { afterEach, test } from 'vitest'

import { CHAT_SENT_MESSAGE_TYPE, installTwitchChatDetector } from './chatDetector'

class FakeElement {
  readonly children: FakeElement[] = []

  constructor(
    readonly tagName: string,
    private readonly attributes: Record<string, string> = {},
    private readonly parent: FakeElement | null = null,
  ) {}

  appendChild(child: FakeElement) {
    this.children.push(child)
    return child
  }

  closest(selector: string) {
    if (this.matches(selector)) {
      return this
    }
    return this.parent?.closest(selector) ?? null
  }

  matches(selector: string) {
    return selector
      .split(',')
      .map((candidate) => candidate.trim())
      .some((candidate) => {
        if (candidate === 'button[data-a-target="chat-send-button"]') {
          return this.tagName === 'BUTTON' && this.attributes['data-a-target'] === 'chat-send-button'
        }
        if (candidate === '[data-a-target="chat-send-button"]') {
          return this.attributes['data-a-target'] === 'chat-send-button'
        }
        if (candidate === '[data-a-target="chat-input"]') {
          return this.attributes['data-a-target'] === 'chat-input'
        }
        if (candidate === 'textarea[data-a-target="chat-input"]') {
          return this.tagName === 'TEXTAREA' && this.attributes['data-a-target'] === 'chat-input'
        }
        if (candidate === 'input[data-a-target="chat-input"]') {
          return this.tagName === 'INPUT' && this.attributes['data-a-target'] === 'chat-input'
        }
        if (candidate === '[contenteditable="true"][data-a-target="chat-input"]') {
          return this.attributes.contenteditable === 'true' && this.attributes['data-a-target'] === 'chat-input'
        }
        return false
      })
  }
}

class FakeDocument {
  private readonly listeners = new Map<string, Array<(event: Event) => void>>()

  addEventListener(type: string, listener: (event: Event) => void) {
    this.listeners.set(type, [...(this.listeners.get(type) ?? []), listener])
  }

  removeEventListener(type: string, listener: (event: Event) => void) {
    this.listeners.set(
      type,
      (this.listeners.get(type) ?? []).filter((candidate) => candidate !== listener),
    )
  }

  dispatch(type: string, event: Event) {
    for (const listener of this.listeners.get(type) ?? []) {
      listener(event)
    }
  }
}

function createKeyboardEvent(
  target: FakeElement,
  key: string,
  options: { isComposing?: boolean; repeat?: boolean; shiftKey?: boolean } = {},
): Event {
  return { key, shiftKey: false, target, ...options } as unknown as Event
}

function createEvent(target: FakeElement): Event {
  return { target } as unknown as Event
}

afterEach(() => {
  delete (globalThis as { chrome?: unknown }).chrome
})

test('installTwitchChatDetector posts CHAT_SENT when the Twitch send button is clicked', () => {
  const document = new FakeDocument()
  const sentMessages: unknown[] = []
  const sendButton = new FakeElement('BUTTON', { 'data-a-target': 'chat-send-button' })

  const stop = installTwitchChatDetector({
    document,
    runtime: {
      sendMessage(message: unknown) {
        sentMessages.push(message)
      },
    },
  })

  document.dispatch('click', createEvent(sendButton))
  stop()
  document.dispatch('click', createEvent(sendButton))

  assert.deepEqual(sentMessages, [{ type: CHAT_SENT_MESSAGE_TYPE }])
})

test('installTwitchChatDetector posts CHAT_SENT for Enter in chat input but ignores Shift+Enter', () => {
  const document = new FakeDocument()
  const sentMessages: unknown[] = []
  const chatInput = new FakeElement('DIV', {
    'data-a-target': 'chat-input',
    contenteditable: 'true',
  })

  installTwitchChatDetector({
    document,
    runtime: {
      sendMessage(message: unknown) {
        sentMessages.push(message)
      },
    },
  })

  document.dispatch('keydown', createKeyboardEvent(chatInput, 'Enter', { shiftKey: true }))
  document.dispatch('keydown', createKeyboardEvent(chatInput, 'Enter'))

  assert.deepEqual(sentMessages, [{ type: CHAT_SENT_MESSAGE_TYPE }])
})

test('installTwitchChatDetector ignores IME composition and repeated Enter keydown events', () => {
  const document = new FakeDocument()
  const sentMessages: unknown[] = []
  const chatInput = new FakeElement('DIV', {
    'data-a-target': 'chat-input',
    contenteditable: 'true',
  })

  installTwitchChatDetector({
    document,
    runtime: {
      sendMessage(message: unknown) {
        sentMessages.push(message)
      },
    },
  })

  document.dispatch('keydown', createKeyboardEvent(chatInput, 'Enter', { isComposing: true }))
  document.dispatch('keydown', createKeyboardEvent(chatInput, 'Enter', { repeat: true }))
  document.dispatch('keydown', createKeyboardEvent(chatInput, 'Enter'))

  assert.deepEqual(sentMessages, [{ type: CHAT_SENT_MESSAGE_TYPE }])
})

test('installTwitchChatDetector ignores unrelated selectors without throwing', () => {
  const document = new FakeDocument()
  const sentMessages: unknown[] = []
  const unrelated = new FakeElement('DIV')

  installTwitchChatDetector({
    document,
    runtime: {
      sendMessage(message: unknown) {
        sentMessages.push(message)
      },
    },
  })

  document.dispatch('click', createEvent(unrelated))
  document.dispatch('keydown', createKeyboardEvent(unrelated, 'Enter'))

  assert.deepEqual(sentMessages, [])
})
