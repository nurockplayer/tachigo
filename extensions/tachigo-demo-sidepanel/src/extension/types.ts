import type { AppLanguage } from '../i18n'

export type DemoScreen = 'login' | 'loading'

export interface DemoState {
  screen: DemoScreen
  language: AppLanguage
}

export const defaultDemoState: DemoState = {
  screen: 'login',
  language: 'en',
}

export function normalizeAppLanguage(language: string | null | undefined): AppLanguage {
  if (language === 'en' || language === 'zh-TW' || language === 'zh-CN') {
    return language
  }

  return defaultDemoState.language
}

export function sanitizeDemoState(value: unknown): DemoState {
  if (!value || typeof value !== 'object') {
    return defaultDemoState
  }

  const candidate = value as Partial<DemoState>
  const screen = candidate.screen === 'login' || candidate.screen === 'loading'
    ? candidate.screen
    : defaultDemoState.screen

  return {
    screen,
    language: normalizeAppLanguage(candidate.language),
  }
}
