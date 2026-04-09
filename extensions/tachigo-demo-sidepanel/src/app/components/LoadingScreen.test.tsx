import { render, screen, waitFor } from '@testing-library/react'

import '../../i18n'
import { LoadingScreen } from './LoadingScreen'

describe('LoadingScreen', () => {
  it('shows loading text and completes automatically after progress finishes', async () => {
    const onComplete = vi.fn()

    render(<LoadingScreen onComplete={onComplete} />)

    expect(screen.getByText('LOADING...')).toBeInTheDocument()

    await waitFor(() => {
      expect(onComplete).toHaveBeenCalledTimes(1)
    }, { timeout: 4000 })
  }, 5000)
})
