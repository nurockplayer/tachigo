import '@testing-library/jest-dom/vitest'

import { afterEach, beforeEach, vi } from 'vitest'
import { cleanup } from '@testing-library/react'

import i18n from '../i18n'

beforeEach(async () => {
  window.localStorage.clear()
  await i18n.changeLanguage('en')
  vi.restoreAllMocks()
})

afterEach(() => {
  vi.useRealTimers()
  cleanup()
})
