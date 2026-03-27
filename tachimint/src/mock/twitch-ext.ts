import type { BitsTransaction } from '../types/twitch'

/**
 * Injects a fake window.Twitch.ext when running outside of a real Twitch iframe.
 * Only active in dev mode and only if Twitch.ext is not already present.
 */
export function injectTwitchExtMock() {
  // In real Twitch Extension environment, the app runs inside an iframe and the helper
  // script provides a working `window.Twitch.ext` that fires callbacks.
  // When running locally (top-level window), the helper script may still define
  // `window.Twitch.ext` but callbacks won't fire, causing the app to hang on "Connecting…".
  // In that case, we force-inject the mock to make localhost development usable.
  const isInIFrame = window.self !== window.top
  if (window.Twitch?.ext && isInIFrame) return
  let onTransactionComplete: ((tx: BitsTransaction) => void) | null = null

  window.Twitch = {
    ext: {
      onContext(cb) {
        setTimeout(() => cb({
          channelId: 'dev-channel-123',
          clientId: 'dev-client-id',
          opaqueUserId: 'U-dev-opaque',
          userId: 'dev-user-123',
          role: 'viewer',
        }), 0)
      },

      onAuthorized(cb) {
        setTimeout(() => cb({
          channelId: 'dev-channel-123',
          clientId: 'dev-client-id',
          token: 'mock.dev.jwt',
          userId: 'dev-user-123',
        }), 0)
      },

      bits: {
        getProducts: () => Promise.resolve([{
          sku: 'tachigo_100',
          displayName: 'tachigo 100 Bits',
          cost: { amount: 100, type: 'bits' as const },
          inDevelopment: true,
        }]),

        useBits(sku) {
          console.log('[Twitch.ext mock] useBits →', sku)
          // Simulate a completed transaction after 1s
          setTimeout(() => {
            onTransactionComplete?.({
              transactionId: `mock-tx-${Date.now()}`,
              transactionReceipt: 'mock-receipt',
              userId: 'dev-user-123',
              displayName: 'DevUser',
              initiator: 'current_user',
              product: {
                sku,
                displayName: 'tachigo 100 Bits',
                cost: { amount: 100, type: 'bits' },
                inDevelopment: true,
              },
            })
          }, 1000)
        },

        onTransactionComplete(cb) { onTransactionComplete = cb },
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        onTransactionCancelled(_cb) {},
      },
    },
  }

  console.log('[Twitch.ext mock] injected')
}
