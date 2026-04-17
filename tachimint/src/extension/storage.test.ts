import assert from 'node:assert/strict'
import test, { afterEach } from 'node:test'

const STORAGE_KEY = 'tachigo.sidepanel.demo-state.v2'
const EXPECTED_DEFAULT_DEMO_STATE = {
  screen: 'login',
  language: 'en',
  hud: {
    points: 0,
    totalPoints: 12847,
    countdown: 60,
    isWatching: true,
    clickCount: 0,
  },
  tcgBalance: 0,
  redeemedCouponIds: [],
}

type StorageLike = {
  getItem: (key: string) => string | null
  setItem: (key: string, value: string) => void
}

type ChromeStorageArea = {
  get: (key: string, callback: (result: Record<string, unknown>) => void) => void
  set: (items: Record<string, unknown>, callback: () => void) => void
}

type TestGlobal = typeof globalThis & {
  chrome?: {
    runtime?: { lastError?: { message?: string } }
    storage?: { local?: ChromeStorageArea }
  }
  window?: { localStorage: StorageLike }
}

const globalForTest = globalThis as TestGlobal
const originalWindow = globalForTest.window
const originalChrome = globalForTest.chrome
const originalConsoleWarn = console.warn

afterEach(() => {
  globalForTest.window = originalWindow
  globalForTest.chrome = originalChrome
  console.warn = originalConsoleWarn
})

function setWindowLocalStorage(localStorage: StorageLike) {
  globalForTest.window = { localStorage }
}

function setChromeStorage(local: ChromeStorageArea | undefined) {
  globalForTest.chrome = local
    ? {
        runtime: {},
        storage: { local },
      }
    : undefined
}

async function importStorageModule() {
  return import(`./storage.ts?test=${Date.now()}-${Math.random()}`)
}

test('loadDemoState returns default state when chrome storage key is missing', async () => {
  let localStorageReads = 0

  setWindowLocalStorage({
    getItem: (key) => {
      assert.equal(key, STORAGE_KEY)
      localStorageReads += 1
      return JSON.stringify({
        screen: 'coupon',
        language: 'zh-TW',
        hud: {
          points: 7,
          totalPoints: 99,
          countdown: 12,
          isWatching: false,
          clickCount: 3,
        },
        tcgBalance: 8,
        redeemedCouponIds: ['coupon-1'],
      })
    },
    setItem: () => {},
  })
  setChromeStorage({
    get: (_key, callback) => callback({}),
    set: (_items, callback) => callback(),
  })

  const storage = await importStorageModule()

  assert.deepEqual(await storage.loadDemoState(), EXPECTED_DEFAULT_DEMO_STATE)
  assert.equal(localStorageReads, 0)
})

test('loadDemoState returns default state when chrome storage read errors', async () => {
  let localStorageReads = 0
  const warnings: unknown[][] = []

  console.warn = (...args: unknown[]) => {
    warnings.push(args)
  }

  setWindowLocalStorage({
    getItem: (key) => {
      assert.equal(key, STORAGE_KEY)
      localStorageReads += 1
      return JSON.stringify({
        screen: 'hud',
        language: 'en',
        hud: {
          points: 4,
          totalPoints: 20,
          countdown: 10,
          isWatching: true,
          clickCount: 2,
        },
        tcgBalance: 5,
        redeemedCouponIds: [],
      })
    },
    setItem: () => {},
  })
  setChromeStorage({
    get: (_key, callback) => {
      if (globalForTest.chrome?.runtime) {
        globalForTest.chrome.runtime.lastError = { message: 'storage failed' }
      }
      callback({})
      if (globalForTest.chrome?.runtime) {
        delete globalForTest.chrome.runtime.lastError
      }
    },
    set: (_items, callback) => callback(),
  })

  const storage = await importStorageModule()

  assert.deepEqual(await storage.loadDemoState(), EXPECTED_DEFAULT_DEMO_STATE)
  assert.equal(localStorageReads, 0)
  assert.equal(warnings.length, 1)
  assert.match(String(warnings[0]?.[0]), /Failed to read demo state from chrome\.storage\.local/)
})

test('loadDemoState returns sanitized chrome storage state without reading localStorage', async () => {
  let localStorageReads = 0

  setWindowLocalStorage({
    getItem: (key) => {
      assert.equal(key, STORAGE_KEY)
      localStorageReads += 1
      return JSON.stringify({ screen: 'coupon' })
    },
    setItem: () => {},
  })
  setChromeStorage({
    get: (_key, callback) =>
      callback({
        [STORAGE_KEY]: {
          screen: 'coupon',
          language: 'unknown',
          hud: {
            points: -5,
            totalPoints: 24,
            countdown: -1,
            isWatching: true,
            clickCount: Number.NaN,
          },
          tcgBalance: -8,
          redeemedCouponIds: ['coupon-1', 12],
        },
      }),
    set: (_items, callback) => callback(),
  })

  const storage = await importStorageModule()

  assert.deepEqual(await storage.loadDemoState(), {
    screen: 'coupon',
    language: 'en',
    hud: {
      points: 0,
      totalPoints: 24,
      countdown: 0,
      isWatching: true,
      clickCount: 0,
    },
    tcgBalance: 0,
    redeemedCouponIds: ['coupon-1'],
  })
  assert.equal(localStorageReads, 0)
})

test('saveDemoState sanitizes values and ignores localStorage write errors when chrome storage is unavailable', async () => {
  let written: { key: string; value: string } | null = null

  setChromeStorage(undefined)
  setWindowLocalStorage({
    getItem: (key) => {
      assert.equal(key, STORAGE_KEY)
      return null
    },
    setItem: (key, value) => {
      assert.equal(key, STORAGE_KEY)
      written = { key, value }
      throw new Error('quota exceeded')
    },
  })

  const storage = await importStorageModule()

  await assert.doesNotReject(() =>
    storage.saveDemoState({
      screen: 'hud',
      language: 'zh-TW',
      hud: {
        points: Number.NaN,
        totalPoints: Number.POSITIVE_INFINITY,
        countdown: Number.NaN,
        isWatching: false,
        clickCount: Number.NEGATIVE_INFINITY,
      },
      tcgBalance: -10,
      redeemedCouponIds: ['coupon-1', 2 as never],
    }),
  )

  assert.ok(written)
  assert.equal(written.key, STORAGE_KEY)
  assert.deepEqual(JSON.parse(written.value), {
    screen: 'hud',
    language: 'zh-TW',
    hud: {
      points: 0,
      totalPoints: 12847,
      countdown: 60,
      isWatching: false,
      clickCount: 0,
    },
    tcgBalance: 0,
    redeemedCouponIds: ['coupon-1'],
  })
})
