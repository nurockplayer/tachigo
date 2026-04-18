import assert from 'node:assert/strict'
import test from 'node:test'

import type { DemoState } from './types.ts'

const STORAGE_KEY = 'tachigo.sidepanel.demo-state.v2'

const EXPECTED_DEFAULT_DEMO_STATE: DemoState = {
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

type ChromeStorageCallbacks = {
  get?: (key: string, callback: (result: Record<string, unknown>) => void) => void
  set?: (items: Record<string, unknown>, callback: () => void) => void
}

type MockLocalStorage = {
  getItem: (key: string) => string | null
  setItem: (key: string, value: string) => void
  clear: () => void
  reads: number
  writes: number
}

function setChromeStorage(callbacks?: ChromeStorageCallbacks) {
  if (!callbacks) {
    delete (globalThis as typeof globalThis & { chrome?: unknown }).chrome
    return
  }

  ;(globalThis as typeof globalThis & { chrome?: unknown }).chrome = {
    runtime: {
      lastError: undefined,
    },
    storage: {
      local: {
        get: callbacks.get ?? ((_key: string, callback: (result: Record<string, unknown>) => void) => callback({})),
        set: callbacks.set ?? ((_items: Record<string, unknown>, callback: () => void) => callback()),
      },
    },
  }
}

function setWindowLocalStorage(initialValue?: DemoState | null): MockLocalStorage {
  const store = new Map<string, string>()

  if (initialValue) {
    store.set(STORAGE_KEY, JSON.stringify(initialValue))
  }

  const localStorage: MockLocalStorage = {
    reads: 0,
    writes: 0,
    getItem(key) {
      this.reads += 1
      return store.get(key) ?? null
    },
    setItem(key, value) {
      this.writes += 1
      store.set(key, value)
    },
    clear() {
      store.clear()
    },
  }

  ;(globalThis as typeof globalThis & { window?: unknown }).window = {
    localStorage,
  }

  return localStorage
}

async function importStorageModule() {
  return import(`./storage.ts?test=${Date.now()}-${Math.random()}`)
}

test.afterEach(() => {
  delete (globalThis as typeof globalThis & { chrome?: unknown }).chrome
  delete (globalThis as typeof globalThis & { window?: unknown }).window
})

test('loadDemoState returns english defaults on first launch', async () => {
  setChromeStorage()
  setWindowLocalStorage()

  const storage = await importStorageModule()

  assert.deepEqual(await storage.loadDemoState(), EXPECTED_DEFAULT_DEMO_STATE)
})

test('loadDemoState returns sanitized chrome storage state without reading localStorage', async () => {
  setChromeStorage({
    get: (_key, callback) => callback({
      [STORAGE_KEY]: {
        screen: 'hud',
        language: 'zh-TW',
        hud: {
          points: -5,
          totalPoints: Number.POSITIVE_INFINITY,
          countdown: 12,
          isWatching: false,
          clickCount: -0,
        },
        tcgBalance: -9,
        redeemedCouponIds: ['coupon-1', 7, 'coupon-2'],
      },
    }),
  })
  const localStorage = setWindowLocalStorage()

  const storage = await importStorageModule()

  assert.deepEqual(await storage.loadDemoState(), {
    screen: 'hud',
    language: 'zh-TW',
    hud: {
      points: 0,
      totalPoints: 12847,
      countdown: 12,
      isWatching: false,
      clickCount: 0,
    },
    tcgBalance: 0,
    redeemedCouponIds: ['coupon-1', 'coupon-2'],
  })
  assert.equal(localStorage.reads, 0)
})

test('loadDemoState falls back to legacy localStorage state when chrome storage has no saved value', async () => {
  setChromeStorage({
    get: (_key, callback) => callback({}),
  })
  const localStorage = setWindowLocalStorage({
    screen: 'coupon',
    language: 'zh-CN',
    hud: {
      points: 48,
      totalPoints: 2048,
      countdown: 9,
      isWatching: false,
      clickCount: 3,
    },
    tcgBalance: 12,
    redeemedCouponIds: ['bundle-120'],
  })

  const storage = await importStorageModule()

  assert.deepEqual(await storage.loadDemoState(), {
    screen: 'coupon',
    language: 'zh-CN',
    hud: {
      points: 48,
      totalPoints: 2048,
      countdown: 9,
      isWatching: false,
      clickCount: 3,
    },
    tcgBalance: 12,
    redeemedCouponIds: ['bundle-120'],
  })
  assert.equal(localStorage.reads, 1)
})

test('saveDemoState falls back to localStorage when chrome storage write fails', async () => {
  setChromeStorage({
    set: (_items, callback) => {
      ;(globalThis.chrome as { runtime?: { lastError?: { message: string } } }).runtime = {
        lastError: { message: 'write failed' },
      }
      callback()
      ;(globalThis.chrome as { runtime?: { lastError?: { message: string } } }).runtime = {
        lastError: undefined,
      }
    },
  })
  const localStorage = setWindowLocalStorage()

  const storage = await importStorageModule()

  await storage.saveDemoState({
    screen: 'hud',
    language: 'zh-TW',
    hud: {
      points: 80,
      totalPoints: 2048,
      countdown: 14,
      isWatching: true,
      clickCount: 6,
    },
    tcgBalance: 5,
    redeemedCouponIds: ['tachiya-95'],
  })

  assert.equal(localStorage.writes, 1)
  assert.deepEqual(JSON.parse(localStorage.getItem(STORAGE_KEY) ?? 'null'), {
    screen: 'hud',
    language: 'zh-TW',
    hud: {
      points: 80,
      totalPoints: 2048,
      countdown: 14,
      isWatching: true,
      clickCount: 6,
    },
    tcgBalance: 5,
    redeemedCouponIds: ['tachiya-95'],
  })
})
