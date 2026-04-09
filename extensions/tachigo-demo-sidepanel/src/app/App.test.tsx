import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'

import '../i18n'
import App from './App'

describe('demo app flow', () => {
  it('starts in english on first launch', async () => {
    render(<App />)

    expect(await screen.findByPlaceholderText('USERNAME')).toBeInTheDocument()
    expect(screen.getByText('FORGOT PASSWORD?')).toBeInTheDocument()
  })

  it('keeps the selected language after reopening the app', async () => {
    const { unmount } = render(<App />)

    expect(await screen.findByPlaceholderText('USERNAME')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '繁中' }))

    expect(await screen.findByPlaceholderText('帳號')).toBeInTheDocument()
    expect(screen.getByText('忘記密碼？')).toBeInTheDocument()

    unmount()

    render(<App />)

    expect(await screen.findByPlaceholderText('帳號')).toBeInTheDocument()
    expect(screen.getByText('忘記密碼？')).toBeInTheDocument()
  })

  it('moves from login to loading after valid credentials are submitted', async () => {
    render(<App />)

    const usernameInput = await screen.findByPlaceholderText('USERNAME')
    const passwordInput = screen.getByPlaceholderText('PASSWORD')

    vi.useFakeTimers()

    fireEvent.change(usernameInput, { target: { value: 'demo-user' } })
    fireEvent.change(passwordInput, { target: { value: 'demo-pass' } })
    fireEvent.keyDown(passwordInput, { key: 'Enter', code: 'Enter' })

    expect(screen.getAllByText('LOGIN...').length).toBeGreaterThan(0)

    await act(async () => {
      vi.advanceTimersByTime(1300)
    })

    expect(screen.getByText('LOADING...')).toBeInTheDocument()

    vi.useRealTimers()
  })

  it('switches to simplified chinese and updates the login copy', async () => {
    render(<App />)

    expect(await screen.findByPlaceholderText('USERNAME')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '简中' }))

    await waitFor(() => {
      expect(screen.getByPlaceholderText('账号')).toBeInTheDocument()
    })
  })
})
