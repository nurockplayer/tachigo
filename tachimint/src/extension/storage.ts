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

function warnStorageFailure(context: string, error: unknown) {
  console.warn(context, error)
}

function getLocalStorageState(): DemoState | null {
  if (typeof window === 'undefined') {
    return null
  }

  let raw: string | null

  try {
    raw = window.localStorage.getItem(STORAGE_KEY)
  } catch {
    return null
  }

  if (!raw) {
    return null
  }

  try {
    return sanitizeDemoState(JSON.parse(raw))
  } catch {
    return null
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
  const chromeStorage = getChromeStorageArea()

  if (chromeStorage) {
    try {
      const chromeState = await getChromeStoredState()

      if (chromeState) {
        return chromeState
      }
    } catch (error) {
      warnStorageFailure(
        'Failed to read Chrome storage in loadDemoState(); falling back to legacy storage/default state.',
        error,
      )
    }
  }

  const legacyState = getLocalStorageState()

  if (legacyState) {
    if (chromeStorage) {
      await setChromeStoredState(legacyState).catch((error) => {
        warnStorageFailure(
          'Failed to migrate legacy demo state into Chrome storage during loadDemoState().',
          error,
        )
      })
    }

    return legacyState
  }

  return createDefaultDemoState()
}

export async function saveDemoState(state: DemoState): Promise<void> {
  const sanitizedState = sanitizeDemoState(state)

  const chromeStorage = getChromeStorageArea()
  if (chromeStorage) {
    try {
      await setChromeStoredState(sanitizedState)
      return
    } catch (error) {
      warnStorageFailure(
        'Failed to write Chrome storage in saveDemoState(); falling back to localStorage.',
        error,
      )
    }
  }

  setLocalStorageState(sanitizedState)
}
