export interface TwitchContext {
  channelId: string
  clientId: string
  userId?: string
  opaqueUserId: string
  role: 'broadcaster' | 'moderator' | 'viewer' | 'external'
}

// TPointTransaction is the payload received from Twitch when a viewer completes a T-point purchase.
// Note: the `type` field value is "bits" — this is a Twitch SDK contract value and must not be changed.
export interface TPointTransaction {
  transactionId: string
  product: {
    sku: string
    displayName: string
    cost: {
      amount: number
      type: 'bits' // Twitch SDK contract — do not rename
    }
    inDevelopment: boolean
  }
  userId: string
  displayName: string
  initiator: 'current_user' | 'other'
  transactionReceipt: string
}

export interface TachigoToken {
  accessToken: string
  refreshToken: string
  expiresIn: number
}
