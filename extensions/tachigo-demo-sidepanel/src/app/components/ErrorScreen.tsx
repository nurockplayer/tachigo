import { useTranslation } from 'react-i18next'

export type ErrorType = 'error-account' | 'error-backend' | 'error-session';

interface ErrorConfig {
  icon: string;
  iconColor: string;
  title: string;
  message: string;
  hint: string;
  accentColor: string;
}

interface Props {
  type: ErrorType;
}

export function ErrorScreen({ type }: Props) {
  const { t } = useTranslation()

  const ERROR_CONFIGS: Record<ErrorType, ErrorConfig> = {
    'error-account': {
      icon: '⛏',
      iconColor: '#c8a849',
      title: t('error.account.title'),
      message: t('error.account.message'),
      hint: t('error.account.hint'),
      accentColor: '#c8a849',
    },
    'error-backend': {
      icon: '⚠',
      iconColor: '#f59e0b',
      title: t('error.backend.title'),
      message: t('error.backend.message'),
      hint: t('error.backend.hint'),
      accentColor: '#f59e0b',
    },
    'error-session': {
      icon: '⟳',
      iconColor: '#ef4444',
      title: t('error.session.title'),
      message: t('error.session.message'),
      hint: t('error.session.hint'),
      accentColor: '#ef4444',
    },
  }

  const config = ERROR_CONFIGS[type];

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
        padding: '0 28px',
        position: 'relative',
        overflow: 'hidden',
      }}
    >
      {/* Background glow */}
      <div
        style={{
          position: 'absolute',
          top: '40%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          width: 220,
          height: 220,
          borderRadius: '50%',
          background: `radial-gradient(ellipse, ${config.accentColor}0A 0%, transparent 70%)`,
          pointerEvents: 'none',
        }}
      />

      {/* Top brand */}
      <div
        className="game-pixel"
        style={{
          position: 'absolute',
          top: 20,
          left: '50%',
          transform: 'translateX(-50%)',
          fontSize: 6,
          color: 'rgba(200,168,73,0.3)',
          letterSpacing: '0.15em',
          whiteSpace: 'nowrap',
        }}
      >
        TACHIGO
      </div>

      {/* Error panel */}
      <div
        style={{
          width: '100%',
          maxWidth: 300,
          background: 'rgba(255,255,255,0.03)',
          border: `1px solid ${config.accentColor}35`,
          borderRadius: 10,
          padding: '22px 20px',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 12,
          boxShadow: `0 0 24px ${config.accentColor}12, inset 0 1px 0 rgba(255,255,255,0.04)`,
        }}
      >
        {/* Icon */}
        <div
          style={{
            width: 48,
            height: 48,
            borderRadius: '50%',
            background: `${config.accentColor}14`,
            border: `1px solid ${config.accentColor}40`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 22,
            color: config.iconColor,
            boxShadow: `0 0 16px ${config.accentColor}20`,
          }}
        >
          {config.icon}
        </div>

        {/* Title */}
        <div
          className="game-pixel"
          style={{
            fontSize: 7,
            color: config.accentColor,
            textAlign: 'center',
            letterSpacing: '0.08em',
            lineHeight: 1.6,
            textShadow: `0 0 10px ${config.accentColor}50`,
          }}
        >
          {config.title}
        </div>

        {/* Divider */}
        <div
          style={{
            width: '100%',
            height: 1,
            background: `linear-gradient(90deg, transparent, ${config.accentColor}25, transparent)`,
          }}
        />

        {/* Main message */}
        <p
          className="game-sans"
          style={{
            fontSize: 13,
            color: '#e5e7eb',
            textAlign: 'center',
            lineHeight: 1.65,
            margin: 0,
            letterSpacing: '0.015em',
          }}
        >
          {config.message}
        </p>

        {/* Hint text */}
        <p
          className="game-sans"
          style={{
            fontSize: 11,
            color: 'rgba(156,163,175,0.6)',
            textAlign: 'center',
            lineHeight: 1.6,
            margin: 0,
            padding: '8px 12px',
            background: 'rgba(255,255,255,0.02)',
            borderRadius: 6,
            border: '1px solid rgba(255,255,255,0.04)',
          }}
        >
          {config.hint}
        </p>
      </div>

      {/* Bottom ore row */}
      <div style={{ position: 'absolute', bottom: 24, display: 'flex', gap: 6, alignItems: 'center', opacity: 0.3 }}>
        {[5, 4, 6, 4, 5].map((size, i) => (
          <div
            key={i}
            style={{
              width: size,
              height: size,
              borderRadius: 2,
              background: config.accentColor,
              transform: 'rotate(45deg)',
            }}
          />
        ))}
      </div>
    </div>
  );
}
