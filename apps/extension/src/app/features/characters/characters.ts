export const CHARACTER_IDS = ['crab', 'dolphin', 'turtle', 'whale', 'capybara'] as const

export type CharacterId = (typeof CHARACTER_IDS)[number]
export type CharacterStage = 1 | 2 | 3

export type CharacterDefinition = {
  id: CharacterId
  displayName: string
  unlockCost: number | null
  stages: CharacterStage[]
  evolutionXpThresholds: {
    stage2: number
    stage3: number
  }
  devOnly: boolean
}

export type CharacterBuffContext = {
  effectiveClicksInWindow?: number
  chatCount?: number
  continuousWatchSeconds?: number
  stage?: CharacterStage
}

const DEFAULT_THRESHOLDS = {
  stage2: 1000,
  stage3: 10000,
} as const

export const CHARACTER_DEFINITIONS: Record<CharacterId, CharacterDefinition> = {
  crab: {
    id: 'crab',
    displayName: 'Crab',
    unlockCost: 0,
    stages: [1, 2, 3],
    evolutionXpThresholds: DEFAULT_THRESHOLDS,
    devOnly: false,
  },
  dolphin: {
    id: 'dolphin',
    displayName: 'Dolphin',
    unlockCost: 50,
    stages: [1, 2, 3],
    evolutionXpThresholds: DEFAULT_THRESHOLDS,
    devOnly: false,
  },
  turtle: {
    id: 'turtle',
    displayName: 'Turtle',
    unlockCost: 1500,
    stages: [1, 2, 3],
    evolutionXpThresholds: DEFAULT_THRESHOLDS,
    devOnly: false,
  },
  whale: {
    id: 'whale',
    displayName: 'Whale',
    unlockCost: null,
    stages: [1, 2, 3],
    evolutionXpThresholds: DEFAULT_THRESHOLDS,
    devOnly: false,
  },
  capybara: {
    id: 'capybara',
    displayName: 'Capybara',
    unlockCost: 0,
    stages: [1, 2, 3],
    evolutionXpThresholds: DEFAULT_THRESHOLDS,
    devOnly: true,
  },
}

export function calculateFamiliarityMultiplier(
  cumulativeWatchSeconds: number,
  daysSinceLastWatch = 0,
): number {
  const watchSeconds = Math.max(0, cumulativeWatchSeconds)
  const watchHours = watchSeconds / 3600

  const multiplier = (() => {
    if (watchHours >= 5) {
      return 1
    }

    if (watchHours >= 1) {
      return 0.5 + ((watchHours - 1) / 4) * 0.5
    }

    return 0.1 + watchHours * 0.4
  })()

  const decayDays = Math.max(0, daysSinceLastWatch - 30)
  const decayedMultiplier = multiplier - decayDays * 0.01

  return roundToHundredths(clamp(decayedMultiplier, 0.1, 1))
}

export function calculateCharacterBuff(
  characterId: CharacterId,
  context: CharacterBuffContext,
): number {
  const stage = context.stage ?? 1
  if (stage !== 1) {
    return 1
  }

  switch (characterId) {
    case 'crab':
      return (context.effectiveClicksInWindow ?? 0) >= 10 ? 1.5 : 1
    case 'dolphin': {
      const chatCount = context.chatCount ?? 0
      if (chatCount >= 3) return 1.8
      if (chatCount === 2) return 1.6
      if (chatCount === 1) return 1.4
      return 1
    }
    case 'turtle': {
      const continuousWatchSeconds = context.continuousWatchSeconds ?? 0
      if (continuousWatchSeconds >= 90 * 60) return 1.35
      if (continuousWatchSeconds >= 60 * 60) return 1.2
      if (continuousWatchSeconds >= 30 * 60) return 1.1
      return 1
    }
    case 'whale':
    case 'capybara':
      return 1
  }
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}

function roundToHundredths(value: number): number {
  return Math.round(value * 100) / 100
}
