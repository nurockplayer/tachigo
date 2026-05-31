import type { SettingsState } from '../../extension/types'
import type { AppLanguage } from '../../i18n'

interface SettingsPanelProps {
  currentLanguage: AppLanguage
  settings: SettingsState
  onBack: () => void
  onChangeLanguage: (language: AppLanguage) => void
  onChange: (settings: SettingsState) => void
}

const languageOptions: Array<{ code: AppLanguage; label: string }> = [
  { code: 'en', label: 'English' },
  { code: 'zh-TW', label: '繁體中文' },
  { code: 'zh-CN', label: '简体中文' },
]

const panelButtonStyle = {
  minHeight: 36,
  border: '1px solid rgba(148, 163, 184, 0.22)',
  borderRadius: 8,
  background: 'rgba(15, 23, 42, 0.58)',
  color: '#e2e8f0',
  cursor: 'pointer',
  fontFamily: 'var(--pixel-font-family)',
  fontSize: 9,
} as const

const toggleRowStyle = {
  minHeight: 58,
  border: '1px solid rgba(226, 232, 240, 0.12)',
  borderRadius: 8,
  padding: '12px 14px',
  background: 'rgba(15, 23, 42, 0.72)',
  color: '#f8fafc',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  gap: 16,
} as const

function updateSetting<K extends keyof SettingsState>(
  settings: SettingsState,
  key: K,
  value: SettingsState[K],
): SettingsState {
  return {
    ...settings,
    [key]: value,
  }
}

export function SettingsPanel({ currentLanguage, settings, onBack, onChangeLanguage, onChange }: SettingsPanelProps) {
  return (
    <div
      style={{
        position: 'relative',
        width: '100%',
        height: '100%',
        minHeight: 600,
        overflow: 'hidden',
        padding: '28px 22px 24px',
        boxSizing: 'border-box',
        background:
          'linear-gradient(180deg, rgba(12, 18, 32, 0.98), rgba(24, 36, 48, 0.97) 54%, rgba(8, 13, 24, 0.99))',
        color: '#f8fafc',
        fontFamily: 'var(--ui-font-family)',
      }}
    >
      <div
        aria-hidden="true"
        style={{
          position: 'absolute',
          inset: 0,
          background:
            'linear-gradient(90deg, rgba(250,204,21,0.08) 1px, transparent 1px), linear-gradient(180deg, rgba(56,189,248,0.05) 1px, transparent 1px)',
          backgroundSize: '28px 28px',
          opacity: 0.72,
        }}
      />
      <div style={{ position: 'relative', zIndex: 1, display: 'grid', height: '100%', gap: 18 }}>
        <header style={{ display: 'grid', gap: 8 }}>
          <div
            style={{
              color: '#facc15',
              fontFamily: 'var(--pixel-font-family)',
              fontSize: 9,
              letterSpacing: 0,
              textTransform: 'uppercase',
            }}
          >
            Control Deck
          </div>
          <h1 style={{ margin: 0, fontFamily: 'var(--pixel-font-family)', fontSize: 28, lineHeight: 1 }}>
            Settings
          </h1>
        </header>

        <fieldset
          style={{
            border: '1px solid rgba(226, 232, 240, 0.12)',
            borderRadius: 8,
            padding: 14,
            margin: 0,
            background: 'rgba(15, 23, 42, 0.72)',
            display: 'grid',
            gap: 12,
          }}
        >
          <legend
            style={{
              padding: '0 8px',
              color: '#facc15',
              fontFamily: 'var(--pixel-font-family)',
              fontSize: 8,
              letterSpacing: 0,
            }}
          >
            Language
          </legend>
          {languageOptions.map((language) => (
            <label
              key={language.code}
              style={{ display: 'flex', alignItems: 'center', gap: 10, fontSize: 13, fontWeight: 800 }}
            >
              <input
                type="radio"
                name="language"
                checked={currentLanguage === language.code}
                onChange={() => onChangeLanguage(language.code)}
              />
              {language.label}
            </label>
          ))}
        </fieldset>

        <section style={{ display: 'grid', alignContent: 'start', gap: 10 }}>
          <label style={toggleRowStyle}>
            <span style={{ fontWeight: 800, fontSize: 13 }}>Sound</span>
            <input
              type="checkbox"
              checked={settings.soundEnabled}
              onChange={(event) => onChange(updateSetting(settings, 'soundEnabled', event.currentTarget.checked))}
            />
          </label>
          <label style={toggleRowStyle}>
            <span style={{ fontWeight: 800, fontSize: 13 }}>Effects</span>
            <input
              type="checkbox"
              checked={settings.effectsEnabled}
              onChange={(event) => onChange(updateSetting(settings, 'effectsEnabled', event.currentTarget.checked))}
            />
          </label>
          <label style={toggleRowStyle}>
            <span style={{ fontWeight: 800, fontSize: 13 }}>HUD</span>
            <input
              type="checkbox"
              checked={settings.hudVisible}
              onChange={(event) => onChange(updateSetting(settings, 'hudVisible', event.currentTarget.checked))}
            />
          </label>
        </section>

        <fieldset
          style={{
            border: '1px solid rgba(226, 232, 240, 0.12)',
            borderRadius: 8,
            padding: 14,
            margin: 0,
            background: 'rgba(15, 23, 42, 0.72)',
            display: 'grid',
            gap: 12,
          }}
        >
          <legend
            style={{
              padding: '0 8px',
              color: '#facc15',
              fontFamily: 'var(--pixel-font-family)',
              fontSize: 8,
              letterSpacing: 0,
            }}
          >
            Screen
          </legend>
          <label style={{ display: 'flex', alignItems: 'center', gap: 10, fontSize: 13, fontWeight: 800 }}>
            <input
              type="radio"
              name="screenMode"
              checked={settings.screenMode === 'compact'}
              onChange={() => onChange(updateSetting(settings, 'screenMode', 'compact'))}
            />
            Compact
          </label>
          <label style={{ display: 'flex', alignItems: 'center', gap: 10, fontSize: 13, fontWeight: 800 }}>
            <input
              type="radio"
              name="screenMode"
              checked={settings.screenMode === 'focus'}
              onChange={() => onChange(updateSetting(settings, 'screenMode', 'focus'))}
            />
            Focus
          </label>
        </fieldset>

        <button type="button" onClick={onBack} style={{ ...panelButtonStyle, alignSelf: 'end' }}>
          Back
        </button>
      </div>
    </div>
  )
}
