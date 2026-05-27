import { CRAB_COMBO_STATE, CRAB_MINING_CONFIG } from "../config/crabMining.ts";

export function createCrabClickController({ config = CRAB_MINING_CONFIG, now = () => Date.now(), random = () => Math.random() } = {}) {
  let pointRemainder = 0;
  const state = {
    tickWindowStartedAt: getTickWindowStartedAt(now(), config),
    validClicksThisTick: 0,
    comboHits: 0,
    totalHits: 0,
    comboState: CRAB_COMBO_STATE.none,
    burstMeter: 0,
    lastBurstMultiplier: null,
    lastBurstEvent: null
  };

  function syncTickWindow() {
    const currentWindow = getTickWindowStartedAt(now(), config);
    if (currentWindow === state.tickWindowStartedAt) return;

    const completedQuota = state.validClicksThisTick >= config.maxValidClicksPerTick;
    state.tickWindowStartedAt = currentWindow;
    state.validClicksThisTick = 0;
    if (!completedQuota) {
      state.comboHits = 0;
      state.comboState = CRAB_COMBO_STATE.none;
      state.burstMeter = 0;
    }
  }

  function registerClick() {
    syncTickWindow();
    if (state.validClicksThisTick >= config.maxValidClicksPerTick) {
      return { valid: false, state };
    }

    state.validClicksThisTick += 1;
    state.comboHits += 1;
    state.totalHits += 1;
    state.comboState = getComboState(state.comboHits, config);

    const burstTriggered = applyBurstProgress(state, config, now, random);
    const continuousMultiplier = getContinuousMultiplier(state.comboState, config);
    const burstMultiplier = burstTriggered ? state.lastBurstMultiplier : 1;
    const rawGain = continuousMultiplier * burstMultiplier + pointRemainder;
    const pointGain = Math.max(1, Math.floor(rawGain));
    pointRemainder = rawGain - pointGain;

    return {
      valid: true,
      pointGain,
      continuousMultiplier,
      burstTriggered,
      burstMultiplier,
      state
    };
  }

  return {
    state,
    registerClick,
    syncTickWindow
  };
}

function getTickWindowStartedAt(timestamp, config) {
  return Math.floor(timestamp / config.tickMs) * config.tickMs;
}

function getComboState(comboHits, config) {
  const { comboThresholds } = config;
  if (comboHits >= comboThresholds.insane) return CRAB_COMBO_STATE.insane;
  if (comboHits >= comboThresholds.fire) return CRAB_COMBO_STATE.fire;
  if (comboHits >= comboThresholds.hype) return CRAB_COMBO_STATE.hype;
  return CRAB_COMBO_STATE.none;
}

function getContinuousMultiplier(comboState, config) {
  const { multipliers } = config;
  const multiplierByState = {
    [CRAB_COMBO_STATE.none]: multipliers.s1,
    [CRAB_COMBO_STATE.hype]: multipliers.hype,
    [CRAB_COMBO_STATE.fire]: multipliers.fire,
    [CRAB_COMBO_STATE.insane]: multipliers.insane
  };
  return Math.min(multiplierByState[comboState], multipliers.continuousCap);
}

function applyBurstProgress(state, config, now, random) {
  const { burst } = config;
  let burstTriggered = false;
  if (state.comboState === CRAB_COMBO_STATE.fire) {
    state.burstMeter += burst.fireGainPerValidClick;
  }
  if (state.comboState === CRAB_COMBO_STATE.insane) {
    state.burstMeter += burst.insaneGainPerValidClick;
  }
  if (state.burstMeter >= burst.maxMeter) {
    state.lastBurstMultiplier = rollBurstMultiplier(config, random);
    state.lastBurstEvent = {
      at: now(),
      multiplier: state.lastBurstMultiplier
    };
    state.burstMeter = 0;
    burstTriggered = true;
  }
  return burstTriggered;
}

function rollBurstMultiplier(config, random) {
  const { burstMax, burstMin } = config.multipliers;
  return Number((burstMin + random() * (burstMax - burstMin)).toFixed(2));
}
