import type { HudDemoState } from '../extension/types'

export function claimFromHudState(previousHudState: HudDemoState, cpcAmount: number) {
  const claimable = Math.max(0, Math.min(cpcAmount, previousHudState.points))

  return {
    claimable,
    nextHudState: {
      ...previousHudState,
      points: previousHudState.points - claimable,
    },
  }
}
