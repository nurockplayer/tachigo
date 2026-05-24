export const CHAT_SENT_MESSAGE_TYPE = 'CHAT_SENT'

export interface ChatSentMessage {
  type: typeof CHAT_SENT_MESSAGE_TYPE
}

interface RuntimeMessenger {
  sendMessage: (message: ChatSentMessage) => void | Promise<unknown>
}

interface EventSource {
  addEventListener: (
    type: string,
    listener: (event: Event) => void,
    options?: boolean | AddEventListenerOptions,
  ) => void
  removeEventListener: (
    type: string,
    listener: (event: Event) => void,
    options?: boolean | AddEventListenerOptions,
  ) => void
}

interface InstallOptions {
  document?: EventSource
  runtime?: RuntimeMessenger
}

const SEND_BUTTON_SELECTOR = 'button[data-a-target="chat-send-button"], [data-a-target="chat-send-button"]'
const CHAT_INPUT_SELECTOR =
  '[data-a-target="chat-input"], textarea[data-a-target="chat-input"], input[data-a-target="chat-input"], [contenteditable="true"][data-a-target="chat-input"]'

function closest(target: EventTarget | null, selector: string) {
  if (!target || typeof (target as Element).closest !== 'function') {
    return null
  }

  return (target as Element).closest(selector)
}

function postChatSent(runtime: RuntimeMessenger | undefined) {
  if (!runtime) {
    return
  }

  try {
    const result = runtime.sendMessage({ type: CHAT_SENT_MESSAGE_TYPE })
    if (result && typeof (result as Promise<unknown>).catch === 'function') {
      void (result as Promise<unknown>).catch(() => undefined)
    }
  } catch {
    // Content scripts should never break Twitch chat if extension messaging is unavailable.
  }
}

export function installTwitchChatDetector(options: InstallOptions = {}) {
  const eventSource = options.document ?? globalThis.document
  const runtime = options.runtime ?? globalThis.chrome?.runtime

  const handleClick = (event: Event) => {
    if (closest(event.target, SEND_BUTTON_SELECTOR)) {
      postChatSent(runtime)
    }
  }

  const handleKeydown = (event: Event) => {
    const keyboardEvent = event as KeyboardEvent
    if (
      keyboardEvent.key !== 'Enter' ||
      keyboardEvent.shiftKey ||
      keyboardEvent.isComposing ||
      keyboardEvent.repeat
    ) {
      return
    }

    if (closest(event.target, CHAT_INPUT_SELECTOR)) {
      postChatSent(runtime)
    }
  }

  eventSource.addEventListener('click', handleClick, true)
  eventSource.addEventListener('keydown', handleKeydown, true)

  return () => {
    eventSource.removeEventListener('click', handleClick, true)
    eventSource.removeEventListener('keydown', handleKeydown, true)
  }
}
