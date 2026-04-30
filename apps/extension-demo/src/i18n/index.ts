import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'

import { defaultNS } from './resources'
import enCommon from './locales/en/common.json'
import zhTWCommon from './locales/zh-TW/common.json'
import zhCNCommon from './locales/zh-CN/common.json'

type AppLanguage = 'en' | 'zh-TW' | 'zh-CN'

const fallbackLanguage: AppLanguage = 'en'

const resources = {
  en: {
    [defaultNS]: enCommon,
  },
  'zh-TW': {
    [defaultNS]: zhTWCommon,
  },
  'zh-CN': {
    [defaultNS]: zhCNCommon,
  },
}

void i18n.use(initReactI18next).init({
  lng: fallbackLanguage,
  fallbackLng: fallbackLanguage,
  supportedLngs: ['en', 'zh-TW', 'zh-CN'],
  resources,
  ns: [defaultNS],
  defaultNS,
  interpolation: { escapeValue: false },
  react: { useSuspense: false },
}).catch((error: unknown) => {
  console.warn('Failed to initialize i18n', error)
})

export default i18n
export type { AppLanguage }
