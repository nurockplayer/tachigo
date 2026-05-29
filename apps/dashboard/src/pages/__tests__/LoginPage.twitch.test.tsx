import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('@refinedev/core', () => ({
  useLogin: () => ({ mutateAsync: vi.fn() }),
}))

vi.mock('@/services/api', () => ({
  apiBaseURL: 'http://localhost:8080',
}))

import LoginPage from '@/pages/LoginPage'

describe('LoginPage Twitch OAuth button', () => {
  afterEach(() => { document.body.innerHTML = '' })

  it('顯示 Twitch 登入按鈕', async () => {
    const container = document.createElement('div')
    document.body.appendChild(container)
    await act(async () => {
      createRoot(container).render(<LoginPage />)
    })
    const btn = container.querySelector('[data-testid="twitch-login-btn"]')
    expect(btn).not.toBeNull()
    expect(btn!.textContent).toContain('Twitch')
  })

  it('點擊 Twitch 按鈕觸發正確 redirect', async () => {
    const container = document.createElement('div')
    document.body.appendChild(container)
    let assignedHref = ''
    Object.defineProperty(window, 'location', {
      value: { ...window.location, set href(v: string) { assignedHref = v } },
      writable: true,
    })
    await act(async () => {
      createRoot(container).render(<LoginPage />)
    })
    const btn = container.querySelector<HTMLButtonElement>('[data-testid="twitch-login-btn"]')!
    await act(async () => { btn.click() })
    expect(assignedHref).toBe('http://localhost:8080/api/v1/auth/twitch?redirect_to=%2F')
  })
})
