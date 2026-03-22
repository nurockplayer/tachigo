import type { BitsTransaction } from '../types/twitch'

/**
 * Injects a fake window.Twitch.ext when running outside of a real Twitch iframe.
 * Only active in dev mode and only if Twitch.ext is not already present.
 */
export function injectTwitchExtMock() {
  if (window.Twitch?.ext) return

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
        onTransactionCancelled(_cb) {},
      },
    },
  }

  console.log('[Twitch.ext mock] injected')
}
