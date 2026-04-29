import assert from 'node:assert/strict'
import test, { afterEach } from 'node:test'

type ChromeMock = typeof globalThis & {
  chrome?: {
    runtime: {
      onInstalled: { addListener: (callback: () => void) => void }
      onStartup: { addListener: (callback: () => void) => void }
      onMessage: { addListener: (callback: () => undefined) => void }
    }
    sidePanel: {
      setPanelBehavior: (options: { openPanelOnActionClick: boolean }) => Promise<void>
    }
    action: {
      onClicked: { addListener: (callback: (tab: { windowId?: number }) => void) => void }
    }
  }
}

const globalForTest = globalThis as ChromeMock
const originalChrome = globalForTest.chrome

afterEach(() => {
  globalForTest.chrome = originalChrome
})

async function importBackgroundModule() {
  return import(`./background.ts?test=${Date.now()}-${Math.random()}`)
}

test('background enables openPanelOnActionClick without registering manual action click handling', async () => {
  const installedListeners: Array<() => void> = []
  const startupListeners: Array<() => void> = []
  const messageListeners: Array<() => undefined> = []
  const panelBehaviorCalls: Array<{ openPanelOnActionClick: boolean }> = []
  let actionClickListeners = 0

  globalForTest.chrome = {
    runtime: {
      onInstalled: { addListener: (callback) => installedListeners.push(callback) },
      onStartup: { addListener: (callback) => startupListeners.push(callback) },
      onMessage: { addListener: (callback) => messageListeners.push(callback) },
    },
    sidePanel: {
      setPanelBehavior: async (options) => {
        panelBehaviorCalls.push(options)
      },
    },
    action: {
      onClicked: {
        addListener: () => {
          actionClickListeners += 1
        },
      },
    },
  }

  await importBackgroundModule()

  assert.equal(installedListeners.length, 1)
  assert.equal(startupListeners.length, 1)
  assert.equal(messageListeners.length, 1)
  assert.equal(actionClickListeners, 0)

  installedListeners[0]()
  startupListeners[0]()
  await Promise.resolve()

  assert.deepEqual(panelBehaviorCalls, [
    { openPanelOnActionClick: true },
    { openPanelOnActionClick: true },
  ])
})
