import { useEffect, useState } from 'react'
import type { TwitchContext } from '../types/twitch'
import { loginWithTwitchExtension, setAuthToken } from '../services/api'

export interface TwitchBitsProduct {
  sku: string
  displayName: string
  cost: { amount: number; type: 'bits' }
  inDevelopment: boolean
}

export function useTwitch() {
  const [context, setContext] = useState<TwitchContext | null>(null)
  const [jwt, setJwt] = useState<string>('')
  const [products, setProducts] = useState<TwitchBitsProduct[]>([])
  const [bitsEnabled, setBitsEnabled] = useState(false)
  const [authError, setAuthError] = useState<string | null>(null)

  useEffect(() => {
    const ext = window.Twitch?.ext
    if (!ext) return

    ext.onContext((ctx: TwitchExtContext) => {
      setContext({
        channelId: ctx.channelId,
        clientId: ctx.clientId,
        userId: ctx.userId,
        opaqueUserId: ctx.opaqueUserId,
        role: ctx.role,
      })
    })

    ext.onAuthorized(async (auth: TwitchExtAuth) => {
      setJwt(auth.token)

      // Login to tachigo backend with the extension JWT
      try {
        const result = await loginWithTwitchExtension(auth.token)
        // result.data.tokens.access_token
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const tokens = (result as any)?.data?.tokens ?? (result as any)?.tokens
        if (tokens?.access_token) {
          setAuthToken(tokens.access_token)
        }
      } catch {
        // Non-fatal: bits flow still works via extension JWT directly
        setAuthError('Backend unavailable')
      }

      // Fetch bits products
      if (ext.bits?.getProducts) {
        ext.bits.getProducts()
          .then((p) => {
            setProducts(p as TwitchBitsProduct[])
            setBitsEnabled(true)
          })
          .catch(() => setBitsEnabled(false))
      }
    })
  }, [])

  return { context, jwt, products, bitsEnabled, authError }
}
