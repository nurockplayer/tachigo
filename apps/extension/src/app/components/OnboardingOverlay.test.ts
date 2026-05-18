// @vitest-environment jsdom
import assert from 'node:assert/strict'
import React from 'react'
import { afterEach, describe, test, vi } from 'vitest'
import { cleanup, fireEvent, render, screen } from '@testing-library/react'

import { OnboardingOverlay } from './OnboardingOverlay'

const translations: Record<string, string> = {
  'onboarding.title': 'Mining tour',
  'onboarding.progress': '{{current}} / {{total}}',
  'onboarding.next': 'Next',
  'onboarding.finish': 'Start mining',
  'onboarding.skip': 'Skip',
  'onboarding.steps.points.title': 'Earn points while watching',
  'onboarding.steps.points.body': 'Stay in the stream to earn CPC.',
  'onboarding.steps.rewards.title': 'Claim rewards',
  'onboarding.steps.rewards.body': 'Turn CPC into TCG and spend it.',
}

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, values?: Record<string, number>) => {
      const template = translations[key] ?? key

      if (!values) {
        return template
      }

      return Object.entries(values).reduce(
        (text, [name, value]) => text.replace(`{{${name}}}`, String(value)),
        template,
      )
    },
  }),
}))

afterEach(() => {
  cleanup()
})

describe('OnboardingOverlay', () => {
  test('renders the first onboarding step as a modal dialog', () => {
    render(React.createElement(OnboardingOverlay, { onComplete: () => undefined }))

    const dialog = screen.getByRole('dialog', { name: 'Mining tour' })

    assert.equal(dialog.getAttribute('aria-modal'), 'true')
    screen.getByText('CPC')
    screen.getByText('Earn points while watching')
    screen.getByText('Stay in the stream to earn CPC.')
    screen.getByText('1 / 2')
    screen.getByRole('button', { name: 'Next' })
    screen.getByRole('button', { name: 'Skip' })
  })

  test('advances to the final rewards step before completing', () => {
    const onComplete = vi.fn()
    render(React.createElement(OnboardingOverlay, { onComplete }))

    fireEvent.click(screen.getByRole('button', { name: 'Next' }))

    screen.getByText('TCG')
    screen.getByText('Claim rewards')
    screen.getByText('Turn CPC into TCG and spend it.')
    screen.getByText('2 / 2')
    assert.equal(onComplete.mock.calls.length, 0)

    fireEvent.click(screen.getByRole('button', { name: 'Start mining' }))

    assert.equal(onComplete.mock.calls.length, 1)
  })

  test('skip completes onboarding from the first step', () => {
    const onComplete = vi.fn()
    render(React.createElement(OnboardingOverlay, { onComplete }))

    fireEvent.click(screen.getByRole('button', { name: 'Skip' }))

    assert.equal(onComplete.mock.calls.length, 1)
  })
})
