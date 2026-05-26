import { act } from 'react'
import { createRoot } from 'react-dom/client'
import { MemoryRouter } from 'react-router'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/services/auth', () => ({
  logout: vi.fn().mockResolvedValue(undefined),
  getUserRole: vi.fn(),
}))

import { getUserRole } from '@/services/auth'
import Layout from '@/components/Layout'

function render(role: string | null) {
  const mockGetUserRole = vi.mocked(getUserRole)
  mockGetUserRole.mockReturnValue(role)

  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = createRoot(container)
  act(() => {
    root.render(
      <MemoryRouter initialEntries={['/']}>
        <Layout />
      </MemoryRouter>,
    )
  })
  return { container, root }
}

describe('Layout nav role filtering', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('streamer 看不到實況主管理與設定', () => {
    const { container } = render('streamer')
    const nav = container.querySelector('nav')!
    expect(nav.textContent).toContain('總覽')
    expect(nav.textContent).toContain('抽獎管理')
    expect(nav.textContent).toContain('交易紀錄')
    expect(nav.textContent).not.toContain('實況主管理')
    expect(nav.textContent).not.toContain('設定')
  })

  it('admin 看到所有導航項目', () => {
    const { container } = render('admin')
    const nav = container.querySelector('nav')!
    expect(nav.textContent).toContain('總覽')
    expect(nav.textContent).toContain('實況主管理')
    expect(nav.textContent).toContain('抽獎管理')
    expect(nav.textContent).toContain('交易紀錄')
    expect(nav.textContent).toContain('設定')
  })

  it('未登入（role null）只看到無限制項目', () => {
    const { container } = render(null)
    const nav = container.querySelector('nav')!
    expect(nav.textContent).not.toContain('實況主管理')
    expect(nav.textContent).not.toContain('設定')
  })
})
