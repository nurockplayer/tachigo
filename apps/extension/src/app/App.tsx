import { useEffect, useRef, useState, type CSSProperties, type Dispatch, type SetStateAction } from 'react'
import { useTranslation } from 'react-i18next'

import { loadDemoState, saveDemoState } from '../extension/storage'
import {
  defaultDemoState,
  defaultSettingsState,
  normalizeAppLanguage,
  type HudDemoState,
  type SettingsState,
} from '../extension/types'
import type { NavigationFlags } from './navigation/types'
import type { AppLanguage } from '../i18n'
import { OnboardingOverlay } from './components/OnboardingOverlay'
import { DevNavigationControls } from './navigation/DevNavigationControls'
import { NavigationProvider } from './navigation/NavigationProvider'
import { OverlayHost } from './navigation/OverlayHost'
import { SceneRenderer } from './navigation/SceneRenderer'
import { useNavigation } from './navigation/useNavigation'
import { markOnboardingComplete, shouldShowOnboarding } from './onboarding'
import { useTwitch } from '../hooks/useTwitch'
import { claimFromHudState } from './claimState'
import { executeCouponRedeem, type CouponRedeemOutcome } from './couponRedeem'
import { applyChatMessageToHudState } from './chatMessages'

export default function App() {
  const { i18n } = useTranslation()
  const { jwt } = useTwitch()
  const isPopupMode = typeof window !== 'undefined' && window.location.pathname.endsWith('/popup.html')
  const [currentLanguage, setCurrentLanguage] = useState<AppLanguage>(defaultDemoState.language)
  const useZpixLanguage = currentLanguage === 'zh-TW' || currentLanguage === 'zh-CN'
  const fontVariables = {
    '--ui-font-family': useZpixLanguage
      ? "'Press Start 2P', 'Zpix CJK', 'Inter', system-ui, sans-serif"
      : "'Inter', system-ui, sans-serif",
    '--pixel-font-family': useZpixLanguage
      ? "'Press Start 2P', 'Zpix CJK', monospace"
      : "'Press Start 2P', monospace",
  } as CSSProperties
  const [isHydrated, setIsHydrated] = useState(false)
  const [flags, setFlags] = useState<NavigationFlags>(() => ({ ...defaultDemoState.flags }))
  const [hudState, setHudState] = useState<HudDemoState>(defaultDemoState.hud)
  const [tcgBalance, setTcgBalance] = useState(defaultDemoState.tcgBalance)
  const [redeemedCouponIds, setRedeemedCouponIds] = useState<string[]>(defaultDemoState.redeemedCouponIds)
  const [settings, setSettings] = useState<SettingsState>(defaultSettingsState)
  const [voucherCodes, setVoucherCodes] = useState<Record<string, string>>({})
  const [currentRaffleId, setCurrentRaffleId] = useState('')
  const tcgBalanceRef = useRef(defaultDemoState.tcgBalance)
  const redeemedCouponIdsRef = useRef<string[]>([...defaultDemoState.redeemedCouponIds])

  useEffect(() => {
    let isActive = true

    void loadDemoState()
      .then(async (storedState) => {
        if (!isActive) {
          return
        }

        setFlags(storedState.flags)
        setHudState(storedState.hud)
        setTcgBalance(storedState.tcgBalance)
        tcgBalanceRef.current = storedState.tcgBalance
        setRedeemedCouponIds(storedState.redeemedCouponIds)
        redeemedCouponIdsRef.current = [...storedState.redeemedCouponIds]
        setSettings(storedState.settings)

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
        language: currentLanguage,
        flags,
        hud: hudState,
        tcgBalance,
        redeemedCouponIds,
        settings,
      }).catch((error: unknown) => {
        console.warn('Failed to persist extension demo state', error)
      })
    }, 120)

    return () => window.clearTimeout(persistTimer)
  }, [currentLanguage, flags, hudState, isHydrated, redeemedCouponIds, settings, tcgBalance])

  useEffect(() => {
    if (!isHydrated) {
      return
    }

    const runtimeMessages = globalThis.chrome?.runtime?.onMessage
    if (!runtimeMessages) {
      return
    }

    const handleRuntimeMessage = (message: unknown) => {
      setHudState((previousHudState) => applyChatMessageToHudState(previousHudState, message))
    }

    runtimeMessages.addListener(handleRuntimeMessage)
    return () => runtimeMessages.removeListener(handleRuntimeMessage)
  }, [isHydrated])

  const handleClaim = (cpcAmount: number) => {
    setHudState((previousHudState) => {
      const { claimable, nextHudState } = claimFromHudState(previousHudState, cpcAmount)
      if (claimable === 0) {
        return previousHudState
      }

      const tcgGained = Number((claimable * 0.1).toFixed(2))
      setTcgBalance((previousTcgBalance) => {
        const nextTcgBalance = Number((previousTcgBalance + tcgGained).toFixed(2))
        tcgBalanceRef.current = nextTcgBalance
        return nextTcgBalance
      })

      return nextHudState
    })
  }

  const handleCouponRedeem = async (couponId: string, cost: number): Promise<CouponRedeemOutcome> => {
    return executeCouponRedeem({
      couponId,
      cost,
      jwt,
      redeemedCouponIdsRef,
      setTcgBalance: (nextBalance) => {
        tcgBalanceRef.current = nextBalance
        setTcgBalance(nextBalance)
      },
      setVoucherCodes,
      setRedeemedCouponIds,
    })
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
  }

  if (!isHydrated) {
    return <HydrationShell fontVariables={fontVariables} />
  }

  return (
    <NavigationProvider initialFlags={flags}>
      <AppShell
        currentLanguage={currentLanguage}
        currentRaffleId={currentRaffleId}
        fontVariables={fontVariables}
        hudState={hudState}
        isPopupMode={isPopupMode}
        redeemedCouponIds={redeemedCouponIds}
        settings={settings}
        tcgBalance={tcgBalance}
        voucherCodes={voucherCodes}
        onChangeLanguage={handleLanguageChange}
        onChangeRaffleId={setCurrentRaffleId}
        onClaim={handleClaim}
        onCouponRedeem={handleCouponRedeem}
        onFlagsChange={setFlags}
        onHudStateChange={setHudState}
        onOpenPopupMode={openPopupMode}
        onSettingsChange={setSettings}
      />
    </NavigationProvider>
  )
}

interface AppShellProps {
  currentLanguage: AppLanguage
  currentRaffleId: string
  fontVariables: CSSProperties
  hudState: HudDemoState
  isPopupMode: boolean
  redeemedCouponIds: string[]
  settings: SettingsState
  tcgBalance: number
  voucherCodes: Record<string, string>
  onChangeLanguage: (language: AppLanguage) => void
  onChangeRaffleId: (raffleId: string) => void
  onClaim: (cpcAmount: number) => void
  onCouponRedeem: (couponId: string, cost: number) => Promise<CouponRedeemOutcome>
  onFlagsChange: Dispatch<SetStateAction<NavigationFlags>>
  onHudStateChange: Dispatch<SetStateAction<HudDemoState>>
  onOpenPopupMode: () => void
  onSettingsChange: Dispatch<SetStateAction<SettingsState>>
}

function AppShell({
  currentLanguage,
  currentRaffleId,
  fontVariables,
  hudState,
  isPopupMode,
  redeemedCouponIds,
  settings,
  tcgBalance,
  voucherCodes,
  onChangeLanguage,
  onChangeRaffleId,
  onClaim,
  onCouponRedeem,
  onFlagsChange,
  onHudStateChange,
  onOpenPopupMode,
  onSettingsChange,
}: AppShellProps) {
  const { state, setFlag } = useNavigation()
  const showOnboarding = shouldShowOnboarding(state.scene, state.flags)

  useEffect(() => {
    onFlagsChange(state.flags)
  }, [onFlagsChange, state.flags])

  const completeOnboarding = () => {
    const nextFlags = markOnboardingComplete(state.flags)
    setFlag('onboardingVersion', nextFlags.onboardingVersion)
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
      <div
        style={{
          width: 'min(100%, 430px)',
          minWidth: 320,
          minHeight: 600,
          height:
            settings.screenMode === 'focus'
              ? 'min(820px, calc(100vh - 40px))'
              : 'min(720px, calc(100vh - 104px))',
          borderRadius: 12,
          overflow: 'hidden',
          border: '1px solid rgba(255,255,255,0.07)',
          boxShadow: '0 0 0 1px rgba(0,0,0,0.9), 0 8px 48px rgba(0,0,0,0.9)',
          flexShrink: 0,
          position: 'relative',
        }}
      >
        <SceneRenderer hudState={hudState} settings={settings} onHudStateChange={onHudStateChange} />
        <OverlayHost
          cpcBalance={hudState.points}
          currentLanguage={currentLanguage}
          tcgBalance={tcgBalance}
          redeemedCouponIds={redeemedCouponIds}
          settings={settings}
          voucherCodes={voucherCodes}
          onChangeLanguage={onChangeLanguage}
          onClaim={onClaim}
          onCouponRedeem={onCouponRedeem}
          onSettingsChange={onSettingsChange}
        />
        {showOnboarding ? <OnboardingOverlay onComplete={completeOnboarding} /> : null}
      </div>

      <DevNavigationControls
        isDev={import.meta.env.DEV}
        isPopupMode={isPopupMode}
        currentLanguage={currentLanguage}
        raffleId={currentRaffleId}
        onChangeLanguage={onChangeLanguage}
        onChangeRaffleId={onChangeRaffleId}
        onOpenPopupMode={onOpenPopupMode}
      />
    </div>
  )
}

function HydrationShell({ fontVariables }: { fontVariables: CSSProperties }) {
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
