import { render, screen } from '@testing-library/react'
import { vi } from 'vitest'

import i18n from '../../i18n'
import { MarioHUD } from './MarioHUD'

vi.mock('../hooks/useSound', () => ({
  useSound: () => ({
    playMiningClick: vi.fn(),
    playRewardComplete: vi.fn(),
    playMaxClicks: vi.fn(),
    playToggleWatch: vi.fn(),
    startBgMusic: vi.fn(),
    stopBgMusic: vi.fn(),
    bridgeStatus: 'unsupported',
  }),
}))

describe('MarioHUD bridge status', () => {
  it('shows a tab-audio warning when the content-script bridge is unavailable', () => {
    render(<MarioHUD />)

    expect(screen.getByText(i18n.t('hud.tabAudioUnavailable'))).toBeInTheDocument()
  })

  it('renders a claim control when navigation is available', () => {
    render(<MarioHUD onNavigate={vi.fn()} />)

    expect(screen.getByRole('button', { name: i18n.t('claim.title') })).toBeInTheDocument()
  })

  it('does not render a claim control when navigation is unavailable', () => {
    render(<MarioHUD />)

    expect(screen.queryByRole('button', { name: i18n.t('claim.title') })).not.toBeInTheDocument()
  })
})
