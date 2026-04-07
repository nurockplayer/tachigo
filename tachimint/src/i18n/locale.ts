export type AppLanguage = 'en' | 'zh-TW' | 'zh-CN'

export function mapTwitchLocaleToAppLanguage(locale: string): AppLanguage {
  if (locale === 'zh-TW' || locale === 'zh-HK' || locale === 'zh-MO') return 'zh-TW'
  if (locale === 'zh-CN' || locale === 'zh-SG') return 'zh-CN'
  if (locale.startsWith('zh')) return 'zh-TW'
  return 'en'
}
