import { useRef, useCallback, useEffect, useState } from 'react';

type SoundType = 'mining-click' | 'reward-complete' | 'max-clicks' | 'toggle-watch'
               | 'start-bg-music' | 'stop-bg-music';
type HitVariant = 'light' | 'normal' | 'critical';

function pickVariant(): HitVariant {
  const r = Math.random();
  if (r < 0.20) return 'light';
  if (r < 0.85) return 'normal';
  return 'critical';
}

// 原創 8-bit 冒險主題（G大調，非任何現有曲子）
const BG_NOTES: [number, number][] = [
  // Phrase 1 — 上行，明亮感
  [392,125],[494,125],[587,125],[784,250],
  [587,125],[494,125],[392,250],[0,125],
  // Phrase 2 — 回應句，略高
  [440,125],[523,125],[659,125],[880,250],
  [659,125],[523,125],[440,375],[0,125],
  // Bridge — 活力衝刺
  [494,100],[587,100],[740,100],[988,200],
  [880,100],[784,100],[740,100],[659,100],
  [784,200],[0,100],[784,200],[0,100],
  // Resolution — 收尾回主調
  [392,125],[494,125],[587,125],[784,125],
  [659,125],[587,125],[494,125],[440,125],
  [392,375],[0,250],
];

async function sendToContentScript(type: SoundType, variant?: HitVariant) {
  const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
  const tabId = tabs[0]?.id;
  if (tabId == null) {
    return false;
  }

  await chrome.tabs.sendMessage(tabId, { type: 'PLAY_SOUND', sound: type, variant });
  return true;
}

function buildCtx(ctxRef: { current: AudioContext | null }): AudioContext {
  if (!ctxRef.current) ctxRef.current = new AudioContext();
  return ctxRef.current;
}

async function getReadyCtx(ctxRef: { current: AudioContext | null }): Promise<AudioContext> {
  const ctx = buildCtx(ctxRef);
  if (ctx.state === 'suspended') {
    await ctx.resume();
  }
  return ctx;
}
// ── 3 Variant 礦石敲擊合成 ───────────────────────────────────
const VARIANT_PARAMS = {
  light:    { pitchMult: 1.15, gainMult: 0.55, duration: 0.06, thumpGain: 0.15, sparkleGain: 0.00, noiseDecay: 0.05 },
  normal:   { pitchMult: 1.00, gainMult: 1.00, duration: 0.08, thumpGain: 0.40, sparkleGain: 0.05, noiseDecay: 0.07 },
  critical: { pitchMult: 1.05, gainMult: 1.30, duration: 0.12, thumpGain: 0.60, sparkleGain: 0.12, noiseDecay: 0.10 },
} as const;

function synthesizeMiningHit(ctx: AudioContext, variant: HitVariant = 'normal') {
  const now   = ctx.currentTime;
  const p     = VARIANT_PARAMS[variant];
  const pitch = (1 + (Math.random() - 0.5) * 0.1) * p.pitchMult;
  const vol   = (0.9 + Math.random() * 0.2) * p.gainMult;

  // Layer 1: Impact transient（三角波）
  const osc1 = ctx.createOscillator(); const gain1 = ctx.createGain();
  osc1.type = 'triangle';
  osc1.frequency.setValueAtTime(900 * pitch, now);
  osc1.frequency.exponentialRampToValueAtTime(400 * pitch, now + 0.04);
  gain1.gain.setValueAtTime(0, now);
  gain1.gain.linearRampToValueAtTime(0.6 * vol, now + 0.002);
  gain1.gain.exponentialRampToValueAtTime(0.001, now + p.duration);
  osc1.connect(gain1); gain1.connect(ctx.destination);
  osc1.start(now); osc1.stop(now + p.duration);

  // Layer 2: Metallic overtone（sawtooth 1800Hz）
  const osc2 = ctx.createOscillator(); const gain2 = ctx.createGain();
  osc2.type = 'sawtooth';
  osc2.frequency.setValueAtTime(1800 * pitch, now);
  gain2.gain.setValueAtTime(0.15 * vol, now + 0.001);
  gain2.gain.exponentialRampToValueAtTime(0.001, now + 0.045);
  osc2.connect(gain2); gain2.connect(ctx.destination);
  osc2.start(now); osc2.stop(now + 0.05);

  // Layer 3: Bandpass noise（debris/friction）
  const bufLen = Math.floor(ctx.sampleRate * 0.12);
  const buf = ctx.createBuffer(1, bufLen, ctx.sampleRate);
  const data = buf.getChannelData(0);
  for (let i = 0; i < bufLen; i++) data[i] = Math.random() * 2 - 1;
  const noise = ctx.createBufferSource(); noise.buffer = buf;
  const bpf = ctx.createBiquadFilter();
  bpf.type = 'bandpass'; bpf.frequency.setValueAtTime(2000, now); bpf.Q.setValueAtTime(1.5, now);
  const gain3 = ctx.createGain();
  gain3.gain.setValueAtTime(0.3 * vol, now);
  gain3.gain.exponentialRampToValueAtTime(0.001, now + p.noiseDecay);
  noise.connect(bpf); bpf.connect(gain3); gain3.connect(ctx.destination);
  noise.start(now); noise.stop(now + p.noiseDecay);

  // Layer 4: Low thump（sine 100Hz）
  const osc4 = ctx.createOscillator(); const gain4 = ctx.createGain();
  osc4.type = 'sine';
  osc4.frequency.setValueAtTime(100 * pitch, now);
  gain4.gain.setValueAtTime(0, now);
  gain4.gain.linearRampToValueAtTime(p.thumpGain * vol, now + 0.001);
  gain4.gain.exponentialRampToValueAtTime(0.001, now + 0.05);
  osc4.connect(gain4); gain4.connect(ctx.destination);
  osc4.start(now); osc4.stop(now + 0.05);

  // Layer 5: Sparkle（light 無，critical 雙 sparkle）
  if (p.sparkleGain > 0) {
    const d1 = 0.025;
    const dur1 = variant === 'critical' ? 0.07 : 0.04;
    const osc5 = ctx.createOscillator(); const gain5 = ctx.createGain();
    osc5.type = 'sine';
    osc5.frequency.setValueAtTime(2800 * pitch, now + d1);
    gain5.gain.setValueAtTime(0, now + d1);
    gain5.gain.linearRampToValueAtTime(p.sparkleGain * vol, now + d1 + 0.003);
    gain5.gain.exponentialRampToValueAtTime(0.001, now + d1 + dur1);
    osc5.connect(gain5); gain5.connect(ctx.destination);
    osc5.start(now + d1); osc5.stop(now + d1 + dur1);

    if (variant === 'critical') {
      const d2 = 0.06;
      const osc6 = ctx.createOscillator(); const gain6 = ctx.createGain();
      osc6.type = 'sine';
      osc6.frequency.setValueAtTime(3500 * pitch, now + d2);
      gain6.gain.setValueAtTime(0, now + d2);
      gain6.gain.linearRampToValueAtTime(0.08 * vol, now + d2 + 0.003);
      gain6.gain.exponentialRampToValueAtTime(0.001, now + d2 + 0.05);
      osc6.connect(gain6); gain6.connect(ctx.destination);
      osc6.start(now + d2); osc6.stop(now + d2 + 0.05);
    }
  }
}

export function useSound() {
  const ctxRef     = useRef<AudioContext | null>(null);
  const bgTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const bgStepRef  = useRef(0);
  const [bridgeStatus, setBridgeStatus] = useState<'ready' | 'unsupported'>('ready');

  const sendBridgeSound = useCallback(async (type: SoundType, variant?: HitVariant) => {
    try {
      await sendToContentScript(type, variant);
      setBridgeStatus('ready');
      return true;
    } catch (error) {
      console.warn('Tab audio bridge unavailable for the current tab.', error);
      setBridgeStatus('unsupported');
      return false;
    }
  }, []);

  // ── 礦石敲擊（3 variant 隨機）────────────────────────────
  const playMiningClick = useCallback(() => {
    const variant = pickVariant();
    void (async () => {
      if (await sendBridgeSound('mining-click', variant)) return;
      synthesizeMiningHit(await getReadyCtx(ctxRef), variant);
    })();
  }, [sendBridgeSound]);

  // ── FF 勝利號角 ────────────────────────────────────────────
  const playRewardComplete = useCallback(() => {
    void (async () => {
      if (await sendBridgeSound('reward-complete')) return;
      const ctx = await getReadyCtx(ctxRef);
      const notes: [number, number, number][] = [
        [440, 0.00, 0.10],[440, 0.13, 0.10],[440, 0.26, 0.10],
        [349, 0.39, 0.18],[523, 0.59, 0.06],[440, 0.67, 0.28],
        [349, 0.97, 0.18],[523, 1.17, 0.06],[440, 1.25, 0.55],
      ];
      notes.forEach(([freq, offset, dur]) => {
        const osc  = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.type = 'square';
        osc.frequency.setValueAtTime(freq, ctx.currentTime + offset);
        gain.gain.setValueAtTime(0.18, ctx.currentTime + offset);
        gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + offset + dur);
        osc.connect(gain); gain.connect(ctx.destination);
        osc.start(ctx.currentTime + offset);
        osc.stop(ctx.currentTime + offset + dur + 0.01);
      });
    })();
  }, [sendBridgeSound]);

  // ── 雙聲答錯蜂鳴器 ────────────────────────────────────────
  const playMaxClicks = useCallback(() => {
    void (async () => {
      if (await sendBridgeSound('max-clicks')) return;
      const ctx = await getReadyCtx(ctxRef);
      [[0, 0.18, 180], [0.23, 0.37, 160]].forEach(([start, end, freq]) => {
        const osc  = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.type = 'sawtooth';
        osc.frequency.setValueAtTime(freq, ctx.currentTime + start);
        gain.gain.setValueAtTime(0.2, ctx.currentTime + start);
        gain.gain.setValueAtTime(0.2, ctx.currentTime + end - 0.02);
        gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + end);
        osc.connect(gain); gain.connect(ctx.destination);
        osc.start(ctx.currentTime + start);
        osc.stop(ctx.currentTime + end);
      });
    })();
  }, [sendBridgeSound]);

  // ── SW 切換音效 ────────────────────────────────────────────
  const playToggleWatch = useCallback(() => {
    void (async () => {
      if (await sendBridgeSound('toggle-watch')) return;
      const ctx  = await getReadyCtx(ctxRef);
      const osc  = ctx.createOscillator();
      const gain = ctx.createGain();
      osc.type = 'sine';
      osc.frequency.setValueAtTime(550, ctx.currentTime);
      gain.gain.setValueAtTime(0.1, ctx.currentTime);
      gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.06);
      osc.connect(gain); gain.connect(ctx.destination);
      osc.start(ctx.currentTime); osc.stop(ctx.currentTime + 0.06);
    })();
  }, [sendBridgeSound]);

  // ── 背景音樂（原創 8-bit 冒險主題）──────────────────────
  const startBgMusic = useCallback(() => {
    if (bgTimerRef.current !== null) return;
    void (async () => {
      if (await sendBridgeSound('start-bg-music')) return;
      const ctx = await getReadyCtx(ctxRef);

      const playStep = () => {
        const [freq, durMs] = BG_NOTES[bgStepRef.current % BG_NOTES.length];
        bgStepRef.current++;

        if (freq > 0) {
          const osc  = ctx.createOscillator();
          const gain = ctx.createGain();
          osc.type = 'square';
          osc.frequency.setValueAtTime(freq, ctx.currentTime);
          gain.gain.setValueAtTime(0.036, ctx.currentTime);
          gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + (durMs / 1000) * 0.85);
          osc.connect(gain); gain.connect(ctx.destination);
          osc.start(ctx.currentTime); osc.stop(ctx.currentTime + durMs / 1000);
        }

        bgTimerRef.current = setTimeout(playStep, durMs);
      };

      playStep();
    })();
  }, [sendBridgeSound]);

  const stopBgMusic = useCallback(() => {
    if (bgTimerRef.current !== null) {
      clearTimeout(bgTimerRef.current);
      bgTimerRef.current = null;
    }
    void sendBridgeSound('stop-bg-music');
  }, [sendBridgeSound]);

  useEffect(() => () => stopBgMusic(), [stopBgMusic]);

  return { playMiningClick, playRewardComplete, playMaxClicks, playToggleWatch, startBgMusic, stopBgMusic, bridgeStatus };
}
