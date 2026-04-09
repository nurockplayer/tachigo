import { loadDemoState, saveDemoState } from './storage'
import { defaultDemoState } from './types'

describe('demo storage', () => {
  it('returns english defaults on first launch', async () => {
    await expect(loadDemoState()).resolves.toEqual(defaultDemoState)
  })

  it('persists the selected language and hud state in localStorage fallback', async () => {
    await saveDemoState({
      screen: 'hud',
      language: 'zh-TW',
      hud: {
        points: 120,
        totalPoints: 13000,
        countdown: 18,
        isWatching: true,
        clickCount: 7,
      },
    })

    await expect(loadDemoState()).resolves.toEqual({
      screen: 'hud',
      language: 'zh-TW',
      hud: {
        points: 120,
        totalPoints: 13000,
        countdown: 18,
        isWatching: true,
        clickCount: 7,
      },
    })
  })
})
