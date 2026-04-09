import { loadDemoState, saveDemoState } from './storage'
import { defaultDemoState } from './types'

describe('demo storage', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  it('returns english defaults on first launch', async () => {
    await expect(loadDemoState()).resolves.toEqual(defaultDemoState)
  })

  it('persists the selected language and hud state in localStorage fallback', async () => {
    await saveDemoState({
      screen: 'loading',
      language: 'zh-TW',
    })

    await expect(loadDemoState()).resolves.toEqual({
      screen: 'loading',
      language: 'zh-TW',
    })
  })
})
