import assert from 'node:assert/strict'
import test, { afterEach } from 'node:test'

const STORAGE_KEY = 'tachigo.sidepanel.demo-state.v2'

const STORAGE_KEY = 'tachigo.sidepanel.demo-state.v2'

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

afterEach(() => {
  globalForTest.window = originalWindow
  globalForTest.chrome = originalChrome
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

test('loadDemoState falls back to localStorage when chrome storage key is missing', async () => {
  const localState = {
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
  }

  setWindowLocalStorage({
    getItem: (key) => {
      assert.equal(key, STORAGE_KEY)
      return JSON.stringify(localState)
    },
    setItem: () => {},
  })
  setChromeStorage({
    get: (_key, callback) => callback({}),
    set: (_items, callback) => callback(),
  })

  const storage = await importStorageModule()

  assert.deepEqual(await storage.loadDemoState(), localState)
})

test('loadDemoState falls back to localStorage when chrome storage read errors', async () => {
  const localState = {
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
  }

  setWindowLocalStorage({
    getItem: (key) => {
      assert.equal(key, STORAGE_KEY)
      return JSON.stringify(localState)
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

  assert.deepEqual(await storage.loadDemoState(), localState)
})

test('loadDemoState returns default state when localStorage read throws', async () => {
  setChromeStorage(undefined)
  setWindowLocalStorage({
    getItem: (key) => {
      assert.equal(key, STORAGE_KEY)
      throw new Error('denied')
    },
    setItem: () => {},
  })

  const storage = await importStorageModule()

  await assert.doesNotReject(() => storage.loadDemoState())
  assert.deepEqual(await storage.loadDemoState(), {
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
  })
})

test('saveDemoState sanitizes values and ignores localStorage write errors', async () => {
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
  assert.equal(written.key, 'tachigo.sidepanel.demo-state.v2')
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

test('saveDemoState falls back to localStorage when chrome storage write fails', async () => {
  let written: { key: string; value: string } | null = null

  setChromeStorage({
    get: (_key, callback) => callback({}),
    set: (_items, callback) => {
      if (globalForTest.chrome?.runtime) {
        globalForTest.chrome.runtime.lastError = { message: 'write failed' }
      }
      callback()
      if (globalForTest.chrome?.runtime) {
        delete globalForTest.chrome.runtime.lastError
      }
    },
  })
  setWindowLocalStorage({
    getItem: (key) => {
      assert.equal(key, STORAGE_KEY)
      return null
    },
    setItem: (key, value) => {
      assert.equal(key, STORAGE_KEY)
      written = { key, value }
    },
  })

  const storage = await importStorageModule()

  await assert.doesNotReject(() =>
    storage.saveDemoState({
      screen: 'claim',
      language: 'zh-CN',
      hud: {
        points: Number.NaN,
        totalPoints: 33,
        countdown: Number.POSITIVE_INFINITY,
        isWatching: true,
        clickCount: 2,
      },
      tcgBalance: -5,
      redeemedCouponIds: ['coupon-2'],
    }),
  )

  assert.ok(written)
  assert.deepEqual(JSON.parse(written.value), {
    screen: 'claim',
    language: 'zh-CN',
    hud: {
      points: 0,
      totalPoints: 33,
      countdown: 60,
      isWatching: true,
      clickCount: 2,
    },
    tcgBalance: 0,
    redeemedCouponIds: ['coupon-2'],
  })
})
