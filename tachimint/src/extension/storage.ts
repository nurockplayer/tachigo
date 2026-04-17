import { createDefaultDemoState, sanitizeDemoState, type DemoState } from './types.ts'

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

      if (!result || !Object.prototype.hasOwnProperty.call(result, STORAGE_KEY)) {
        resolve(null)
        return
      }

      resolve(sanitizeDemoState(result[STORAGE_KEY]))
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
    return createDefaultDemoState()
  }

  let raw: string | null

  try {
    raw = window.localStorage.getItem(STORAGE_KEY)
  } catch {
    return createDefaultDemoState()
  }

  if (!raw) {
    return createDefaultDemoState()
  }

  try {
    return sanitizeDemoState(JSON.parse(raw))
  } catch {
    return createDefaultDemoState()
  }
}

function setLocalStorageState(state: DemoState) {
  if (typeof window === 'undefined') {
    return
  }

  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
  } catch {
    // Ignore localStorage write failures in restricted environments.
  }
}

export async function loadDemoState(): Promise<DemoState> {
  if (getChromeStorageArea()) {
    const chromeState = await getChromeStoredState().catch(() => null)
    return chromeState ?? createDefaultDemoState()
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
