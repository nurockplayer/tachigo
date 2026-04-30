import { defaultDemoState, sanitizeDemoState, type DemoState } from './types'

const STORAGE_KEY = 'tachigo.sidepanel.demo-state.v2'

function getChromeStorageArea() {
  return globalThis.chrome?.storage?.local
}

async function getChromeStoredState(): Promise<DemoState | null> {
  const storage = getChromeStorageArea()

  if (!storage) {
    return null
  }

  return new Promise((resolve, reject) => {
    storage.get(STORAGE_KEY, (result) => {
      const runtimeError = globalThis.chrome?.runtime?.lastError

      if (runtimeError) {
        reject(new Error(runtimeError.message))
        return
      }

      resolve(sanitizeDemoState(result?.[STORAGE_KEY]))
    })
  })
}

async function setChromeStoredState(state: DemoState): Promise<void> {
  const storage = getChromeStorageArea()

  if (!storage) {
    return
  }

  return new Promise((resolve, reject) => {
    storage.set({ [STORAGE_KEY]: state }, () => {
      const runtimeError = globalThis.chrome?.runtime?.lastError

      if (runtimeError) {
        reject(new Error(runtimeError.message))
        return
      }

      resolve()
    })
  })
}

function getLocalStorageState(): DemoState {
  if (typeof window === 'undefined') {
    return defaultDemoState
  }

  const raw = window.localStorage.getItem(STORAGE_KEY)

  if (!raw) {
    return defaultDemoState
  }

  try {
    return sanitizeDemoState(JSON.parse(raw))
  } catch {
    return defaultDemoState
  }
}

function setLocalStorageState(state: DemoState) {
  if (typeof window === 'undefined') {
    return
  }

  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
}

export async function loadDemoState(): Promise<DemoState> {
  const chromeState = await getChromeStoredState().catch(() => null)

  if (chromeState) {
    return chromeState
  }

  return getLocalStorageState()
}

export async function saveDemoState(state: DemoState): Promise<void> {
  const sanitizedState = sanitizeDemoState(state)

  const chromeStorage = getChromeStorageArea()
  if (chromeStorage) {
    await setChromeStoredState(sanitizedState)
    return
  }

  setLocalStorageState(sanitizedState)
}
