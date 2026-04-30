import { act, fireEvent, render, screen } from '@testing-library/react'

import '../../i18n'
import { LoginScreen } from './LoginScreen'

describe('LoginScreen', () => {
  it('prevents duplicate submissions while loading', async () => {
    const onLogin = vi.fn()
    render(<LoginScreen onLogin={onLogin} />)

    vi.useFakeTimers()

    fireEvent.change(screen.getByPlaceholderText('USERNAME'), { target: { value: 'demo-user' } })
    fireEvent.change(screen.getByPlaceholderText('PASSWORD'), { target: { value: 'demo-pass' } })

    fireEvent.click(screen.getByRole('button', { name: 'LOGIN' }))
    fireEvent.click(screen.getByRole('button', { name: 'LOGIN...' }))
    fireEvent.keyDown(screen.getByPlaceholderText('PASSWORD'), { key: 'Enter', code: 'Enter' })

    await act(async () => {
      vi.advanceTimersByTime(1300)
    })

    expect(onLogin).toHaveBeenCalledTimes(1)
  })

  it('clears the pending unlock timer on unmount', async () => {
    const onLogin = vi.fn()
    const { unmount } = render(<LoginScreen onLogin={onLogin} />)

    vi.useFakeTimers()

    fireEvent.change(screen.getByPlaceholderText('USERNAME'), { target: { value: 'demo-user' } })
    fireEvent.change(screen.getByPlaceholderText('PASSWORD'), { target: { value: 'demo-pass' } })
    fireEvent.click(screen.getByRole('button', { name: 'LOGIN' }))

    unmount()

    await act(async () => {
      vi.advanceTimersByTime(1300)
    })

    expect(onLogin).not.toHaveBeenCalled()
  })
})
