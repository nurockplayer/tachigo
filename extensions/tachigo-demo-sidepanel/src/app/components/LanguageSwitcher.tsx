import type { AppLanguage } from '../../i18n/index'

const LANGS: { code: AppLanguage; label: string }[] = [
  { code: 'en', label: 'EN' },
  { code: 'zh-TW', label: '繁中' },
  { code: 'zh-CN', label: '简中' },
]

interface LanguageSwitcherProps {
  currentLanguage: AppLanguage
  onChangeLanguage: (language: AppLanguage) => void
}

export function LanguageSwitcher({ currentLanguage, onChangeLanguage }: LanguageSwitcherProps) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6, position: 'relative', zIndex: 2 }}>
      {LANGS.map((lang, idx) => (
        <div key={lang.code} style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          {idx > 0 ? (
            <span style={{ fontSize: 10, color: 'rgba(100,100,140,0.3)', fontFamily: 'var(--pixel-font-family)' }}>·</span>
          ) : null}
          <button
            type="button"
            onClick={() => onChangeLanguage(lang.code)}
            style={{
              padding: '5px 10px',
              borderRadius: 4,
              border: '1px solid rgba(255,255,255,0.1)',
              background: currentLanguage === lang.code ? 'rgba(145,70,255,0.15)' : 'transparent',
              color: currentLanguage === lang.code ? 'rgba(145,70,255,0.9)' : 'rgba(100,100,140,0.4)',
              fontSize: 9,
              fontFamily: 'var(--pixel-font-family)',
              cursor: 'pointer',
              letterSpacing: '0.08em',
              pointerEvents: 'auto',
              touchAction: 'manipulation',
              position: 'relative',
              zIndex: 3,
            }}
          >
            {lang.label}
          </button>
        </div>
      ))}
    </div>
  )
}
