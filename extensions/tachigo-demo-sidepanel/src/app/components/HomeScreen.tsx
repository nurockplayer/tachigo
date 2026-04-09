import React, { useState, useEffect, useRef, useCallback } from 'react';
import { CapybaraMiner, type CapybaraState } from './CapybaraMiner';
import { CaveSparkles } from './CaveSparkles';
import { FloatingReward, type FloatItem } from './FloatingReward';

type MineButtonState = 'init' | 'ready' | 'cooldown' | 'error';

function formatPoints(n: number): string {
  if (n >= 1000000) return (n / 1000000).toFixed(2) + 'M';
  if (n >= 1000) return (n / 1000).toFixed(2) + 'K';
  return n.toLocaleString();
}

function formatPointsFull(n: number): string {
  return n.toLocaleString();
}

export function HomeScreen() {
  // Balance
  const [availablePoints, setAvailablePoints] = useState(6743);
  const [totalPoints, setTotalPoints] = useState(42891);
  const [balanceBump, setBalanceBump] = useState(false);

  // Mine button
  const [mineState, setMineState] = useState<MineButtonState>('init');
  const [cooldown, setCooldown] = useState(0);
  const cooldownRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Capybara
  const [capyState, setCapyState] = useState<CapybaraState>('loading');
  const capyTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Passive mining
  const [passiveError, setPassiveError] = useState(false);

  // Floating rewards
  const [floatRewards, setFloatRewards] = useState<FloatItem[]>([]);
  const floatIdRef = useRef(0);

  // ── Init sequence ───────────────────────────────────────────
  useEffect(() => {
    const t = setTimeout(() => {
      setMineState('ready');
      setCapyState('idle');
    }, 1200);
    return () => clearTimeout(t);
  }, []);

  // ── Passive mining ticks ─────────────────────────────────────
  useEffect(() => {
    const interval = setInterval(() => {
      const reward = Math.floor(Math.random() * 4) + 1;
      setAvailablePoints((p) => p + reward);
      setTotalPoints((p) => p + reward);
      setBalanceBump(true);
      setTimeout(() => setBalanceBump(false), 450);
    }, 7000);
    return () => clearInterval(interval);
  }, []);

  // ── Cleanup on unmount ───────────────────────────────────────
  useEffect(() => {
    return () => {
      if (cooldownRef.current) clearInterval(cooldownRef.current);
      if (capyTimeoutRef.current) clearTimeout(capyTimeoutRef.current);
    };
  }, []);

  // ── Emit floating reward ─────────────────────────────────────
  const emitFloat = useCallback((amount: number) => {
    const id = ++floatIdRef.current;
    const x = 30 + Math.random() * 40;
    setFloatRewards((p) => [...p, { id, amount, x }]);
    setTimeout(() => {
      setFloatRewards((p) => p.filter((r) => r.id !== id));
    }, 1600);
  }, []);

  // ── Click mine ───────────────────────────────────────────────
  const handleMineClick = useCallback(() => {
    if (mineState !== 'ready') return;

    setMineState('cooldown');
    setCapyState('mining');

    const reward = Math.random() > 0.35 ? 1 : 2;
    const COOLDOWN_SECS = 3.0;
    setCooldown(COOLDOWN_SECS);

    // Award after brief swing
    const awardTimeout = setTimeout(() => {
      setAvailablePoints((p) => p + reward);
      setTotalPoints((p) => p + reward);
      setBalanceBump(true);
      setTimeout(() => setBalanceBump(false), 450);
      emitFloat(reward);
      setCapyState('success');

      const resetCapyTimeout = setTimeout(() => {
        setCapyState('idle');
      }, 700);
      capyTimeoutRef.current = resetCapyTimeout;
    }, 380);
    capyTimeoutRef.current = awardTimeout;

    // Countdown
    let remaining = COOLDOWN_SECS;
    if (cooldownRef.current) clearInterval(cooldownRef.current);
    cooldownRef.current = setInterval(() => {
      remaining = Math.round((remaining - 0.1) * 10) / 10;
      setCooldown(remaining);
      if (remaining <= 0) {
        if (cooldownRef.current) clearInterval(cooldownRef.current);
        setMineState('ready');
        setCooldown(0);
      }
    }, 100);
  }, [mineState, emitFloat]);

  // ── Toggle passive error (demo) ──────────────────────────────
  const handleTogglePassiveError = () => setPassiveError((e) => !e);

  // ── Mine button rendering ────────────────────────────────────
  const renderMineButton = () => {
    const base: React.CSSProperties = {
      width: '100%',
      padding: '14px 0',
      borderRadius: 8,
      border: 'none',
      cursor: 'pointer',
      fontFamily: "'Inter', sans-serif",
      fontWeight: 600,
      fontSize: 15,
      letterSpacing: '0.04em',
      transition: 'opacity 0.15s, transform 0.1s',
      position: 'relative',
      overflow: 'hidden',
    };

    if (mineState === 'init') {
      return (
        <button
          disabled
          style={{
            ...base,
            background: 'rgba(255,255,255,0.04)',
            color: 'rgba(156,163,175,0.5)',
            border: '1px solid rgba(255,255,255,0.06)',
            cursor: 'not-allowed',
          }}
        >
          初始化中…
        </button>
      );
    }

    if (mineState === 'ready') {
      return (
        <button
          className="btn-mine-ready"
          onClick={handleMineClick}
          style={{
            ...base,
            background: 'linear-gradient(180deg, #c8a849 0%, #a8882c 100%)',
            color: '#0d0d18',
            border: '1px solid rgba(245,200,66,0.6)',
          }}
          onMouseDown={(e) => (e.currentTarget.style.transform = 'scale(0.97)')}
          onMouseUp={(e) => (e.currentTarget.style.transform = 'scale(1)')}
          onMouseLeave={(e) => (e.currentTarget.style.transform = 'scale(1)')}
        >
          ⛏ 點擊挖礦
        </button>
      );
    }

    if (mineState === 'cooldown') {
      return (
        <button
          disabled
          style={{
            ...base,
            background: 'rgba(255,255,255,0.04)',
            color: 'rgba(156,163,175,0.6)',
            border: '1px solid rgba(255,255,255,0.08)',
            cursor: 'not-allowed',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 8,
          }}
        >
          {/* Cooldown progress bar */}
          <div
            style={{
              position: 'absolute',
              left: 0,
              bottom: 0,
              height: 3,
              background: 'linear-gradient(90deg, #c8a849, #f5c842)',
              width: `${((3 - cooldown) / 3) * 100}%`,
              transition: 'width 0.1s linear',
              borderRadius: '0 0 8px 8px',
              boxShadow: '0 0 6px rgba(200,168,73,0.5)',
            }}
          />
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" style={{ opacity: 0.6 }}>
            <circle cx="7" cy="7" r="6" stroke="currentColor" strokeWidth="1.5" fill="none" />
            <path d="M7 3.5V7L9.5 9" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
          </svg>
          冷卻中 {cooldown.toFixed(1)}s
        </button>
      );
    }

    if (mineState === 'error') {
      return (
        <button
          disabled
          style={{
            ...base,
            background: 'rgba(239,68,68,0.08)',
            color: '#ef4444',
            border: '1px solid rgba(239,68,68,0.3)',
            cursor: 'not-allowed',
            fontSize: 12,
          }}
        >
          請稍後再試
        </button>
      );
    }
  };

  return (
    <div
      className="screen-enter"
      style={{
        width: '100%',
        height: '100%',
        background: '#0d0d18',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
        fontFamily: "'Inter', system-ui, sans-serif",
        position: 'relative',
      }}
    >
      {/* ════════════════════════════════════════════ */}
      {/* TOP HUD — pinned                            */}
      {/* ════════════════════════════════════════════ */}
      <div
        style={{
          height: 48,
          flexShrink: 0,
          background: 'rgba(10,10,22,0.95)',
          borderBottom: '1px solid rgba(200,168,73,0.18)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '0 14px',
          backdropFilter: 'blur(8px)',
          position: 'relative',
          zIndex: 10,
        }}
      >
        {/* Left: logo + channel */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          {/* Pickaxe icon mark */}
          <svg width="22" height="22" viewBox="0 0 22 22" fill="none" style={{ flexShrink: 0 }}>
            <circle cx="11" cy="11" r="10" fill="rgba(200,168,73,0.12)" stroke="rgba(200,168,73,0.35)" strokeWidth="1" />
            <line x1="5" y1="17" x2="15" y2="7" stroke="#c8a849" strokeWidth="1.8" strokeLinecap="round" />
            <path d="M14,5.5 C15,4.5 17.5,5 18,6.5 C16.5,8 14.5,8 14,7 Z" fill="#f5c842" />
            <path d="M14,7 C13,8 13.5,10 14.5,10.5 C14.5,9 15,8 15.5,7.5 Z" fill="#f5c842" fillOpacity="0.6" />
          </svg>

          <div style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <span
              className="game-pixel"
              style={{ fontSize: 6, color: '#f5c842', letterSpacing: '0.12em', lineHeight: 1 }}
            >
              TACHIGO
            </span>
            <span
              className="game-sans"
              style={{ fontSize: 10, color: 'rgba(156,163,175,0.75)', lineHeight: 1, letterSpacing: '0.02em' }}
            >
              Mining Server
            </span>
          </div>
        </div>

        {/* Right: point capsule */}
        <div
          className="capsule-glow"
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 5,
            padding: '4px 10px',
            borderRadius: 20,
            background: 'rgba(200,168,73,0.1)',
            border: '1px solid rgba(200,168,73,0.3)',
          }}
        >
          {/* Small coin icon */}
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
            <circle cx="6" cy="6" r="5.5" fill="#c8a849" stroke="#f5c842" strokeWidth="0.5" />
            <text x="6" y="9" textAnchor="middle" fontSize="6" fill="#0d0d18" fontWeight="bold" fontFamily="sans-serif">
              ⛏
            </text>
          </svg>
          <span
            className={`game-sans ${balanceBump ? 'balance-bump' : ''}`}
            style={{
              fontSize: 13,
              fontWeight: 700,
              color: '#f5c842',
              letterSpacing: '0.03em',
              minWidth: 42,
              textAlign: 'right',
            }}
          >
            {formatPoints(availablePoints)}
          </span>
          <span className="game-sans" style={{ fontSize: 9, color: 'rgba(200,168,73,0.6)', lineHeight: 1 }}>
            pts
          </span>
        </div>
      </div>

      {/* ════════════════════════════════════════════ */}
      {/* SCROLLABLE CONTENT                          */}
      {/* ════════════════════════════════════════════ */}
      <div
        style={{
          flex: 1,
          overflowY: 'auto',
          overflowX: 'hidden',
          display: 'flex',
          flexDirection: 'column',
          scrollbarWidth: 'none',
        }}
      >
        {/* ── Cave stage ─────────────────────────────── */}
        <div
          style={{
            position: 'relative',
            height: 260,
            flexShrink: 0,
            background:
              'radial-gradient(ellipse 80% 70% at 50% 55%, #181530 0%, #0f0e24 45%, #0d0d18 100%)',
            borderBottom: '1px solid rgba(200,168,73,0.1)',
            overflow: 'hidden',
            display: 'flex',
            alignItems: 'flex-end',
            justifyContent: 'center',
          }}
        >
          {/* Sparkle particles */}
          <CaveSparkles />

          {/* Cave top arch gradient */}
          <div
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              right: 0,
              height: 80,
              background:
                'linear-gradient(180deg, rgba(0,0,0,0.4) 0%, transparent 100%)',
              pointerEvents: 'none',
            }}
          />

          {/* Rock formation bottom — left */}
          <svg
            style={{ position: 'absolute', bottom: 0, left: 0, width: '100%', height: 60 }}
            viewBox="0 0 360 60"
            preserveAspectRatio="none"
          >
            <path
              d="M0,60 L0,42 L20,30 L38,42 L55,18 L70,35 L85,20 L95,38 L108,28 L118,42 L130,30 L145,44 L155,32 L165,48 L175,36 L188,48 L200,38 L214,52 L228,38 L240,50 L255,35 L268,48 L282,32 L298,46 L310,34 L325,48 L340,36 L355,46 L360,42 L360,60 Z"
              fill="#0a0918"
            />
            <path
              d="M0,60 L0,50 L25,38 L45,48 L60,36 L78,46 L92,36 L105,46 L120,38 L136,50 L150,40 L162,52 L176,44 L192,56 L208,44 L222,56 L238,44 L252,54 L270,42 L284,52 L300,42 L316,50 L332,40 L348,52 L360,48 L360,60 Z"
              fill="#080816"
            />
            {/* Ore veins */}
            <path
              d="M60,40 L65,35 L70,38 L72,34 L77,37"
              stroke="#c8a849"
              strokeWidth="1.5"
              fill="none"
              strokeOpacity="0.4"
            />
            <path
              d="M200,42 L206,37 L210,40 L214,35 L218,40"
              stroke="#c8a849"
              strokeWidth="1.5"
              fill="none"
              strokeOpacity="0.35"
            />
            <path
              d="M295,44 L300,38 L304,42"
              stroke="#60a5fa"
              strokeWidth="1.5"
              fill="none"
              strokeOpacity="0.3"
            />
            {/* Ore deposits */}
            <rect x="62" y="42" width="4" height="4" rx="1" fill="#c8a849" fillOpacity="0.5" transform="rotate(15,64,44)" />
            <rect x="202" y="44" width="3" height="3" rx="1" fill="#c8a849" fillOpacity="0.45" transform="rotate(-10,203,45)" />
            <rect x="297" y="46" width="3" height="3" rx="1" fill="#60a5fa" fillOpacity="0.4" transform="rotate(20,298,47)" />
          </svg>

          {/* Cave wall left crevice glow */}
          <div
            style={{
              position: 'absolute',
              left: -20,
              top: '20%',
              width: 80,
              height: 120,
              background:
                'radial-gradient(ellipse, rgba(200,168,73,0.06) 0%, transparent 70%)',
              pointerEvents: 'none',
            }}
          />
          {/* Cave wall right crevice glow */}
          <div
            style={{
              position: 'absolute',
              right: -20,
              top: '35%',
              width: 80,
              height: 100,
              background:
                'radial-gradient(ellipse, rgba(96,165,250,0.04) 0%, transparent 70%)',
              pointerEvents: 'none',
            }}
          />

          {/* Floating rewards overlay */}
          <FloatingReward items={floatRewards} />

          {/* Capybara miner */}
          <div
            style={{
              position: 'relative',
              zIndex: 5,
              marginBottom: 52,
              filter: 'drop-shadow(0 8px 20px rgba(0,0,0,0.6))',
            }}
          >
            <CapybaraMiner state={capyState} size={148} />
          </div>

          {/* Ground glow under capy */}
          <div
            style={{
              position: 'absolute',
              bottom: 48,
              left: '50%',
              transform: 'translateX(-50%)',
              width: 120,
              height: 20,
              background:
                'radial-gradient(ellipse, rgba(200,168,73,0.12) 0%, transparent 70%)',
              pointerEvents: 'none',
              zIndex: 1,
            }}
          />
        </div>

        {/* ── Resource strip ───────────────────────── */}
        <div
          style={{
            flexShrink: 0,
            padding: '14px 14px 10px',
            background: 'rgba(255,255,255,0.015)',
            borderBottom: '1px solid rgba(200,168,73,0.1)',
          }}
        >
          {/* Two stats side by side */}
          <div style={{ display: 'flex', gap: 8, marginBottom: 10 }}>
            {/* Available points */}
            <div
              style={{
                flex: 1,
                background: 'rgba(200,168,73,0.06)',
                border: '1px solid rgba(200,168,73,0.18)',
                borderRadius: 8,
                padding: '10px 12px',
                boxShadow: 'inset 0 1px 0 rgba(255,255,255,0.03)',
              }}
            >
              <div
                className="game-sans"
                style={{
                  fontSize: 9.5,
                  color: 'rgba(200,168,73,0.6)',
                  letterSpacing: '0.06em',
                  marginBottom: 5,
                  textTransform: 'uppercase',
                }}
              >
                ��用點數
              </div>
              <div
                className={`game-sans ${balanceBump ? 'balance-bump' : ''}`}
                style={{
                  fontSize: 17,
                  fontWeight: 700,
                  color: '#f5c842',
                  letterSpacing: '0.02em',
                  textShadow: '0 0 10px rgba(245,200,66,0.3)',
                }}
              >
                {formatPointsFull(availablePoints)}
              </div>
            </div>

            {/* Total points */}
            <div
              style={{
                flex: 1,
                background: 'rgba(255,255,255,0.03)',
                border: '1px solid rgba(255,255,255,0.07)',
                borderRadius: 8,
                padding: '10px 12px',
                boxShadow: 'inset 0 1px 0 rgba(255,255,255,0.03)',
              }}
            >
              <div
                className="game-sans"
                style={{
                  fontSize: 9.5,
                  color: 'rgba(156,163,175,0.55)',
                  letterSpacing: '0.06em',
                  marginBottom: 5,
                  textTransform: 'uppercase',
                }}
              >
                累計點數
              </div>
              <div
                className="game-sans"
                style={{
                  fontSize: 17,
                  fontWeight: 700,
                  color: '#d1d5db',
                  letterSpacing: '0.02em',
                }}
              >
                {formatPointsFull(totalPoints)}
              </div>
            </div>
          </div>

          {/* Passive sync status */}
          {!passiveError ? (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 7,
                padding: '6px 10px',
                borderRadius: 6,
                background: 'rgba(34,197,94,0.06)',
                border: '1px solid rgba(34,197,94,0.15)',
              }}
            >
              <div
                className="passive-dot"
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  background: '#22c55e',
                  flexShrink: 0,
                }}
              />
              <span
                className="game-sans"
                style={{ fontSize: 11, color: 'rgba(134,239,172,0.75)', letterSpacing: '0.02em' }}
              >
                掛台收益同步中
              </span>
              {/* Demo toggle for passive error */}
              <button
                onClick={handleTogglePassiveError}
                style={{
                  marginLeft: 'auto',
                  fontSize: 9,
                  color: 'rgba(156,163,175,0.35)',
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  padding: '2px 4px',
                  borderRadius: 3,
                  fontFamily: 'monospace',
                }}
              >
                [test]
              </button>
            </div>
          ) : (
            <div
              className="slide-down"
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 7,
                padding: '6px 10px',
                borderRadius: 6,
                background: 'rgba(239,68,68,0.07)',
                border: '1px solid rgba(239,68,68,0.2)',
              }}
            >
              <div
                className="passive-dot-error"
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  background: '#ef4444',
                  flexShrink: 0,
                }}
              />
              <span
                className="game-sans"
                style={{
                  fontSize: 10.5,
                  color: 'rgba(248,113,113,0.85)',
                  letterSpacing: '0.01em',
                  lineHeight: 1.4,
                }}
              >
                觀看收益同步中斷，請確認網路連線
              </span>
              <button
                onClick={handleTogglePassiveError}
                style={{
                  marginLeft: 'auto',
                  fontSize: 9,
                  color: 'rgba(156,163,175,0.35)',
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  padding: '2px 4px',
                  borderRadius: 3,
                  fontFamily: 'monospace',
                  flexShrink: 0,
                }}
              >
                [fix]
              </button>
            </div>
          )}
        </div>

        {/* ── Click mine area ───────────────────────── */}
        <div
          style={{
            flexShrink: 0,
            padding: '14px 14px 16px',
          }}
        >
          {/* Annotation label */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              marginBottom: 8,
            }}
          >
            <span
              className="game-pixel"
              style={{
                fontSize: 6,
                color: 'rgba(200,168,73,0.4)',
                letterSpacing: '0.1em',
              }}
            >
              CLICK MINE
            </span>
            {/* 1.5s CSS animation annotation */}
            <span
              style={{
                fontSize: 9,
                color: 'rgba(100,100,140,0.5)',
                fontFamily: 'monospace',
              }}
            >
              1.5s CSS anim ↑
            </span>
          </div>

          {/* Mine button */}
          {renderMineButton()}

          {/* Gain hint below button */}
          {mineState === 'ready' && (
            <div
              className="game-sans"
              style={{
                textAlign: 'center',
                marginTop: 7,
                fontSize: 10,
                color: 'rgba(156,163,175,0.4)',
                letterSpacing: '0.02em',
              }}
            >
              每次獲得 +1 ～ +2 點 · 冷卻 3 秒
            </div>
          )}
          {mineState === 'cooldown' && (
            <div
              className="game-sans"
              style={{
                textAlign: 'center',
                marginTop: 7,
                fontSize: 10,
                color: 'rgba(200,168,73,0.4)',
                letterSpacing: '0.02em',
              }}
            >
              下次挖礦可用時間：{cooldown.toFixed(1)}s
            </div>
          )}
        </div>

        {/* Bottom spacer */}
        <div style={{ flex: 1, minHeight: 12 }} />
      </div>
    </div>
  );
}
