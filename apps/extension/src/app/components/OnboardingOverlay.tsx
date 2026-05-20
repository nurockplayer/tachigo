import { useState } from 'react'
import { useTranslation } from 'react-i18next'

const onboardingSteps = [
  {
    key: 'points',
    marker: 'CPC',
    accent: '#ffd866',
  },
  {
    key: 'rewards',
    marker: 'TCG',
    accent: '#6fffd2',
  },
] as const

interface OnboardingOverlayProps {
  onComplete: () => void
}

export function OnboardingOverlay({ onComplete }: OnboardingOverlayProps) {
  const { t } = useTranslation()
  const [stepIndex, setStepIndex] = useState(0)
  const step = onboardingSteps[stepIndex]
  const isFinalStep = stepIndex === onboardingSteps.length - 1

  const handlePrimaryAction = () => {
    if (isFinalStep) {
      onComplete()
      return
    }

    setStepIndex((current) => current + 1)
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="tachigo-onboarding-title"
      style={{
        position: 'absolute',
        inset: 0,
        zIndex: 30,
        display: 'flex',
        alignItems: 'flex-end',
        justifyContent: 'center',
        padding: 14,
        boxSizing: 'border-box',
        background:
          'linear-gradient(180deg, rgba(4,6,20,0.18) 0%, rgba(4,6,20,0.72) 46%, rgba(4,6,20,0.94) 100%)',
        backdropFilter: 'blur(2px)',
      }}
    >
      <div
        style={{
          width: '100%',
          borderRadius: 8,
          border: '1px solid rgba(255,255,255,0.14)',
          background:
            'linear-gradient(180deg, rgba(19,24,50,0.96) 0%, rgba(10,12,28,0.98) 100%)',
          boxShadow: `0 0 0 1px rgba(0,0,0,0.88), 0 -10px 28px rgba(0,0,0,0.38), 0 0 24px ${step.accent}22`,
          padding: '13px 13px 12px',
          boxSizing: 'border-box',
          color: '#fff7d6',
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 10,
            marginBottom: 9,
          }}
        >
          <div
            id="tachigo-onboarding-title"
            style={{
              fontFamily: 'var(--pixel-font-family)',
              fontSize: 9,
              letterSpacing: '0.08em',
              color: '#fff7d6',
              textTransform: 'uppercase',
              lineHeight: 1.5,
            }}
          >
            {t('onboarding.title')}
          </div>
          <div
            style={{
              flex: '0 0 auto',
              borderRadius: 4,
              border: `1px solid ${step.accent}88`,
              color: step.accent,
              fontFamily: 'var(--pixel-font-family)',
              fontSize: 8,
              lineHeight: 1,
              padding: '5px 6px',
              background: `${step.accent}14`,
            }}
          >
            {step.marker}
          </div>
        </div>

        <div
          style={{
            fontFamily: 'var(--pixel-font-family)',
            fontSize: 11,
            lineHeight: 1.55,
            color: step.accent,
            marginBottom: 7,
          }}
        >
          {t(`onboarding.steps.${step.key}.title`)}
        </div>
        <p
          style={{
            margin: 0,
            color: 'rgba(255,255,255,0.78)',
            fontSize: 12,
            lineHeight: 1.55,
          }}
        >
          {t(`onboarding.steps.${step.key}.body`)}
        </p>

        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 10,
            marginTop: 13,
          }}
        >
          <div
            style={{
              color: 'rgba(255,255,255,0.45)',
              fontFamily: 'var(--pixel-font-family)',
              fontSize: 8,
              letterSpacing: '0.06em',
            }}
          >
            {t('onboarding.progress', {
              current: stepIndex + 1,
              total: onboardingSteps.length,
            })}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 7 }}>
            <button
              type="button"
              onClick={onComplete}
              style={{
                border: '1px solid rgba(255,255,255,0.12)',
                borderRadius: 4,
                background: 'rgba(255,255,255,0.04)',
                color: 'rgba(255,255,255,0.62)',
                fontFamily: 'var(--pixel-font-family)',
                fontSize: 8,
                lineHeight: 1,
                padding: '8px 9px',
                cursor: 'pointer',
              }}
            >
              {t('onboarding.skip')}
            </button>
            <button
              type="button"
              onClick={handlePrimaryAction}
              style={{
                border: `1px solid ${step.accent}`,
                borderRadius: 4,
                background: step.accent,
                color: '#111423',
                fontFamily: 'var(--pixel-font-family)',
                fontSize: 8,
                lineHeight: 1,
                padding: '8px 10px',
                boxShadow: `0 0 14px ${step.accent}55`,
                cursor: 'pointer',
              }}
            >
              {isFinalStep ? t('onboarding.finish') : t('onboarding.next')}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
