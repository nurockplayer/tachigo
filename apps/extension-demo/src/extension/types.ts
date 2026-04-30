import type { AppLanguage } from '../i18n'

export type DemoScreen = 'login' | 'loading' | 'hud' | 'claim' | 'coupon'

export interface HudDemoState {
  points: number
  totalPoints: number
  countdown: number
  isWatching: boolean
  clickCount: number
}

export type CouponRedeemResult = 'success' | 'insufficient' | 'already_redeemed'

export interface DemoState {
  screen: DemoScreen
  language: AppLanguage
  hud: HudDemoState
  tcgBalance: number
  redeemedCouponIds: string[]
}

export const defaultHudDemoState: HudDemoState = {
  points: 0,
  totalPoints: 12847,
  countdown: 60,
  isWatching: true,
  clickCount: 0,
}

export const defaultDemoState: DemoState = {
  screen: 'login',
  language: 'en',
  hud: defaultHudDemoState,
  tcgBalance: 0,
  redeemedCouponIds: [],
}

export function normalizeAppLanguage(language: string | null | undefined): AppLanguage {
  if (language === 'en' || language === 'zh-TW' || language === 'zh-CN') {
    return language
  }

  return defaultDemoState.language
}

export function sanitizeHudDemoState(value: unknown): HudDemoState {
  if (!value || typeof value !== 'object') {
    return defaultHudDemoState
  }

  const candidate = value as Partial<HudDemoState>

  return {
    points: typeof candidate.points === 'number' ? candidate.points : defaultHudDemoState.points,
    totalPoints: typeof candidate.totalPoints === 'number' ? candidate.totalPoints : defaultHudDemoState.totalPoints,
    countdown: typeof candidate.countdown === 'number' ? candidate.countdown : defaultHudDemoState.countdown,
    isWatching: typeof candidate.isWatching === 'boolean' ? candidate.isWatching : defaultHudDemoState.isWatching,
    clickCount: typeof candidate.clickCount === 'number' ? candidate.clickCount : defaultHudDemoState.clickCount,
  }
}

export function sanitizeDemoState(value: unknown): DemoState {
  if (!value || typeof value !== 'object') {
    return defaultDemoState
  }

  const candidate = value as Partial<DemoState>
  const screen = candidate.screen === 'login' || candidate.screen === 'loading' || candidate.screen === 'hud' || candidate.screen === 'claim' || candidate.screen === 'coupon'
    ? candidate.screen
    : defaultDemoState.screen

  const tcgBalance =
    typeof candidate.tcgBalance === 'number' && Number.isFinite(candidate.tcgBalance)
      ? Math.max(0, candidate.tcgBalance)
      : defaultDemoState.tcgBalance

  const redeemedRaw = candidate.redeemedCouponIds
  const redeemedCouponIds = Array.isArray(redeemedRaw)
    ? redeemedRaw.filter((id): id is string => typeof id === 'string')
    : defaultDemoState.redeemedCouponIds

  return {
    screen,
    language: normalizeAppLanguage(candidate.language),
    hud: sanitizeHudDemoState(candidate.hud),
    tcgBalance,
    redeemedCouponIds,
  }
}
