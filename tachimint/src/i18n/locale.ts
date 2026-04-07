export type AppLanguage = 'en' | 'zh-TW' | 'zh-CN'

export function mapTwitchLocaleToAppLanguage(locale: string): AppLanguage {
  const normalizedLocale = locale.trim().toLowerCase().replace(/_/g, '-')

  if (normalizedLocale === 'zh-tw' || normalizedLocale === 'zh-hk' || normalizedLocale === 'zh-mo') {
    return 'zh-TW'
  }
  if (normalizedLocale === 'zh-cn' || normalizedLocale === 'zh-sg' || normalizedLocale.startsWith('zh-hans')) {
    return 'zh-CN'
  }
  if (normalizedLocale.startsWith('zh-hant') || normalizedLocale.startsWith('zh')) return 'zh-TW'
  return 'en'
}
