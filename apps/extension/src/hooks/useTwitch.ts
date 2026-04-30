import { useEffect, useState } from 'react'
import type { TwitchContext } from '../types/twitch'
import {
  clearAuthToken,
  loginWithTwitchExtension,
  setAuthToken,
  setExtensionJwtForRecovery,
} from '../services/api'
import i18n, { mapTwitchLocaleToAppLanguage } from '../i18n'

export interface TwitchTPointProduct {
  sku: string
  displayName: string
  cost: { amount: number; type: 'bits' }
  inDevelopment: boolean
}

export function useTwitch() {
  const [context, setContext] = useState<TwitchContext | null>(null)
  const [jwt, setJwt] = useState<string>('')
  const [products, setProducts] = useState<TwitchTPointProduct[]>([])
  const [tPointEnabled, setTPointEnabled] = useState(false)
  const [authError, setAuthError] = useState<string | null>(null)
  const [backendReady, setBackendReady] = useState(false)

  useEffect(() => {
    let mounted = true
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

      if (mounted) {
        setContext({
          channelId: ctx.channelId,
          clientId: ctx.clientId,
          userId: ctx.userId,
          opaqueUserId: ctx.opaqueUserId,
          role: ctx.role,
        })
      }
    })

    ext.onAuthorized(async (auth: TwitchExtAuth) => {
      if (mounted) setJwt(auth.token)
      setExtensionJwtForRecovery(auth.token)
      if (mounted) setBackendReady(false)

      // Login to tachigo backend with the extension JWT
      try {
        const result = await loginWithTwitchExtension(auth.token)
        // result.data.tokens.access_token
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const tokens = (result as any)?.data?.tokens ?? (result as any)?.tokens
        if (tokens?.access_token) {
          setAuthToken(tokens.access_token)
          if (mounted) {
            setBackendReady(true)
            setAuthError(null)
          }
        }
      } catch {
        // Non-fatal: t-point flow still works via extension JWT directly
        clearAuthToken()
        if (mounted) {
          setBackendReady(false)
          setAuthError('Backend unavailable')
        }
      }

      // Fetch T-point products
      if (ext.bits?.getProducts) {
        ext.bits.getProducts()
          .then((p) => {
            if (mounted) {
              setProducts(p as TwitchTPointProduct[])
              setTPointEnabled(true)
            }
          })
          .catch(() => {
            if (mounted) setTPointEnabled(false)
          })
      }
    })

    return () => {
      mounted = false
      setExtensionJwtForRecovery(null)
      clearAuthToken()
    }
  }, [])

  useEffect(() => {
    if (!jwt || backendReady) {
      return
    }

    let cancelled = false
    let inFlight = false
    const retryTimer = window.setInterval(() => {
      if (inFlight) return
      inFlight = true
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
        } finally {
          inFlight = false
        }
      })()
    }, 15_000)

    return () => {
      cancelled = true
      window.clearInterval(retryTimer)
    }
  }, [backendReady, jwt])

  return { context, jwt, products, tPointEnabled, authError, backendReady }
}
