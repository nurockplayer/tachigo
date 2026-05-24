import type { HudDemoState } from '../extension/types'

export const CHAT_SENT_MESSAGE_TYPE = 'CHAT_SENT'

interface ChatSentRuntimeMessage {
  type: typeof CHAT_SENT_MESSAGE_TYPE
}

function isChatSentRuntimeMessage(message: unknown): message is ChatSentRuntimeMessage {
  return (
    typeof message === 'object'
    && message !== null
    && (message as Partial<ChatSentRuntimeMessage>).type === CHAT_SENT_MESSAGE_TYPE
  )
}

export function applyChatMessageToHudState(
  previousHudState: HudDemoState,
  message: unknown,
): HudDemoState {
  if (!isChatSentRuntimeMessage(message)) {
    return previousHudState
  }

  return {
    ...previousHudState,
    chatCount: previousHudState.chatCount + 1,
  }
}
