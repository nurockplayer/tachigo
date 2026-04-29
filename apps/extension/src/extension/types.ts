import type { AppLanguage } from '../i18n'

export type DemoScreen = 'login' | 'loading' | 'hud' | 'claim' | 'coupon' | 'raffle'

export interface RaffleResultEntry {
  id: string
  twitch_login: string
  display_name: string
}

export interface RaffleResultDraw {
  id: string
  raffle_id: string
  drawn_at: string
  entry: RaffleResultEntry
}

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
  hud: { ...defaultHudDemoState },
  tcgBalance: 0,
  redeemedCouponIds: [],
}

export function createDefaultHudDemoState(): HudDemoState {
  return { ...defaultHudDemoState }
}

export function createDefaultDemoState(): DemoState {
  return {
    ...defaultDemoState,
    hud: createDefaultHudDemoState(),
    redeemedCouponIds: [...defaultDemoState.redeemedCouponIds],
  }
}

function toFiniteNumber(value: unknown, fallback: number) {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

function toNonNegativeFiniteNumber(value: unknown, fallback: number) {
  return Math.max(0, toFiniteNumber(value, fallback))
}

export function normalizeAppLanguage(language: string | null | undefined): AppLanguage {
  if (language === 'en' || language === 'zh-TW' || language === 'zh-CN') {
    return language
  }

  return defaultDemoState.language
}

export function sanitizeHudDemoState(value: unknown): HudDemoState {
  if (!value || typeof value !== 'object') {
    return createDefaultHudDemoState()
  }

  const candidate = value as Partial<HudDemoState>

  return {
    points: toNonNegativeFiniteNumber(candidate.points, defaultHudDemoState.points),
    totalPoints: toNonNegativeFiniteNumber(candidate.totalPoints, defaultHudDemoState.totalPoints),
    countdown: toNonNegativeFiniteNumber(candidate.countdown, defaultHudDemoState.countdown),
    isWatching: typeof candidate.isWatching === 'boolean' ? candidate.isWatching : defaultHudDemoState.isWatching,
    clickCount: toNonNegativeFiniteNumber(candidate.clickCount, defaultHudDemoState.clickCount),
  }
}

export function sanitizeDemoState(value: unknown): DemoState {
  if (!value || typeof value !== 'object') {
    return createDefaultDemoState()
  }

  const candidate = value as Partial<DemoState>
  const screen = candidate.screen === 'login' || candidate.screen === 'loading' || candidate.screen === 'hud' || candidate.screen === 'claim' || candidate.screen === 'coupon' || candidate.screen === 'raffle'
    ? candidate.screen
    : defaultDemoState.screen

  const tcgBalance =
    typeof candidate.tcgBalance === 'number' && Number.isFinite(candidate.tcgBalance)
      ? Math.max(0, candidate.tcgBalance)
      : defaultDemoState.tcgBalance

  const redeemedRaw = candidate.redeemedCouponIds
  const redeemedCouponIds = Array.isArray(redeemedRaw)
    ? redeemedRaw.filter((id): id is string => typeof id === 'string')
    : createDefaultDemoState().redeemedCouponIds

  return {
    screen,
    language: normalizeAppLanguage(candidate.language),
    hud: sanitizeHudDemoState(candidate.hud),
    tcgBalance,
    redeemedCouponIds: [...redeemedCouponIds],
  }
}
