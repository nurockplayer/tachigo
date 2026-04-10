import { render, screen } from '@testing-library/react'
import { vi } from 'vitest'

import '../../i18n'
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

    expect(screen.getByText('TAB AUDIO OFF')).toBeInTheDocument()
  })
})
