import assert from 'node:assert/strict'
import { existsSync } from 'node:fs'
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

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

export function flattenStringValues(
  value: unknown,
  prefix = '',
): Record<string, string> {
  if (typeof value === 'string') {
    return prefix ? { [prefix]: value } : {}
  }

  if (!value || typeof value !== 'object') {
    return {}
  }

  if (Array.isArray(value)) {
    return value.reduce<Record<string, string>>((accumulator, child, index) => {
      const nextPrefix = prefix ? `${prefix}.${index}` : `${index}`
      return {
        ...accumulator,
        ...flattenStringValues(child, nextPrefix),
      }
    }, {})
  }

  return Object.entries(value).reduce<Record<string, string>>((accumulator, [key, child]) => {
    const nextPrefix = prefix ? `${prefix}.${key}` : key
    return {
      ...accumulator,
      ...flattenStringValues(child, nextPrefix),
    }
  }, {})
}

export function extractInterpolationTokens(value: string): string[] {
  return Array.from(
    new Set(
      Array.from(value.matchAll(/\{\{\s*([a-zA-Z0-9_]+)\s*\}\}/g), (match) => match[1]),
    ),
  ).sort()
}

export function checkLocaleParity(locales: Record<string, Record<string, unknown>>) {
  const entries = Object.entries(locales)
  assert.ok(entries.length > 0, 'locale parity check requires at least one locale')

  const [baseFile, baseLocale] = entries[0]
  const baseLocaleStrings = flattenStringValues(baseLocale)
  const baseKeys = Object.keys(baseLocaleStrings).sort()

  for (const [file, locale] of entries.slice(1)) {
    const localeStrings = flattenStringValues(locale)

    for (const key of baseKeys) {
      const localeValue = localeStrings[key]

      assert.equal(
        typeof localeValue,
        'string',
        `${file} should provide a string value for ${key}`,
      )

      assert.deepEqual(
        extractInterpolationTokens(localeValue),
        extractInterpolationTokens(baseLocaleStrings[key]),
        `${file} has mismatched interpolation tokens for ${key}`,
      )
    }

    assert.deepEqual(
      Object.keys(localeStrings).sort(),
      baseKeys,
      `${file} should have the same translation keys as ${baseFile}`,
    )
  }

  return {
    baseFile,
    baseKeys,
  }
}

const [baseFile] = localeFiles
const loadedLocales = Object.fromEntries(localeFiles.map((file) => [file, readLocale(file)]))
const { baseKeys } = checkLocaleParity(loadedLocales)
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

function runCheckI18n() {
  console.log(`i18n check passed: ${localeCases.length} locale mappings, ${baseKeys.length} keys`)
}

if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  runCheckI18n()
}
