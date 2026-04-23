import { useEffect, useRef, useState, type CSSProperties } from 'react';
import { useTranslation } from 'react-i18next'

import { loadDemoState, saveDemoState } from '../extension/storage';
import {
  defaultDemoState,
  normalizeAppLanguage,
  type CouponRedeemResult,
  type DemoScreen,
  type HudDemoState,
} from '../extension/types';
import type { AppLanguage } from '../i18n';
import { LoadingScreen } from './components/LoadingScreen';
import { LoginScreen } from './components/LoginScreen';
import { LanguageSwitcher } from './components/LanguageSwitcher';
import { MarioHUD } from './components/MarioHUD';
import { ClaimPanel } from './components/ClaimPanel';
import { CouponShopPanel } from './components/CouponShopPanel';
import { useTwitch } from '../hooks/useTwitch';
import { redeemCoupon } from '../services/api';

type CouponRedeemOutcome = CouponRedeemResult | 'error'

function isInsufficientFundsError(error: unknown) {
  return error instanceof Error && /insufficient|balance|402/i.test(error.message)
}

export default function App() {
  const { i18n } = useTranslation()
  const { jwt } = useTwitch()
  const isPopupMode = typeof window !== 'undefined' && window.location.pathname.endsWith('/popup.html')
  const [currentLanguage, setCurrentLanguage] = useState<AppLanguage>(defaultDemoState.language);
  const useZpixLanguage = currentLanguage === 'zh-TW' || currentLanguage === 'zh-CN'
  const fontVariables = {
    '--ui-font-family': useZpixLanguage
      ? "'Press Start 2P', 'Zpix CJK', 'Inter', system-ui, sans-serif"
      : "'Inter', system-ui, sans-serif",
    '--pixel-font-family': useZpixLanguage
      ? "'Press Start 2P', 'Zpix CJK', monospace"
      : "'Press Start 2P', monospace",
  } as CSSProperties
  const [isHydrated, setIsHydrated] = useState(false);
  const [screen, setScreen] = useState<DemoScreen>(defaultDemoState.screen);
  const [hudState, setHudState] = useState<HudDemoState>(defaultDemoState.hud);
  const [tcgBalance, setTcgBalance] = useState(defaultDemoState.tcgBalance);
  const [redeemedCouponIds, setRedeemedCouponIds] = useState<string[]>(defaultDemoState.redeemedCouponIds);
  const [voucherCodes, setVoucherCodes] = useState<Record<string, string>>({});
  const tcgBalanceRef = useRef(defaultDemoState.tcgBalance);
  const redeemedCouponIdsRef = useRef<string[]>([...defaultDemoState.redeemedCouponIds]);

  useEffect(() => {
    let isActive = true

    void loadDemoState()
      .then(async (storedState) => {
        if (!isActive) {
          return
        }

        setScreen(storedState.screen)
        setHudState(storedState.hud)
        setTcgBalance(storedState.tcgBalance)
        tcgBalanceRef.current = storedState.tcgBalance
        setRedeemedCouponIds(storedState.redeemedCouponIds)
        redeemedCouponIdsRef.current = [...storedState.redeemedCouponIds]

        const targetLanguage = normalizeAppLanguage(storedState.language)
        setCurrentLanguage(targetLanguage)
        if (i18n.language !== targetLanguage || i18n.resolvedLanguage !== targetLanguage) {
          await i18n.changeLanguage(targetLanguage).catch((error: unknown) => {
            console.warn('Failed to hydrate i18n language from storage', error)
          })
        }

        if (isActive) {
          setIsHydrated(true)
        }
      })
      .catch((error: unknown) => {
        console.warn('Failed to load extension demo state', error)
        if (isActive) {
          setIsHydrated(true)
        }
      })

    return () => {
      isActive = false
    }
  }, [i18n])

  useEffect(() => {
    if (!isHydrated) {
      return
    }

    const persistTimer = window.setTimeout(() => {
      void saveDemoState({
        screen,
        language: currentLanguage,
        hud: hudState,
        tcgBalance,
        redeemedCouponIds,
      }).catch((error: unknown) => {
        console.warn('Failed to persist extension demo state', error)
      })
    }, 120)

    return () => window.clearTimeout(persistTimer)
  }, [currentLanguage, hudState, isHydrated, redeemedCouponIds, screen, tcgBalance])

  const handleClaim = (cpcAmount: number) => {
    const claimable = Math.max(0, Math.min(cpcAmount, hudState.points))
    if (claimable === 0) {
      return
    }

    const tcgGained = Number((claimable * 0.1).toFixed(2))
    setHudState((s) => ({ ...s, points: s.points - claimable }))
    setTcgBalance((t) => {
      const next = Number((t + tcgGained).toFixed(2))
      tcgBalanceRef.current = next
      return next
    })
  }

  const handleCouponRedeem = async (couponId: string, cost: number): Promise<CouponRedeemOutcome> => {
    if (!Number.isFinite(cost) || cost <= 0) {
      return 'insufficient'
    }

    if (redeemedCouponIdsRef.current.includes(couponId)) {
      return 'already_redeemed'
    }

    if (!jwt) {
      return 'error'
    }

    try {
      const result = await redeemCoupon(couponId, cost, jwt)
      tcgBalanceRef.current = result.balance
      setTcgBalance(result.balance)
      setVoucherCodes((currentCodes) => ({
        ...currentCodes,
        [couponId]: result.voucher_code,
      }))

      const nextRedeemed = [...redeemedCouponIdsRef.current, couponId]
      redeemedCouponIdsRef.current = nextRedeemed
      setRedeemedCouponIds(nextRedeemed)

      return 'success'
    } catch (error) {
      return isInsufficientFundsError(error) ? 'insufficient' : 'error'
    }
  }

  const openPopupMode = () => {
    const popupUrl = globalThis.chrome?.runtime?.getURL('popup.html') ?? `${window.location.origin}/popup.html`
    window.open(popupUrl, 'tachigo-demo-popup', 'popup=yes,width=430,height=820,resizable=yes')
  }

  const handleLanguageChange = (language: AppLanguage) => {
    setCurrentLanguage(language)
    void i18n.changeLanguage(language).catch((error: unknown) => {
      console.warn('Failed to switch panel language', error)
    })
    void saveDemoState({
      screen,
      language,
      hud: hudState,
      tcgBalance,
      redeemedCouponIds,
    }).catch((error: unknown) => {
      console.warn('Failed to persist language switch', error)
    })
  }

  if (!isHydrated) {
    return (
      <div
        style={{
          minHeight: '100vh',
          background: '#06060f',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: 24,
          fontFamily: 'var(--pixel-font-family)',
          ...fontVariables,
        }}
      >
        <div
          style={{
            width: 320,
            height: 600,
            borderRadius: 12,
            overflow: 'hidden',
            border: '1px solid rgba(255,255,255,0.07)',
            boxShadow: '0 0 0 1px rgba(0,0,0,0.9), 0 8px 48px rgba(0,0,0,0.9)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: 'rgba(145,70,255,0.9)',
            letterSpacing: '0.08em',
            fontSize: 10,
          }}
        >
          PREPARING PANEL...
        </div>
      </div>
    )
  }

  return (
    <div
      style={{
        minHeight: '100vh',
        background: '#06060f',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'flex-start',
        gap: 12,
        padding: '16px 16px 18px',
        boxSizing: 'border-box',
        fontFamily: 'var(--ui-font-family)',
        ...fontVariables,
      }}
    >
      {/* Extension frame */}
      <div
        style={{
          width: 320,
          height: 600,
          borderRadius: 12,
          overflow: 'hidden',
          border: '1px solid rgba(255,255,255,0.07)',
          boxShadow:
            '0 0 0 1px rgba(0,0,0,0.9), 0 8px 48px rgba(0,0,0,0.9)',
          flexShrink: 0,
          position: 'relative',
        }}
      >
        {screen === 'login' ? (
          <LoginScreen onLogin={() => setScreen('loading')} />
        ) : screen === 'loading' ? (
          <LoadingScreen onComplete={() => setScreen('hud')} />
        ) : screen === 'claim' ? (
          <ClaimPanel
              onBack={() => setScreen('hud')}
              cpcBalance={hudState.points}
              tcgBalance={tcgBalance}
              onClaim={handleClaim}
            />
        ) : screen === 'coupon' ? (
          <CouponShopPanel
            onBack={() => setScreen('hud')}
            tcgBalance={tcgBalance}
            redeemedCouponIds={redeemedCouponIds}
            voucherCodes={voucherCodes}
            onRedeem={handleCouponRedeem}
          />
        ) : (
          <MarioHUD state={hudState} onStateChange={setHudState} onNavigate={(s) => setScreen(s)} />
        )}
      </div>

      {/* Demo controls */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 8,
          width: 320,
          maxWidth: '100%',
          position: 'relative',
          zIndex: 2,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8, flexWrap: 'wrap' }}>
          <button
            onClick={() => setScreen('login')}
            style={{
              padding: '4px 12px',
              borderRadius: 4,
              border: '1px solid rgba(255,255,255,0.1)',
              background: screen === 'login' ? 'rgba(200,168,73,0.15)' : 'transparent',
              color: screen === 'login' ? 'rgba(200,168,73,0.8)' : 'rgba(100,100,140,0.4)',
              fontSize: 9,
              fontFamily: 'var(--pixel-font-family)',
              cursor: 'pointer',
              letterSpacing: '0.08em',
            }}
          >
            LOGIN
          </button>
          <span style={{ fontSize: 10, color: 'rgba(100,100,140,0.3)', fontFamily: 'var(--pixel-font-family)' }}>·</span>
          <button
            onClick={() => setScreen('loading')}
            style={{
              padding: '4px 12px',
              borderRadius: 4,
              border: '1px solid rgba(255,255,255,0.1)',
              background: screen === 'loading' ? 'rgba(200,168,73,0.15)' : 'transparent',
              color: screen === 'loading' ? 'rgba(200,168,73,0.8)' : 'rgba(100,100,140,0.4)',
              fontSize: 9,
              fontFamily: 'var(--pixel-font-family)',
              cursor: 'pointer',
              letterSpacing: '0.08em',
            }}
          >
            LOAD
          </button>
          <span style={{ fontSize: 10, color: 'rgba(100,100,140,0.3)', fontFamily: 'var(--pixel-font-family)' }}>·</span>
          <button
            onClick={() => setScreen('hud')}
            style={{
              padding: '4px 12px',
              borderRadius: 4,
              border: '1px solid rgba(255,255,255,0.1)',
              background: screen === 'hud' ? 'rgba(200,168,73,0.15)' : 'transparent',
              color: screen === 'hud' ? 'rgba(200,168,73,0.8)' : 'rgba(100,100,140,0.4)',
              fontSize: 9,
              fontFamily: 'var(--pixel-font-family)',
              cursor: 'pointer',
              letterSpacing: '0.08em',
            }}
          >
            HUD
          </button>
          <span style={{ fontSize: 10, color: 'rgba(100,100,140,0.3)', fontFamily: 'var(--pixel-font-family)' }}>·</span>
          <button
            onClick={() => setScreen('coupon')}
            style={{
              padding: '4px 12px',
              borderRadius: 4,
              border: '1px solid rgba(255,255,255,0.1)',
              background: screen === 'coupon' ? 'rgba(200,168,73,0.15)' : 'transparent',
              color: screen === 'coupon' ? 'rgba(200,168,73,0.8)' : 'rgba(100,100,140,0.4)',
              fontSize: 9,
              fontFamily: 'var(--pixel-font-family)',
              cursor: 'pointer',
              letterSpacing: '0.08em',
            }}
          >
            SHOP
          </button>
          <span style={{ fontSize: 10, color: 'rgba(100,100,140,0.3)', fontFamily: 'var(--pixel-font-family)' }}>·</span>
          <button
            onClick={() => setScreen('claim')}
            style={{
              padding: '4px 12px',
              borderRadius: 4,
              border: '1px solid rgba(255,255,255,0.1)',
              background: screen === 'claim' ? 'rgba(200,168,73,0.15)' : 'transparent',
              color: screen === 'claim' ? 'rgba(200,168,73,0.8)' : 'rgba(100,100,140,0.4)',
              fontSize: 9,
              fontFamily: 'var(--pixel-font-family)',
              cursor: 'pointer',
              letterSpacing: '0.08em',
            }}
          >
            CLAIM
          </button>
          <span style={{ fontSize: 9, color: 'rgba(100,100,140,0.3)', fontFamily: 'var(--pixel-font-family)', marginLeft: 6 }}>
            320 × 600
          </span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8, flexWrap: 'wrap' }}>
          <LanguageSwitcher
            currentLanguage={currentLanguage}
            onChangeLanguage={handleLanguageChange}
          />
          {!isPopupMode && (
            <>
              <span style={{ fontSize: 10, color: 'rgba(100,100,140,0.3)', fontFamily: 'var(--pixel-font-family)' }}>·</span>
              <button
                onClick={openPopupMode}
                style={{
                  padding: '4px 10px',
                  borderRadius: 4,
                  border: '1px solid rgba(255,255,255,0.1)',
                  background: 'transparent',
                  color: 'rgba(145,70,255,0.85)',
                  fontSize: 9,
                  fontFamily: 'var(--pixel-font-family)',
                  cursor: 'pointer',
                  letterSpacing: '0.08em',
                }}
              >
                POPUP
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
