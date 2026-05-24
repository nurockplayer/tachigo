import type { CSSProperties } from 'react'
import { useTranslation } from 'react-i18next'

interface EntrySceneProps {
  onEnter: () => void
}

const bubbleStyle: CSSProperties = {
  position: 'absolute',
  border: '1px solid rgba(191, 219, 254, 0.6)',
  borderRadius: 999,
  background: 'rgba(186, 230, 253, 0.12)',
  boxShadow: '0 0 16px rgba(125, 211, 252, 0.25)',
}

export function EntryScene({ onEnter }: EntrySceneProps) {
  const { t } = useTranslation()

  return (
    <button
      type="button"
      aria-label={t('entry.cta')}
      onClick={onEnter}
      style={{
        appearance: 'none',
        position: 'relative',
        display: 'grid',
        width: '100%',
        height: '100%',
        minHeight: 600,
        overflow: 'hidden',
        border: 0,
        padding: 'clamp(24px, 8vw, 44px) clamp(20px, 7vw, 36px)',
        cursor: 'pointer',
        color: '#f8fafc',
        textAlign: 'left',
        fontFamily: 'var(--pixel-font-family)',
        background:
          'linear-gradient(180deg, #082f49 0%, #075985 35%, #0f766e 72%, #052e2b 100%)',
      }}
    >
      <div
        aria-hidden="true"
        style={{
          position: 'absolute',
          inset: 0,
          background:
            'radial-gradient(circle at 20% 12%, rgba(224,242,254,0.28), transparent 20%), radial-gradient(circle at 75% 28%, rgba(45,212,191,0.28), transparent 24%), linear-gradient(180deg, transparent 55%, rgba(8,47,73,0.44) 100%)',
        }}
      />
      <div style={{ ...bubbleStyle, top: '16%', left: '18%', width: 11, height: 11 }} />
      <div style={{ ...bubbleStyle, top: '23%', right: '22%', width: 17, height: 17 }} />
      <div style={{ ...bubbleStyle, bottom: '33%', left: '12%', width: 8, height: 8 }} />

      <div
        aria-hidden="true"
        style={{
          position: 'absolute',
          left: '50%',
          top: '36%',
          width: 'min(68%, 250px)',
          aspectRatio: '2.2 / 1',
          transform: 'translateX(-50%)',
        }}
      >
        <div
          style={{
            position: 'absolute',
            inset: '18% 16% 8% 8%',
            borderRadius: '52% 46% 46% 56%',
            background: 'linear-gradient(150deg, #dbeafe 0%, #60a5fa 52%, #2563eb 100%)',
            boxShadow: '0 22px 42px rgba(8, 47, 73, 0.35)',
          }}
        />
        <div
          style={{
            position: 'absolute',
            right: '4%',
            top: '28%',
            width: '24%',
            height: '34%',
            borderRadius: '70% 20% 70% 20%',
            background: '#93c5fd',
            transform: 'rotate(23deg)',
          }}
        />
        <div
          style={{
            position: 'absolute',
            left: '26%',
            top: '42%',
            width: 9,
            height: 9,
            borderRadius: 999,
            background: '#0f172a',
            boxShadow: '0 0 0 3px rgba(255,255,255,0.35)',
          }}
        />
        <div
          style={{
            position: 'absolute',
            left: '34%',
            top: '60%',
            width: '22%',
            height: 2,
            borderRadius: 999,
            background: 'rgba(15,23,42,0.38)',
          }}
        />
      </div>

      <div
        aria-hidden="true"
        style={{
          position: 'absolute',
          inset: 'auto -8% 0',
          height: '22%',
          background:
            'linear-gradient(175deg, rgba(20,184,166,0.72) 0 42%, transparent 43%), linear-gradient(185deg, transparent 0 32%, rgba(13,148,136,0.7) 33% 68%, transparent 69%)',
          opacity: 0.9,
        }}
      />

      <div
        style={{
          position: 'relative',
          zIndex: 1,
          display: 'grid',
          alignSelf: 'end',
          gap: 14,
          width: '100%',
          maxWidth: 330,
        }}
      >
        <div
          style={{
            color: '#bae6fd',
            fontSize: 10,
            letterSpacing: 0,
            textTransform: 'uppercase',
            textShadow: '0 2px 10px rgba(8,47,73,0.85)',
          }}
        >
          {t('entry.eyebrow')}
        </div>
        <h1
          style={{
            margin: 0,
            color: '#f8fafc',
            fontSize: 38,
            lineHeight: 0.92,
            letterSpacing: 0,
            textShadow: '0 4px 0 rgba(14, 116, 144, 0.55), 0 18px 38px rgba(8,47,73,0.8)',
          }}
        >
          TACHIGO
        </h1>
        <p
          style={{
            margin: 0,
            maxWidth: 290,
            color: 'rgba(224, 242, 254, 0.86)',
            fontFamily: 'var(--ui-font-family)',
            fontSize: 13,
            lineHeight: 1.55,
            textShadow: '0 2px 12px rgba(8,47,73,0.7)',
          }}
        >
          {t('entry.subtitle')}
        </p>
        <div
          style={{
            display: 'inline-flex',
            width: 'fit-content',
            minHeight: 30,
            alignItems: 'center',
            borderTop: '1px solid rgba(186,230,253,0.5)',
            color: '#fef08a',
            fontSize: 10,
            letterSpacing: 0,
            textShadow: '0 2px 10px rgba(8,47,73,0.75)',
          }}
        >
          {t('entry.cta')}
        </div>
      </div>
    </button>
  )
}
