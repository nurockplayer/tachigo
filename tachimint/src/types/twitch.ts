export interface TwitchContext {
  channelId: string
  clientId: string
  userId?: string
  opaqueUserId: string
  role: 'broadcaster' | 'moderator' | 'viewer' | 'external'
}

export interface BitsTransaction {
  transactionId: string
  product: {
    sku: string
    displayName: string
    cost: {
      amount: number
      type: 'bits'
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
