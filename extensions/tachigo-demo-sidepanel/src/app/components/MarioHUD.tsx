import { useState, useEffect, useRef, useCallback } from 'react';
import { useTranslation } from 'react-i18next'

import type { HudDemoState } from '../../extension/types'
import { useSound } from '../hooks/useSound'
import { hudPanelBackground } from '../theme/backgrounds'

// ─── Types ────────────────────────────────────────────────────
interface FloatItem { id: number; amount: number; offsetX: number }

// ─── Constants ───────────────────────────────────────────────
const CYCLE = 60;           // seconds per passive reward cycle
const MAX_CLICKS_PER_CYCLE = 30;

// ─── Simple Capybara Character (front-facing, clickable) ─────
function ClickableCapybara({
  onClick,
  isIdle,
  isSuccess,
  animState = 'idle',
  size = 180
}: {
  onClick: () => void;
  isIdle: boolean;
  isSuccess: boolean;
  animState?: 'idle' | 'mining' | 'big-mining';
  size?: number;
}) {
  const [isPressed, setIsPressed] = useState(false);

  // Determine animation class
  const getAnimClass = () => {
    if (animState === 'big-mining') return 'capy-big-mining';
    if (animState === 'mining') return 'capy-mining';
    return 'capy-idle';
  };

  return (
    <div
      onClick={onClick}
      onMouseDown={() => setIsPressed(true)}
      onMouseUp={() => setIsPressed(false)}
      onMouseLeave={() => setIsPressed(false)}
      className={getAnimClass()}
      style={{
        cursor: isIdle ? 'default' : 'pointer',
        transform: isPressed && !isIdle ? 'scale(0.95)' : 'scale(1)',
        transition: 'transform 0.1s ease',
        filter: isIdle ? 'grayscale(0.3) opacity(0.7)' : 'none',
        position: 'relative',
      }}
    >
      <svg
        viewBox="0 0 320 230"
        width={size}
        height={Math.round(size * 0.72)}
        style={{ display: 'block' }}
      >
        {/* Body */}
        <ellipse cx="160" cy="225" rx="138" ry="42" fill="#7A5228" />
        <ellipse cx="160" cy="215" rx="100" ry="28" fill="#8B6038" fillOpacity="0.6" />

        {/* Neck */}
        <rect x="118" y="185" width="84" height="50" rx="24" fill="#905A30" />

        {/* Head */}
        <rect x="52" y="82" width="216" height="138" rx="54" fill="#C07A3C" />
        <ellipse cx="160" cy="168" rx="94" ry="54" fill="#D4904E" fillOpacity="0.48" />

        {/* Left ear */}
        <ellipse cx="62" cy="100" rx="30" ry="26" fill="#A86030" />
        <ellipse cx="63" cy="104" rx="17" ry="14" fill="#E09A5A" />

        {/* Right ear */}
        <ellipse cx="258" cy="100" rx="30" ry="26" fill="#A86030" />
        <ellipse cx="257" cy="104" rx="17" ry="14" fill="#E09A5A" />

        {/* Mining helmet */}
        <defs>
          <clipPath id="hatClip">
            <rect x="0" y="0" width="320" height="100" />
          </clipPath>
        </defs>
        <ellipse
          cx="160" cy="100" rx="108" ry="86"
          fill="#F5C842"
          clipPath="url(#hatClip)"
        />
        <ellipse
          cx="146" cy="52" rx="52" ry="26"
          fill="#FFE95A"
          fillOpacity="0.52"
          clipPath="url(#hatClip)"
        />
        {/* Brim */}
        <rect x="47" y="97" width="226" height="15" rx="7.5" fill="#C8A010" />
        <rect x="49" y="97" width="224" height="5" rx="2.5" fill="#F0C820" fillOpacity="0.38" />

        {/* Headlamp */}
        <ellipse cx="216" cy="76" rx="20" ry="14" fill="#FFF8CC" />
        <ellipse cx="216" cy="76" rx="13" ry="9" fill="#FFE820" />
        <circle cx="216" cy="76" r="5.5" fill="white" fillOpacity="0.95" />

        {/* Eyes */}
        {!isSuccess && (
          <>
            {/* Left eye */}
            <ellipse cx="110" cy="136" rx="33" ry="35" fill="white" />
            <ellipse cx="110" cy="148" rx="30" ry="22" fill="rgba(192,120,60,0.09)" />
            <ellipse cx="112" cy="139" rx="23" ry="25" fill="#1C0A00" />
            <ellipse cx="112" cy="139" rx="16" ry="18" fill="#5E3010" />
            <ellipse cx="114" cy="141" rx="10" ry="11" fill="#050100" />
            <ellipse cx="122" cy="128" rx="9.5" ry="8.5" fill="white" />

            {/* Right eye */}
            <ellipse cx="210" cy="136" rx="33" ry="35" fill="white" />
            <ellipse cx="210" cy="148" rx="30" ry="22" fill="rgba(192,120,60,0.09)" />
            <ellipse cx="208" cy="139" rx="23" ry="25" fill="#1C0A00" />
            <ellipse cx="208" cy="139" rx="16" ry="18" fill="#5E3010" />
            <ellipse cx="206" cy="141" rx="10" ry="11" fill="#050100" />
            <ellipse cx="198" cy="128" rx="9.5" ry="8.5" fill="white" />
          </>
        )}

        {/* Success eyes ^.^ */}
        {isSuccess && (
          <>
            <path d="M82,136 Q110,126 136,136" stroke="#1C0A00" strokeWidth="6" fill="none" strokeLinecap="round" />
            <path d="M184,136 Q210,126 238,136" stroke="#1C0A00" strokeWidth="6" fill="none" strokeLinecap="round" />
            {/* Sparkles */}
            <text x="70" y="120" fill="#FFB000" fontSize="20">✦</text>
            <text x="250" y="120" fill="#FFB000" fontSize="20">✦</text>
          </>
        )}

        {/* Eyebrows */}
        <path d="M82,112 Q110,102 136,107" stroke="#8B4A20"
              strokeWidth="4" fill="none" strokeLinecap="round" strokeOpacity="0.62" />
        <path d="M184,107 Q210,102 238,112" stroke="#8B4A20"
              strokeWidth="4" fill="none" strokeLinecap="round" strokeOpacity="0.62" />

        {/* Rosy cheeks */}
        <ellipse cx="78" cy="172" rx="32" ry="21" fill="#FF8888" fillOpacity="0.24" />
        <ellipse cx="242" cy="172" rx="32" ry="21" fill="#FF8888" fillOpacity="0.24" />

        {/* Snout */}
        <rect x="106" y="166" width="108" height="46" rx="23" fill="#9A5A28" />
        <rect x="116" y="173" width="88" height="30" rx="15" fill="#AA6A38" />
        <ellipse cx="160" cy="177" rx="28" ry="7" fill="rgba(255,200,150,0.18)" />
        {/* Nostrils */}
        <ellipse cx="141" cy="184" rx="11" ry="7.5" fill="#6A3618" fillOpacity="0.78" />
        <ellipse cx="179" cy="184" rx="11" ry="7.5" fill="#6A3618" fillOpacity="0.78" />

        {/* Mouth */}
        <path d="M137,207 Q160,220 183,207" stroke="#7A3E18"
              strokeWidth="3.5" fill="none" strokeLinecap="round" />
        {/* Teeth */}
        <rect x="150" y="209" width="8" height="7" rx="2" fill="white" fillOpacity="0.82" />
        <rect x="162" y="209" width="8" height="7" rx="2" fill="white" fillOpacity="0.82" />

        {/* Pickaxe (animated separately) */}
        <g
          id="pickaxe"
          className={`pickaxe-${animState}`}
          style={{
            transformOrigin: '240px 210px',
            transformBox: 'fill-box',
          }}
        >
          {/* Handle */}
          <rect
            x="235"
            y="140"
            width="10"
            height="70"
            rx="5"
            fill="#8B6038"
          />
          <rect
            x="236"
            y="142"
            width="4"
            height="66"
            rx="2"
            fill="#A87448"
            fillOpacity="0.5"
          />

          {/* Pickaxe head */}
          <path
            d="M220,135 L235,140 L235,145 L220,148 Z"
            fill="#7A8B9A"
          />
          <path
            d="M245,140 L260,132 L263,137 L245,145 Z"
            fill="#7A8B9A"
          />
          {/* Metal shine */}
          <path
            d="M223,137 L233,141 L233,143 L223,141 Z"
            fill="#A8C0D0"
            fillOpacity="0.6"
          />
          <path
            d="M247,141 L258,134 L259,136 L247,143 Z"
            fill="#A8C0D0"
            fillOpacity="0.6"
          />
        </g>
      </svg>
    </div>
  );
}

// ─── Spinning Coin (CLAIM button) ────────────────────────────
function SpinningCoin({ onClick, ariaLabel }: { onClick: () => void; ariaLabel: string }) {
  // 8 幀：3 幀正面（延長正面停留）
  const FRAME_HWS   = [1, 6, 14, 20, 20, 20, 14, 6]
  // 圓形輪廓：10 列掃描半寬（依圓方程式計算，max=20）
  const OUTLINE_HWS = [9, 14, 17, 19, 20, 20, 19, 17, 14, 9]
  const [frame, setFrame] = useState(0)

  useEffect(() => {
    const t = setInterval(() => setFrame(f => (f + 1) % 8), 160)
    return () => clearInterval(t)
  }, [])

  const fhw      = FRAME_HWS[frame]
  const maxHw    = 20
  const cx       = 26
  const showFace = fhw >= 14

  return (
    <button
      aria-label={ariaLabel}
      onClick={onClick}
      style={{
        background: 'none',
        border: 'none',
        cursor: 'pointer',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        margin: '12px auto 0',
        padding: '4px 8px',
      }}
    >
      <svg
        width="52"
        height="44"
        viewBox="0 0 52 44"
        shapeRendering="crispEdges"
        style={{ imageRendering: 'pixelated' }}
      >
        <defs>
          {/* 裁切範圍跟隨幣寬，y=2 起始（10列×4px=40px 置中於 44px SVG） */}
          <clipPath id="sc-clip">
            <rect x={cx - fhw} y={2} width={fhw * 2} height={40} />
          </clipPath>
        </defs>
        {/* 掃描列：rect 像素風，y 從 2 起始保持圓形置中 */}
        {OUTLINE_HWS.map((ohw, i) => {
          const hw  = Math.max(1, Math.round(ohw * fhw / maxHw))
          const iHw = showFace ? Math.max(0, Math.round((ohw - 4) * fhw / maxHw)) : 0
          const x   = cx - hw
          const y   = i * 4 + 2
          return (
            <g key={i}>
              {/* 陰影列 */}
              <rect x={x + 1} y={y + 1} width={hw * 2} height={4} fill="#A06800" />
              {/* 幣面列 */}
              <rect x={x} y={y} width={hw * 2} height={4} fill="#FFB000" />
              {/* 內環列 */}
              {showFace && iHw > 0 && (
                <rect x={cx - iHw} y={y} width={iHw * 2} height={4} fill="#E09800" />
              )}
            </g>
          )
        })}
        {/* 高光（左上角第 1 列內） */}
        {showFace && (
          <rect
            x={cx - Math.round(6 * fhw / maxHw)}
            y={6}
            width={Math.max(1, Math.round(6 * fhw / maxHw))}
            height={5}
            fill="#FFE870"
            opacity={0.75}
          />
        )}
        {/* CLAIM 文字（粗體，裁切在幣形內，y=22 為圓心） */}
        {showFace && (
          <text
            x={cx}
            y={22}
            textAnchor="middle"
            dominantBaseline="middle"
            fontSize="4"
            fontFamily="var(--pixel-font-family)"
            fontWeight="bold"
            letterSpacing="0.8"
            fill="#7A4E00"
            clipPath="url(#sc-clip)"
          >
            CLAIM
          </text>
        )}
      </svg>
    </button>
  )
}

// ─── Main HUD Component ───────────────────────────────────────
interface MarioHUDProps {
  state?: HudDemoState
  onStateChange?: (state: HudDemoState) => void
  onNavigate?: (screen: 'claim' | 'coupon') => void
}

export function MarioHUD({ state, onStateChange, onNavigate }: MarioHUDProps) {
  const { t } = useTranslation()
  const { playMiningClick, playRewardComplete, playMaxClicks, playToggleWatch, startBgMusic, stopBgMusic, bridgeStatus } = useSound()
  const [points, setPoints]               = useState(state?.points ?? 0);
  const [totalPoints, setTotalPoints]     = useState(state?.totalPoints ?? 12847);
  const [countdown, setCountdown]         = useState(state?.countdown ?? 60);
  const [isWatching, setIsWatching]       = useState(state?.isWatching ?? true);
  const [floats, setFloats]               = useState<FloatItem[]>([]);
  const [balanceBump, setBalanceBump]     = useState(false);
  const [clickCount, setClickCount]       = useState(state?.clickCount ?? 0);
  const [showSuccess, setShowSuccess]     = useState(false);
  const [capyState, setCapyState]         = useState<'idle' | 'mining' | 'big-mining'>('idle');
  const [bgMusicOn, setBgMusicOn]         = useState(false);

  const floatId = useRef(0);

  // ── Helpers ──────────────────────────────────────────────
  const formatPts = (n: number) =>
    n >= 10000 ? (n / 1000).toFixed(1) + 'K' : n.toLocaleString();

  const triggerBalancePop = useCallback(() => {
    setBalanceBump(true);
    setTimeout(() => setBalanceBump(false), 420);
  }, []);

  const spawnFloat = useCallback((amount: number) => {
    const id = ++floatId.current;
    const offsetX = (Math.random() - 0.5) * 40;
    setFloats(f => [...f, { id, amount, offsetX }]);
    setTimeout(() => setFloats(f => f.filter(x => x.id !== id)), 1600);
  }, []);

  const awardPoints = useCallback(
    (amount: number) => {
      setPoints(p => p + amount);
      setTotalPoints(p => p + amount);
      triggerBalancePop();
      spawnFloat(amount);

      // Success animation
      setShowSuccess(true);
      setTimeout(() => setShowSuccess(false), 600);

      // Trigger mining animation based on amount
      if (amount >= 100) {
        // Big mining for +100 points
        setCapyState('big-mining');
        setTimeout(() => setCapyState('idle'), 500);
      } else {
        // Regular mining for +1 point
        setCapyState('mining');
        setTimeout(() => setCapyState('idle'), 250);
      }
    },
    [triggerBalancePop, spawnFloat]
  );

  // ── Passive countdown (60s cycle, awards +100 when complete) ────
  useEffect(() => {
    if (!isWatching) return;
    const tick = setInterval(() => {
      setCountdown(c => {
        if (c <= 1) {
          awardPoints(100);
          playRewardComplete();
          setClickCount(0);
          return CYCLE;
        }
        return c - 1;
      });
    }, 1000);
    return () => clearInterval(tick);
  }, [isWatching, awardPoints, playRewardComplete]);

  // ── 背景音樂：watching 且 bgMusicOn 時才播放 ─────────────
  useEffect(() => {
    if (isWatching && bgMusicOn) startBgMusic();
    else stopBgMusic();
  }, [isWatching, bgMusicOn, startBgMusic, stopBgMusic]);

  useEffect(() => {
    onStateChange?.({
      points,
      totalPoints,
      countdown,
      isWatching,
      clickCount,
    })
  }, [clickCount, countdown, isWatching, onStateChange, points, totalPoints])

  // ── Click handler (awards +1, 30 clicks per cycle) ───────
  const handleCapybaraClick = useCallback(() => {
    if (!isWatching) return;
    if (clickCount >= MAX_CLICKS_PER_CYCLE) {
      playMaxClicks();
      return;
    }
    playMiningClick();
    awardPoints(1);
    setClickCount(c => c + 1);
  }, [isWatching, clickCount, awardPoints, playMiningClick, playMaxClicks]);

  const progress = (CYCLE - countdown) / CYCLE;

  // ─────────────────────────────────────────────────────────
  return (
    <div
      style={{
        width: 320,
        height: 600,
        background: hudPanelBackground,
        display: 'flex',
        flexDirection: 'column',
        position: 'relative',
        overflow: 'hidden',
        fontFamily: 'var(--pixel-font-family)',
        userSelect: 'none',
      }}
    >
      {/* ── Float layer ── */}
      {floats.map(f => (
        <div
          key={f.id}
          style={{
            position: 'absolute',
            top: '40%',
            left: `calc(50% + ${f.offsetX}px)`,
            zIndex: 50,
            pointerEvents: 'none',
            fontFamily: 'var(--pixel-font-family)',
            fontSize: 16,
            color: '#FFB000',
            textShadow: '0 0 12px rgba(255,176,0,0.9), 0 2px 4px rgba(0,0,0,0.9)',
            animation: 'mario-float 1.5s ease-out forwards',
          }}
        >
          +{f.amount}
        </div>
      ))}

      {/* ════════════════════════════════════════════
          TOP HUD (固定区)
      ════════════════════════════════════════════ */}
      <div
        style={{
          padding: '12px 16px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: '2px solid rgba(145,70,255,0.2)',
        }}
      >
        {/* Left: Status */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <div
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              background: isWatching ? '#9146FF' : '#555555',
              boxShadow: isWatching ? '0 0 8px #9146FF' : 'none',
            }}
          />
          <span
            style={{
              fontSize: 7,
              color: isWatching ? '#9146FF' : '#555555',
              letterSpacing: '0.1em',
            }}
          >
            {isWatching ? t('hud.watching') : t('hud.idle')}
          </span>
        </div>

        {/* Right: Channel */}
        <span
          style={{
            fontSize: 8,
            color: '#9146FF',
            letterSpacing: '0.08em',
          }}
        >
          #jd_onlymusic
        </span>

        {/* Demo toggle */}
        <button
          onClick={() => { playToggleWatch(); setIsWatching(w => !w); }}
          style={{
            padding: '2px 6px',
            borderRadius: 2,
            border: '1px solid rgba(145,70,255,0.3)',
            background: 'transparent',
            color: '#555555',
            fontSize: 6,
            cursor: 'pointer',
            fontFamily: 'var(--pixel-font-family)',
          }}
      >
          SW
        </button>
      </div>

      {bridgeStatus === 'unsupported' && (
        <div
          style={{
            padding: '6px 16px 0',
            fontSize: 6,
            color: '#FFB000',
            letterSpacing: '0.08em',
            textAlign: 'right',
          }}
        >
          {t('hud.tabAudioUnavailable')}
        </div>
      )}


      {/* ════════════════════════════════════════════
          POINTS DISPLAY (最大字)
      ════════════════════════════════════════════ */}
      <div
        style={{
          padding: '24px 16px 16px',
          textAlign: 'center',
        }}
      >
        {/* Current points (HUGE) */}
        <div
          style={{
            fontSize: 52,
            color: '#FFB000',
            letterSpacing: '-0.02em',
            lineHeight: 1,
            fontVariantNumeric: 'tabular-nums',
            textShadow: '0 0 20px rgba(255,176,0,0.5)',
            animation: balanceBump ? 'mario-pts-pop 0.42s ease-out' : 'none',
          }}
        >
          {formatPts(points)}
        </div>

        {/* CLAIM 旋轉金幣按鈕 */}
        {onNavigate && (
          <SpinningCoin ariaLabel={t('claim.title')} onClick={() => onNavigate('claim')} />
        )}

        {/* Total cumulative (small gray) */}
        <div
          style={{
            fontSize: 7,
            color: '#666666',
            letterSpacing: '0.1em',
            marginTop: 8,
          }}
        >
          {t('hud.cumulativeTotal')}
        </div>
        <div
          style={{
            fontSize: 10,
            color: '#555555',
            letterSpacing: '0.05em',
            marginTop: 4,
          }}
        >
          {formatPts(totalPoints)}
        </div>
      </div>

      {/* ════════════════════════════════════════════
          CENTRAL FOCUS (水豚 + 進度條)
      ════════════════════════════════════════════ */}
      <div
        style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 16,
          position: 'relative',
        }}
      >
        {/* Capybara character (30% height, clickable) */}
        <ClickableCapybara
          onClick={handleCapybaraClick}
          isIdle={!isWatching}
          isSuccess={showSuccess}
          animState={capyState}
          size={180}
        />

        {/* Progress bar */}
        <div style={{ width: '80%', maxWidth: 240 }}>
          <div
            style={{
              height: 20,
              background: 'rgba(145,70,255,0.15)',
              border: '2px solid #9146FF',
              borderRadius: 2,
              overflow: 'hidden',
              position: 'relative',
            }}
          >
            <div
              style={{
                height: '100%',
                width: `${progress * 100}%`,
                background: '#9146FF',
                transition: 'width 1s linear',
                boxShadow: '0 0 10px rgba(145,70,255,0.8)',
              }}
            />
          </div>

          {/* Next reward countdown */}
          <div
            style={{
              fontSize: 8,
              color: '#9146FF',
              textAlign: 'center',
              marginTop: 8,
              letterSpacing: '0.08em',
            }}
          >
            {t('hud.nextIn', { count: countdown })}
          </div>
        </div>

        {/* Click quota indicator */}
        <div
          style={{
            fontSize: 7,
            color: clickCount >= MAX_CLICKS_PER_CYCLE ? '#ef4444' : '#555555',
            letterSpacing: '0.08em',
          }}
        >
          {clickCount >= MAX_CLICKS_PER_CYCLE
            ? t('hud.clicksExhausted')
            : t('hud.clicksLeft', { count: MAX_CLICKS_PER_CYCLE - clickCount })}
        </div>

        {onNavigate && (
          <div
            style={{
              width: '100%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              marginTop: 12,
              paddingBottom: 4,
            }}
          >
            <button
              aria-label={t('coupon.title')}
              onClick={() => onNavigate('coupon')}
              style={{
                minWidth: 120,
                border: '1px solid rgba(255,176,0,0.32)',
                background: 'linear-gradient(180deg, rgba(255,176,0,0.18) 0%, rgba(255,176,0,0.06) 100%)',
                color: '#FFB000',
                borderRadius: 999,
                padding: '8px 16px',
                fontSize: 8,
                cursor: 'pointer',
                fontFamily: 'var(--pixel-font-family)',
                letterSpacing: '0.12em',
                boxShadow: '0 0 16px rgba(255,176,0,0.18)',
              }}
            >
              {t('coupon.entry')}
            </button>
          </div>
        )}
      </div>

      {/* ════════════════════════════════════════════
          BGM 獨立控制列（HUD 主內容以外）
      ════════════════════════════════════════════ */}
      <div
        style={{
          borderTop: '1px solid rgba(145,70,255,0.15)',
          padding: '6px 16px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          background: 'rgba(0,0,0,0.3)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
          <span style={{ fontSize: 6, color: '#444', letterSpacing: '0.1em' }}>
            {t('hud.bgmLabel')}
          </span>
          <button
            onClick={() => setBgMusicOn(v => !v)}
            style={{
              padding: '3px 8px',
              borderRadius: 2,
              border: `1px solid ${bgMusicOn ? '#9146FF' : 'rgba(145,70,255,0.2)'}`,
              background: bgMusicOn ? 'rgba(145,70,255,0.15)' : 'transparent',
              color: bgMusicOn ? '#9146FF' : '#444',
              fontSize: 7,
              cursor: 'pointer',
              fontFamily: 'var(--pixel-font-family)',
              letterSpacing: '0.08em',
            }}
          >
            {bgMusicOn ? t('hud.bgmOn') : t('hud.bgmOff')}
          </button>
        </div>
      </div>
    </div>
  );
}
