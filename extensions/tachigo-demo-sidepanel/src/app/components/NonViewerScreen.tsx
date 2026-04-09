import { useTranslation } from 'react-i18next'

export function NonViewerScreen() {
  const { t } = useTranslation()

  return (
    <div
      className="screen-enter"
      style={{
        width: '100%',
        height: '100%',
        background: 'linear-gradient(180deg, #0d0d18 0%, #12102a 100%)',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '0 32px',
        gap: 0,
        position: 'relative',
      }}
    >
      {/* Subtle glow behind icon */}
      <div
        style={{
          position: 'absolute',
          top: '35%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          width: 200,
          height: 200,
          borderRadius: '50%',
          background: 'radial-gradient(ellipse, rgba(200,168,73,0.05) 0%, transparent 70%)',
          pointerEvents: 'none',
        }}
      />

      {/* Pickaxe & watch icon */}
      <div style={{ marginBottom: 22 }}>
        <svg width="52" height="52" viewBox="0 0 52 52" fill="none">
          <circle cx="26" cy="26" r="24" fill="rgba(200,168,73,0.07)" stroke="rgba(200,168,73,0.2)" strokeWidth="1.5" />
          {/* Eye / watch symbol */}
          <ellipse cx="26" cy="26" rx="12" ry="8" stroke="rgba(200,168,73,0.5)" strokeWidth="2" fill="none" />
          <circle cx="26" cy="26" r="4" fill="rgba(200,168,73,0.4)" />
          <circle cx="26" cy="26" r="2" fill="#c8a849" />
          {/* Small sparkle dots */}
          <circle cx="12" cy="16" r="1.5" fill="#c8a849" fillOpacity="0.4" />
          <circle cx="40" cy="36" r="1.5" fill="#c8a849" fillOpacity="0.3" />
          <circle cx="38" cy="15" r="1" fill="#c8a849" fillOpacity="0.35" />
        </svg>
      </div>

      {/* Brand */}
      <div
        className="game-pixel"
        style={{
          fontSize: 7,
          color: 'rgba(200,168,73,0.5)',
          letterSpacing: '0.15em',
          marginBottom: 22,
        }}
      >
        TACHIGO
      </div>

      {/* Divider */}
      <div
        style={{
          width: 48,
          height: 1,
          background: 'linear-gradient(90deg, transparent, rgba(200,168,73,0.3), transparent)',
          marginBottom: 22,
        }}
      />

      {/* Main message */}
      <p
        className="game-sans"
        style={{
          fontSize: 13,
          color: '#d1d5db',
          textAlign: 'center',
          lineHeight: 1.7,
          margin: 0,
          marginBottom: 20,
          letterSpacing: '0.02em',
        }}
      >
        {t('nonViewer.titleLine1')}
        <br />
        <span style={{ color: '#c8a849' }}>{t('nonViewer.titleHighlight')}</span>
      </p>

      {/* Sub-hint */}
      <p
        className="game-sans"
        style={{
          fontSize: 11,
          color: 'rgba(156,163,175,0.5)',
          textAlign: 'center',
          lineHeight: 1.6,
          margin: 0,
          letterSpacing: '0.02em',
        }}
      >
        {t('nonViewer.hintLine1')}
        <br />
        {t('nonViewer.hintLine2')}
      </p>

      {/* Bottom ore decoration */}
      <div style={{ position: 'absolute', bottom: 28, display: 'flex', gap: 8, alignItems: 'center' }}>
        {[
          { color: '#c8a849', size: 5, opacity: 0.4 },
          { color: '#93c5fd', size: 4, opacity: 0.35 },
          { color: '#c8a849', size: 6, opacity: 0.3 },
          { color: '#93c5fd', size: 4, opacity: 0.3 },
          { color: '#c8a849', size: 5, opacity: 0.35 },
        ].map((dot, i) => (
          <div
            key={i}
            style={{
              width: dot.size,
              height: dot.size,
              borderRadius: 2,
              background: dot.color,
              opacity: dot.opacity,
              transform: 'rotate(45deg)',
            }}
          />
        ))}
      </div>
    </div>
  );
}
