import { useEffect, useState } from 'react'
import type { TwitchContext } from '../types/twitch'
import {
  clearAuthToken,
  loginWithTwitchExtension,
  setAuthToken,
  setExtensionJwtForRecovery,
} from '../services/api'
import i18n, { mapTwitchLocaleToAppLanguage } from '../i18n'

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
  const [backendReady, setBackendReady] = useState(false)

  useEffect(() => {
    const ext = window.Twitch?.ext
    if (!ext) return

    ext.onContext((ctx: TwitchExtContext) => {
      const rawLocale = ctx.locale ?? ctx.language ?? 'en'
      const appLang = mapTwitchLocaleToAppLanguage(rawLocale)
      if (i18n.language !== appLang && i18n.resolvedLanguage !== appLang) {
        void i18n.changeLanguage(appLang).catch((error: unknown) => {
          console.warn('Failed to change i18n language', error)
        })
      }

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
      setExtensionJwtForRecovery(auth.token)
      setBackendReady(false)

      // Login to tachigo backend with the extension JWT
      try {
        const result = await loginWithTwitchExtension(auth.token)
        // result.data.tokens.access_token
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const tokens = (result as any)?.data?.tokens ?? (result as any)?.tokens
        if (tokens?.access_token) {
          setAuthToken(tokens.access_token)
          setBackendReady(true)
          setAuthError(null)
        }
      } catch {
        // Non-fatal: bits flow still works via extension JWT directly
        clearAuthToken()
        setBackendReady(false)
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

    return () => {
      setExtensionJwtForRecovery(null)
      clearAuthToken()
    }
  }, [])

  useEffect(() => {
    if (!jwt || backendReady) {
      return
    }

    let cancelled = false
    const retryTimer = window.setInterval(() => {
      void (async () => {
        try {
          const result = await loginWithTwitchExtension(jwt)
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const tokens = (result as any)?.data?.tokens ?? (result as any)?.tokens
          if (!cancelled && tokens?.access_token) {
            setAuthToken(tokens.access_token)
            setBackendReady(true)
            setAuthError(null)
          }
        } catch {
          if (!cancelled) {
            setBackendReady(false)
          }
        }
      })()
    }, 15_000)

    return () => {
      cancelled = true
      window.clearInterval(retryTimer)
    }
  }, [backendReady, jwt])

  return { context, jwt, products, bitsEnabled, authError, backendReady }
}
