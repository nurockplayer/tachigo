// Type declarations for the Twitch Extension Helper (twitch-ext.min.js)
// Loaded via <script> in index.html

interface TwitchExtBits {
  getProducts(): Promise<TwitchBitsProduct[]>
  useBits(sku: string): void
  onTransactionComplete(callback: (transaction: import('./twitch').BitsTransaction) => void): void
  onTransactionCancelled(callback: () => void): void
}

interface TwitchBitsProduct {
  sku: string
  displayName: string
  cost: { amount: number; type: 'bits' }
  inDevelopment: boolean
}

interface TwitchExtAuth {
  channelId: string
  clientId: string
  token: string
  userId: string
}

interface TwitchExtContext {
  channelId: string
  clientId: string
  userId?: string
  opaqueUserId: string
  role: 'broadcaster' | 'moderator' | 'viewer' | 'external'
  [key: string]: unknown
}

interface TwitchExt {
  bits: TwitchExtBits
  onAuthorized(callback: (auth: TwitchExtAuth) => void): void
  onContext(callback: (context: TwitchExtContext) => void): void
}

interface Window {
  Twitch?: {
    ext: TwitchExt
  }
}
