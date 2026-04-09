import React, { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next'

// ─── Cute front-facing capybara (redesigned from scratch) ─────
function CuteCapybaraLogin() {
  return (
    <svg
      viewBox="0 0 320 230"
      width="320"
      height="230"
      style={{ display: 'block' }}
    >
      {/* ── Body (barely peeking, gets cropped at bottom) ── */}
      <ellipse cx="160" cy="225" rx="138" ry="42" fill="#7A5228" />
      <ellipse cx="160" cy="215" rx="100" ry="28" fill="#8B6038" fillOpacity="0.6" />

      {/* ── Neck ── */}
      <rect x="118" y="185" width="84" height="50" rx="24" fill="#905A30" />

      {/* ── Head (wider than tall — capybara head shape) ── */}
      <rect x="52" y="82" width="216" height="138" rx="54" fill="#C07A3C" />

      {/* ── Face center lighter patch ── */}
      <ellipse cx="160" cy="168" rx="94" ry="54" fill="#D4904E" fillOpacity="0.48" />

      {/* ── Left ear ── */}
      <ellipse cx="62" cy="100" rx="30" ry="26" fill="#A86030" />
      <ellipse cx="63" cy="104" rx="17" ry="14" fill="#E09A5A" />

      {/* ── Right ear ── */}
      <ellipse cx="258" cy="100" rx="30" ry="26" fill="#A86030" />
      <ellipse cx="257" cy="104" rx="17" ry="14" fill="#E09A5A" />

      {/* ════ GOLD MINING HELMET ════ */}
      {/* Dome — clipped to above brim */}
      <defs>
        <clipPath id="hatClipLogin">
          <rect x="0" y="0" width="320" height="100" />
        </clipPath>
      </defs>
      <ellipse
        cx="160" cy="100" rx="108" ry="86"
        fill="#F5C842"
        clipPath="url(#hatClipLogin)"
      />
      {/* Dome shine */}
      <ellipse
        cx="146" cy="52" rx="52" ry="26"
        fill="#FFE95A"
        fillOpacity="0.52"
        clipPath="url(#hatClipLogin)"
      />
      {/* Brim */}
      <rect x="47" y="97" width="226" height="15" rx="7.5" fill="#C8A010" />
      <rect x="49" y="97" width="224" height="5" rx="2.5" fill="#F0C820" fillOpacity="0.38" />

      {/* ── Headlamp ── */}
      <ellipse cx="216" cy="76" rx="20" ry="14" fill="#FFF8CC" />
      <ellipse cx="216" cy="76" rx="13" ry="9" fill="#FFE820" />
      <circle cx="216" cy="76" r="5.5" fill="white" fillOpacity="0.95" />
      {/* Lamp rays */}
      <line x1="232" y1="69" x2="252" y2="56" stroke="#FFF5AA" strokeWidth="2.5"
            strokeOpacity="0.42" strokeLinecap="round" />
      <line x1="234" y1="76" x2="256" y2="71" stroke="#FFF5AA" strokeWidth="2"
            strokeOpacity="0.28" strokeLinecap="round" />
      <line x1="232" y1="83" x2="250" y2="88" stroke="#FFF5AA" strokeWidth="1.5"
            strokeOpacity="0.18" strokeLinecap="round" />

      {/* ════ EYES — the key to cute ════ */}

      {/* Left eye group with blink animation */}
      <g className="capy-eye-blink" style={{ transformOrigin: '110px 136px' }}>
        {/* Left eye — white sclera (large, round) */}
        <ellipse cx="110" cy="136" rx="33" ry="35" fill="white" />
        {/* Subtle under-eye warmth */}
        <ellipse cx="110" cy="148" rx="30" ry="22" fill="rgba(192,120,60,0.09)" />
        {/* Iris (dark ring) */}
        <ellipse cx="112" cy="139" rx="23" ry="25" fill="#1C0A00" />
        {/* Iris color */}
        <ellipse cx="112" cy="139" rx="16" ry="18" fill="#5E3010" />
        {/* Pupil */}
        <ellipse cx="114" cy="141" rx="10" ry="11" fill="#050100" />
        {/* Main sparkle highlight */}
        <ellipse cx="122" cy="128" rx="9.5" ry="8.5" fill="white" />
        {/* Small secondary highlight */}
        <ellipse cx="113" cy="124" rx="4" ry="3.5" fill="white" fillOpacity="0.68" />
      </g>

      {/* Right eye group with blink animation */}
      <g className="capy-eye-blink" style={{ transformOrigin: '210px 136px' }}>
        {/* Right eye — white sclera */}
        <ellipse cx="210" cy="136" rx="33" ry="35" fill="white" />
        <ellipse cx="210" cy="148" rx="30" ry="22" fill="rgba(192,120,60,0.09)" />
        {/* Iris */}
        <ellipse cx="208" cy="139" rx="23" ry="25" fill="#1C0A00" />
        <ellipse cx="208" cy="139" rx="16" ry="18" fill="#5E3010" />
        {/* Pupil */}
        <ellipse cx="206" cy="141" rx="10" ry="11" fill="#050100" />
        {/* Main sparkle */}
        <ellipse cx="198" cy="128" rx="9.5" ry="8.5" fill="white" />
        <ellipse cx="207" cy="124" rx="4" ry="3.5" fill="white" fillOpacity="0.68" />
      </g>

      {/* ── Gentle eyebrows (hopeful / welcoming expression) ── */}
      <path d="M82,112 Q110,102 136,107" stroke="#8B4A20"
            strokeWidth="4" fill="none" strokeLinecap="round" strokeOpacity="0.62" />
      <path d="M184,107 Q210,102 238,112" stroke="#8B4A20"
            strokeWidth="4" fill="none" strokeLinecap="round" strokeOpacity="0.62" />

      {/* ── Rosy cheeks (essential for cute!) ── */}
      <ellipse cx="78" cy="172" rx="32" ry="21" fill="#FF8888" fillOpacity="0.24" />
      <ellipse cx="242" cy="172" rx="32" ry="21" fill="#FF8888" fillOpacity="0.24" />

      {/* ── Snout — wide & flat (capybara's signature look, done cute) ── */}
      <rect x="106" y="166" width="108" height="46" rx="23" fill="#9A5A28" />
      <rect x="116" y="173" width="88" height="30" rx="15" fill="#AA6A38" />
      {/* Nose shine */}
      <ellipse cx="160" cy="177" rx="28" ry="7" fill="rgba(255,200,150,0.18)" />
      {/* Nostrils (wide oval — capybara characteristic) */}
      <ellipse cx="141" cy="184" rx="11" ry="7.5" fill="#6A3618" fillOpacity="0.78" />
      <ellipse cx="179" cy="184" rx="11" ry="7.5" fill="#6A3618" fillOpacity="0.78" />

      {/* ── Mouth — friendly smile ── */}
      <path d="M137,207 Q160,220 183,207" stroke="#7A3E18"
            strokeWidth="3.5" fill="none" strokeLinecap="round" />
      {/* Two little front teeth peaking (capybara style, very cute) */}
      <rect x="150" y="209" width="8" height="7" rx="2" fill="white" fillOpacity="0.82" />
      <rect x="162" y="209" width="8" height="7" rx="2" fill="white" fillOpacity="0.82" />

      {/* ── Ground shadow ── */}
      <ellipse cx="160" cy="228" rx="120" ry="10" fill="rgba(0,0,0,0.25)" />
    </svg>
  );
}

// ─── Login Screen (Mario Design Philosophy) ──────────────────
export function LoginScreen({ onLogin }: { onLogin?: () => void }) {
  const { t } = useTranslation()
  const [username, setUsername]   = useState('');
  const [pass, setPass]           = useState('');
  const [error, setError]         = useState(false);
  const [loading, setLoading]     = useState(false);
  const [focusedUser, setFocusedUser] = useState(false);
  const [focusedPass, setFocusedPass] = useState(false);
  const unlockTimerRef = useRef<number | null>(null)

  const hasInput = username.length > 0 && pass.length > 0;

  useEffect(() => {
    return () => {
      if (unlockTimerRef.current !== null) {
        window.clearTimeout(unlockTimerRef.current)
      }
    }
  }, [])

  const handleUnlock = () => {
    if (loading) {
      return;
    }

    if (!hasInput) {
      setError(true);
      return;
    }

    if (unlockTimerRef.current !== null) {
      window.clearTimeout(unlockTimerRef.current)
    }

    setLoading(true);
    setError(false);
    unlockTimerRef.current = window.setTimeout(() => {
      setLoading(false);
      unlockTimerRef.current = null
      onLogin?.();
    }, 1300);
  };

  const handleKey = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleUnlock();
    }
  };

  return (
    <div
      style={{
        width: 320,
        height: 600,
        background: '#0b0b13',
        display: 'flex',
        flexDirection: 'column',
        position: 'relative',
        overflow: 'hidden',
        fontFamily: 'var(--pixel-font-family)',
        userSelect: 'none',
      }}
    >
      {/* ════════════════════════════════
          TOP SPACER
      ════════════════════════════════ */}
      <div style={{ height: 44, flexShrink: 0 }} />

      {/* ════════════════════════════════
          1. LOGO / BRAND  (元素 1)
      ════════════════════════════════ */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          flexShrink: 0,
        }}
      >
        {/* Wordmark */}
        <div
          style={{
            fontFamily: 'var(--pixel-font-family)',
            fontSize: 28,
            color: '#9146FF',
            letterSpacing: '0.02em',
            lineHeight: 1,
          }}
        >
          TACHIGO
        </div>

        {/* Tagline */}
        <div
          style={{
            fontFamily: 'var(--pixel-font-family)',
            fontSize: 6.5,
            color: '#FFB000',
            letterSpacing: '0.18em',
            marginTop: 16,
          }}
        >
          MINE · WATCH · EARN
        </div>
      </div>

      {/* ════════════════════════════════
          FORM SPACER
      ════════════════════════════════ */}
      <div style={{ height: 56, flexShrink: 0 }} />

      {/* ════════════════════════════════
          2. INPUT  (元素 2)
          3. CTA    (元素 3)
          4. LINK   (元素 4)
      ════════════════════════════════ */}
      <div style={{ padding: '0 26px', flexShrink: 0 }}>

        {/* ── Username input ── */}
        <div style={{ marginBottom: 14 }}>
          <input
            type="text"
            value={username}
            onChange={e => { setUsername(e.target.value); setError(false); }}
            onKeyDown={handleKey}
            onFocus={() => setFocusedUser(true)}
            onBlur={() => setFocusedUser(false)}
            placeholder={t('login.usernamePlaceholder')}
            autoComplete="username"
            style={{
              width: '100%',
              padding: '15px 17px',
              background: 'rgba(255,255,255,0.05)',
              border: error
                ? '1.5px solid rgba(239,68,68,0.7)'
                : focusedUser
                ? '1.5px solid rgba(145,70,255,0.55)'
                : '1.5px solid rgba(255,255,255,0.1)',
              borderRadius: 11,
              color: 'white',
              fontSize: 9,
              outline: 'none',
              fontFamily: 'var(--pixel-font-family)',
              boxSizing: 'border-box',
              transition: 'border-color 0.18s',
            }}
          />
        </div>

        {/* ── Password input ── */}
        <div>
          <input
            type="password"
            value={pass}
            onChange={e => { setPass(e.target.value); setError(false); }}
            onKeyDown={handleKey}
            onFocus={() => setFocusedPass(true)}
            onBlur={() => setFocusedPass(false)}
            placeholder={t('login.passwordPlaceholder')}
            autoComplete="current-password"
            style={{
              width: '100%',
              padding: '15px 17px',
              background: 'rgba(255,255,255,0.05)',
              border: error
                ? '1.5px solid rgba(239,68,68,0.7)'
                : focusedPass
                ? '1.5px solid rgba(145,70,255,0.55)'
                : '1.5px solid rgba(255,255,255,0.1)',
              borderRadius: 11,
              color: 'white',
              fontSize: 9,
              outline: 'none',
              fontFamily: 'var(--pixel-font-family)',
              boxSizing: 'border-box',
              transition: 'border-color 0.18s',
            }}
          />
        </div>

        {/* Error message */}
        {error && (
          <div
            style={{
              fontFamily: 'var(--pixel-font-family)',
              fontSize: 7,
              color: '#ef4444',
              marginTop: 6,
              paddingLeft: 4,
              letterSpacing: '0.08em',
            }}
          >
            {t('login.required')}
          </div>
        )}

        {/* ── CTA Button ── */}
        <button
          onClick={handleUnlock}
          disabled={loading}
          style={{
            width: '100%',
            marginTop: error ? 10 : 12,
            padding: '16px 0',
            borderRadius: 11,
            border: 'none',
            background: hasInput
              ? '#9146FF'
              : 'rgba(255,255,255,0.07)',
            color: hasInput ? '#ffffff' : 'rgba(107,114,128,0.5)',
            fontFamily: 'var(--pixel-font-family)',
            fontSize: 10,
            letterSpacing: '0.1em',
            cursor: loading ? 'wait' : 'pointer',
            boxShadow: hasInput
              ? '0 0 22px rgba(145,70,255,0.4), 0 2px 8px rgba(0,0,0,0.4)'
              : 'none',
            transition: 'all 0.2s ease',
            lineHeight: 1,
          }}
          onMouseDown={e => {
            if (hasInput) e.currentTarget.style.transform = 'scale(0.98)';
          }}
          onMouseUp={e => (e.currentTarget.style.transform = '')}
          onMouseLeave={e => (e.currentTarget.style.transform = '')}
        >
          {loading ? t('login.buttonLoading') : t('login.button')}
        </button>

        {/* ── Forgot / Help link ── */}
        <div style={{ textAlign: 'center', marginTop: 22 }}>
          <a
            href="#"
            onClick={e => e.preventDefault()}
            style={{
              fontFamily: 'var(--pixel-font-family)',
              fontSize: 7,
              color: '#9146FF',
              textDecoration: 'none',
              letterSpacing: '0.08em',
              transition: 'opacity 0.15s',
              opacity: 0.7,
            }}
            onMouseEnter={e => (e.currentTarget.style.opacity = '1')}
            onMouseLeave={e => (e.currentTarget.style.opacity = '0.7')}
          >
            {t('login.forgotPassword')}
          </a>
        </div>
      </div>

      {/* ════════════════════════════════
          FLEX SPACER — pushes capybara to bottom
      ════════════════════════════════ */}
      <div style={{ flex: 1 }} />

      {/* ════════════════════════════════
          CUTE CAPYBARA — peeking from bottom
          (follows MetaMask fox philosophy)
      ════════════════════════════════ */}
      <CuteCapybaraLogin />
    </div>
  );
}
