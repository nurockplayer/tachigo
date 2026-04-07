import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'

import enCommon from './locales/en/common.json'
import zhCnCommon from './locales/zh-CN/common.json'
import zhTwCommon from './locales/zh-TW/common.json'
export { mapTwitchLocaleToAppLanguage } from './locale'
export type { AppLanguage } from './locale'

i18n.use(initReactI18next).init({
  resources: {
    en: { common: enCommon },
    'zh-TW': { common: zhTwCommon },
    'zh-CN': { common: zhCnCommon },
  },
  lng: 'en',
  fallbackLng: 'en',
  defaultNS: 'common',
  interpolation: { escapeValue: false },
})

export default i18n
