import i18n, { type BackendModule } from 'i18next'
import { initReactI18next } from 'react-i18next'

import { defaultNS } from './resources'
import { mapTwitchLocaleToAppLanguage, type AppLanguage } from './locale'

export { mapTwitchLocaleToAppLanguage }
export type { AppLanguage } from './locale'

const fallbackLanguage: AppLanguage = 'en'

const commonNamespaceLoaders = {
  en: () => import('./locales/en/common.json'),
  'zh-TW': () => import('./locales/zh-TW/common.json'),
  'zh-CN': () => import('./locales/zh-CN/common.json'),
} satisfies Record<AppLanguage, () => Promise<{ default: Record<string, unknown> }>>

function isAppLanguage(language: string): language is AppLanguage {
  return language === 'en' || language === 'zh-TW' || language === 'zh-CN'
}

const lazyCommonNamespaceBackend: BackendModule = {
  type: 'backend',
  init() {},
  read(language, namespace, callback) {
    if (namespace !== defaultNS) {
      callback(new Error(`Unsupported namespace: ${namespace}`), null)
      return
    }

    const appLanguage = isAppLanguage(language) ? language : fallbackLanguage
    commonNamespaceLoaders[appLanguage]()
      .then((resource) => callback(null, resource.default))
      .catch((error: unknown) => {
        callback(error instanceof Error ? error : String(error), null)
      })
  },
}

void i18n.use(lazyCommonNamespaceBackend).use(initReactI18next).init({
  lng: fallbackLanguage,
  fallbackLng: fallbackLanguage,
  supportedLngs: ['en', 'zh-TW', 'zh-CN'],
  ns: [defaultNS],
  defaultNS,
  interpolation: { escapeValue: false },
  react: { useSuspense: false },
}).catch((error: unknown) => {
  console.warn('Failed to initialize i18n', error)
})

export default i18n
