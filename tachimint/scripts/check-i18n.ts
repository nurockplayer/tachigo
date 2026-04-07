import assert from 'node:assert/strict'
import { existsSync } from 'node:fs'
import { readFileSync } from 'node:fs'

import {
  mapTwitchLocaleToAppLanguage,
  type AppLanguage,
} from '../src/i18n/locale.ts'

const localeCases: Array<[locale: string, language: AppLanguage]> = [
  ['en', 'en'],
  ['en-US', 'en'],
  ['zh-TW', 'zh-TW'],
  ['zh-tw', 'zh-TW'],
  ['zh-HK', 'zh-TW'],
  ['zh_HK', 'zh-TW'],
  ['zh-MO', 'zh-TW'],
  ['zh-CN', 'zh-CN'],
  ['zh-cn', 'zh-CN'],
  ['zh-SG', 'zh-CN'],
  ['zh_sg', 'zh-CN'],
  ['zh-Hans', 'zh-CN'],
  ['zh_Hans', 'zh-CN'],
  ['zh-Hans-CN', 'zh-CN'],
  ['zh_Hans_CN', 'zh-CN'],
  ['zh-Hant', 'zh-TW'],
  ['zh-Hant-TW', 'zh-TW'],
  [' zh-CN ', 'zh-CN'],
  ['zh', 'zh-TW'],
  ['zh-unknown', 'zh-TW'],
  ['unknown', 'en'],
]

for (const [locale, expectedLanguage] of localeCases) {
  assert.equal(
    mapTwitchLocaleToAppLanguage(locale),
    expectedLanguage,
    `${locale} should map to ${expectedLanguage}`,
  )
}

const localeFiles = [
  'en/common.json',
  'zh-TW/common.json',
  'zh-CN/common.json',
] as const

function readLocale(file: string): Record<string, unknown> {
  return JSON.parse(
    readFileSync(new URL(`../src/i18n/locales/${file}`, import.meta.url), 'utf8'),
  ) as Record<string, unknown>
}

function flattenKeys(value: unknown, prefix = ''): string[] {
  if (!value || typeof value !== 'object') return [prefix]

  if (Array.isArray(value)) {
    return value.flatMap((child, index) => {
      const nextPrefix = prefix ? `${prefix}.${index}` : `${index}`
      return flattenKeys(child, nextPrefix)
    })
  }

  return Object.entries(value).flatMap(([key, child]) => {
    const nextPrefix = prefix ? `${prefix}.${key}` : key
    return flattenKeys(child, nextPrefix)
  })
}

const [baseFile, ...translatedFiles] = localeFiles
const baseKeys = flattenKeys(readLocale(baseFile)).sort()
const requiredKeys = [
  'contextLoading.title',
  'contextLoading.subtitle',
  'common.retry',
  'common.initializing',
  'common.points',
]

for (const key of requiredKeys) {
  assert.ok(baseKeys.includes(key), `${baseFile} should include ${key}`)
}

for (const file of translatedFiles) {
  assert.deepEqual(
    flattenKeys(readLocale(file)).sort(),
    baseKeys,
    `${file} should have the same translation keys as ${baseFile}`,
  )
}

function readProjectFile(file: string): string {
  return readFileSync(new URL(`../${file}`, import.meta.url), 'utf8')
}

const appSource = readProjectFile('src/App.tsx')
assert.ok(
  !appSource.includes("t('contextLoading.title')") && !appSource.includes("t('contextLoading.subtitle')"),
  'pre-context loading UI should stay language-neutral until Twitch onContext supplies a locale',
)

const i18nSource = readProjectFile('src/i18n/index.ts')
assert.ok(
  !i18nSource.includes("from './locales/"),
  'i18n initialization should lazy-load locale JSON instead of eager importing all locales',
)
assert.ok(
  /\.init\(\{[\s\S]*?\}\)\s*\.catch\(/.test(i18nSource),
  'i18n init promise should be caught so lazy locale chunk failures do not become unhandled rejections',
)

assert.ok(
  existsSync(new URL('../src/i18n/resources.ts', import.meta.url)),
  'i18n should expose typed resource metadata for translation key checking',
)
assert.ok(
  existsSync(new URL('../src/i18n/i18next.d.ts', import.meta.url)),
  'i18n should augment i18next types for translation key checking',
)

const viteConfigSource = readProjectFile('vite.config.ts')
assert.match(
  viteConfigSource,
  /chunkSizeWarningLimit/,
  'vite config should include a chunk size warning limit for the extension bundle',
)

console.log(`i18n check passed: ${localeCases.length} locale mappings, ${baseKeys.length} keys`)
