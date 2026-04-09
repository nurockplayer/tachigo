import React, { useState, useEffect } from 'react';

export type CapybaraState = 'loading' | 'idle' | 'mining' | 'success' | 'error';

interface Props {
  state: CapybaraState;
  size?: number;
}

export function CapybaraMiner({ state, size = 150 }: Props) {
  const [pickaxeAngle, setPickaxeAngle] = useState(-15);

  useEffect(() => {
    if (state !== 'mining') {
      setPickaxeAngle(-15);
      return;
    }
    let angle = -38;
    let direction = 1;
    const tick = setInterval(() => {
      angle += direction * 6;
      if (angle >= 28) direction = -1;
      if (angle <= -38) direction = 1;
      setPickaxeAngle(angle);
    }, 28);
    return () => clearInterval(tick);
  }, [state]);

  const wrapperStyle: React.CSSProperties = {
    width: size,
    height: Math.round(size * 0.94),
    position: 'relative',
    display: 'inline-block',
  };

  // Aspect is viewBox 168×158 → 0.94 ratio
  const scale = size / 168;
  const svgH = Math.round(158 * scale);

  return (
    <div className={`capy-wrapper capy-${state}`} style={wrapperStyle}>
      <svg
        viewBox="0 0 168 158"
        width={size}
        height={svgH}
        style={{ overflow: 'visible', display: 'block' }}
      >
        {/* ── Ground shadow ──────────────────────── */}
        <ellipse cx="82" cy="150" rx="62" ry="7" fill="#000" fillOpacity="0.32" />

        {/* ── Loading: supply pack on back ─────────── */}
        {state === 'loading' && (
          <g>
            <rect x="22" y="80" width="32" height="26" rx="6" fill="#2d1e0c" />
            <rect x="28" y="75" width="20" height="12" rx="4" fill="#3d2a10" />
            <path d="M28,82 Q18,92 20,104" stroke="#1e1208" strokeWidth="3" fill="none" strokeLinecap="round" />
            <path d="M50,82 Q60,92 58,104" stroke="#1e1208" strokeWidth="2" fill="none" strokeLinecap="round" />
            {/* Gems/supplies visible in pack */}
            <circle cx="30" cy="88" r="3" fill="#c8a849" fillOpacity="0.7" />
            <circle cx="38" cy="86" r="2.5" fill="#60a5fa" fillOpacity="0.6" />
            <circle cx="44" cy="90" r="2" fill="#c8a849" fillOpacity="0.5" />
          </g>
        )}

        {/* ── Far back legs ────────────────────────── */}
        <rect x="28" y="126" width="14" height="24" rx="6" fill="#6E4A25" />
        <rect x="46" y="128" width="12" height="22" rx="5" fill="#6E4A25" />
        {/* Feet */}
        <ellipse cx="35" cy="149" rx="9" ry="5" fill="#5C3D1C" />
        <ellipse cx="52" cy="149" rx="8" ry="4.5" fill="#5C3D1C" />

        {/* ── Main body ────────────────────────────── */}
        <ellipse cx="79" cy="103" rx="54" ry="36" fill="#8B6340" />
        {/* Body fur shading */}
        <ellipse cx="79" cy="111" rx="40" ry="23" fill="#A07848" fillOpacity="0.38" />
        {/* Back-spine darker streak */}
        <path d="M28,90 Q55,78 80,75 Q108,73 130,80" stroke="#6E4A25" strokeWidth="4" fill="none" strokeLinecap="round" strokeOpacity="0.35" />

        {/* ── Tail ─────────────────────────────────── */}
        <ellipse cx="25" cy="96" rx="10" ry="7" fill="#7A5530" transform="rotate(-22,25,96)" />
        <ellipse cx="23" cy="94" rx="6" ry="4" fill="#6A4520" transform="rotate(-22,23,94)" />

        {/* ── Neck bridge ──────────────────────────── */}
        <ellipse cx="105" cy="90" rx="17" ry="14" fill="#8B6340" />

        {/* ── Head ─────────────────────────────────── */}
        <ellipse cx="122" cy="73" rx="32" ry="27" fill="#8B6340" />
        {/* Head shading on underside */}
        <ellipse cx="122" cy="84" rx="22" ry="12" fill="#7A5530" fillOpacity="0.3" />

        {/* ── Ear ──────────────────────────────────── */}
        <ellipse cx="112" cy="47" rx="12" ry="9" fill="#8B6340" />
        <ellipse cx="112" cy="48" rx="7" ry="5.5" fill="#C49A6C" />

        {/* ── Hard hat ──────────────────────────────── */}
        {/* Helmet dome */}
        <path
          d="M91,62 C89,43 106,37 122,38 C138,38 149,45 148,60 L144,58 C142,48 134,43 122,43 C110,43 96,49 94,62 Z"
          fill="#F5C842"
        />
        {/* Helmet shadow/depth */}
        <path
          d="M95,62 C93,49 108,43 122,43 C136,43 146,49 145,59 L143,58 C141,50 133,46 122,46 C111,46 99,51 97,62 Z"
          fill="#DAA318"
          fillOpacity="0.45"
        />
        {/* Brim */}
        <rect x="89" y="59" width="60" height="7" rx="3.5" fill="#C8A415" />
        <rect x="90" y="59" width="59" height="3" rx="1.5" fill="#F5D040" fillOpacity="0.45" />

        {/* Headlamp circle */}
        <ellipse cx="134" cy="46" rx="7" ry="5.5" fill="#FFF9C4" />
        <ellipse cx="134" cy="46" rx="5" ry="3.5" fill="#FFEE00" />
        <circle cx="134" cy="46" r="2.5" fill="white" fillOpacity="0.85" />
        {/* Lamp beam rays */}
        <line x1="140" y1="43" x2="154" y2="36" stroke="#FFF9C4" strokeWidth="1.8" strokeOpacity="0.5" strokeLinecap="round" />
        <line x1="140" y1="46" x2="156" y2="42" stroke="#FFF9C4" strokeWidth="1.2" strokeOpacity="0.35" strokeLinecap="round" />
        <line x1="139" y1="49" x2="153" y2="50" stroke="#FFF9C4" strokeWidth="1" strokeOpacity="0.25" strokeLinecap="round" />

        {/* ── Snout (big flat capybara snout) ──────── */}
        <ellipse cx="146" cy="80" rx="15" ry="10" fill="#9B7045" />
        <ellipse cx="150" cy="80" rx="9" ry="7" fill="#8B5E35" />
        {/* Nostrils */}
        <ellipse cx="144" cy="77" rx="2.8" ry="2" fill="#5C3820" fillOpacity="0.75" />
        <ellipse cx="151" cy="77" rx="2.8" ry="2" fill="#5C3820" fillOpacity="0.75" />

        {/* ── Eye ──────────────────────────────────── */}
        {state !== 'success' && (
          <>
            <circle cx="116" cy="63" r="8" fill="#EDE0CC" />
            <circle cx="118" cy="64" r="5" fill="#1A0D00" />
            <circle cx="115" cy="62" r="2.5" fill="#3D2000" fillOpacity="0.5" />
            <circle cx="120" cy="62" r="2" fill="white" />
            <circle cx="117" cy="60" r="0.9" fill="white" fillOpacity="0.6" />
          </>
        )}
        {/* Success: happy ^.^ eyes */}
        {state === 'success' && (
          <>
            <path d="M108,60 Q116,54 124,60" stroke="#1A0D00" strokeWidth="3" fill="none" strokeLinecap="round" />
            <ellipse cx="112" cy="68" rx="6" ry="4" fill="#FF8080" fillOpacity="0.3" />
          </>
        )}

        {/* ── Mouth ────────────────────────────────── */}
        {state !== 'success' && state !== 'error' && (
          <path d="M136,87 Q142,91 148,88" stroke="#7A5020" strokeWidth="2.2" fill="none" strokeLinecap="round" />
        )}
        {state === 'success' && (
          <path d="M133,86 Q141,93 149,87" stroke="#7A5020" strokeWidth="2.5" fill="none" strokeLinecap="round" />
        )}
        {state === 'error' && (
          <path d="M136,91 Q142,87 148,90" stroke="#7A5020" strokeWidth="2.2" fill="none" strokeLinecap="round" />
        )}

        {/* ── State overlays ───────────────────────── */}
        {state === 'success' && (
          <>
            {/* Gold stars */}
            <text x="90" y="52" fill="#F5C842" fontSize="14" fontFamily="sans-serif">✦</text>
            <text x="146" y="48" fill="#F5C842" fontSize="10" fontFamily="sans-serif">✦</text>
            <text x="100" y="40" fill="#F5C842" fontSize="8" fontFamily="sans-serif">✦</text>
          </>
        )}
        {state === 'error' && (
          <>
            {/* Sweat drop */}
            <ellipse cx="107" cy="56" rx="4" ry="6" fill="#93C5FD" fillOpacity="0.65" />
            <ellipse cx="107" cy="50" rx="3" ry="3" fill="#93C5FD" fillOpacity="0.65" />
            {/* Question mark */}
            <text x="144" y="52" fill="#EF4444" fontSize="15" fontFamily="sans-serif" fontWeight="bold">?</text>
          </>
        )}

        {/* ── Near front legs ──────────────────────── */}
        <rect x="96" y="126" width="13" height="22" rx="6" fill="#8B6340" />
        <rect x="114" y="128" width="12" height="20" rx="5" fill="#8B6340" />
        {/* Feet */}
        <ellipse cx="102.5" cy="147" rx="9" ry="5" fill="#7A5530" />
        <ellipse cx="120" cy="147" rx="8" ry="4.5" fill="#7A5530" />

        {/* ── Pickaxe arm ──────────────────────────── */}
        {/* Group rotates around shoulder (106, 109) */}
        <g transform={`rotate(${pickaxeAngle}, 106, 109)`}>
          {/* Upper arm */}
          <path d="M106,109 C114,118 122,126 126,132" stroke="#8B6340" strokeWidth="10" strokeLinecap="round" fill="none" />
          {/* Forearm */}
          <path d="M122,126 C128,132 133,136 137,140" stroke="#7A5530" strokeWidth="8" strokeLinecap="round" fill="none" />
          {/* Pickaxe handle (wood) */}
          <line x1="130" y1="134" x2="152" y2="148" stroke="#7B4A1C" strokeWidth="5" strokeLinecap="round" />
          <line x1="130" y1="134" x2="152" y2="148" stroke="#A06030" strokeWidth="2" strokeLinecap="round" strokeOpacity="0.5" />
          {/* Pickaxe metal head */}
          {/* Forward prong */}
          <path d="M148,142 C152,135 158,130 162,134 C160,138 156,140 152,144 Z" fill="#9BA8B8" />
          {/* Back prong */}
          <path d="M148,142 C144,146 138,148 136,144 C138,140 142,140 146,142 Z" fill="#9BA8B8" />
          {/* Center band (gold) */}
          <ellipse cx="148" cy="142" rx="5" ry="4" fill="#C8A849" />
          <ellipse cx="148" cy="142" rx="3" ry="2.5" fill="#F5D040" fillOpacity="0.7" />
          {/* Tip sparkle when mining */}
          {state === 'mining' && pickaxeAngle > 15 && (
            <>
              <circle cx="162" cy="134" r="2" fill="#F5C842" fillOpacity="0.9" />
              <line x1="162" y1="130" x2="166" y2="126" stroke="#F5C842" strokeWidth="1.5" strokeOpacity="0.7" />
              <line x1="165" y1="134" x2="170" y2="133" stroke="#F5C842" strokeWidth="1.5" strokeOpacity="0.6" />
            </>
          )}
        </g>
      </svg>
    </div>
  );
}
