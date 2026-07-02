// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { joinRaffle } from '../../services/api'
import { useTwitch } from '../../hooks/useTwitch'
import { RaffleJoinButton } from './RaffleJoinButton'

vi.mock('../../hooks/useTwitch')
vi.mock('../../services/api', () => ({
  joinRaffle: vi.fn(),
}))

const mockedUseTwitch = vi.mocked(useTwitch)
const mockedJoinRaffle = vi.mocked(joinRaffle)

function mockBackendReady(backendReady: boolean) {
  mockedUseTwitch.mockReturnValue({
    context: null,
    jwt: '',
    products: [],
    tPointEnabled: false,
    authError: null,
    backendReady,
  })
}

describe('RaffleJoinButton', () => {
  beforeEach(() => {
    mockBackendReady(true)
    mockedJoinRaffle.mockResolvedValue({ status: 200 })
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('renders nothing when backendReady is false', () => {
    mockBackendReady(false)

    const { container } = render(
      <RaffleJoinButton raffleId="raffle-1" entryOpen raffleActive />,
    )

    expect(container.textContent).toBe('')
  })

  it('renders nothing when raffleActive is false', () => {
    const { container } = render(
      <RaffleJoinButton raffleId="raffle-1" entryOpen raffleActive={false} />,
    )

    expect(container.textContent).toBe('')
  })

  it('shows join button when raffle is active and entry is open', () => {
    render(<RaffleJoinButton raffleId="raffle-1" entryOpen raffleActive />)

    expect(screen.getByRole('button', { name: '我要抽獎' })).toBeTruthy()
  })

  it('shows closed text when raffle is active and entry is closed', () => {
    render(<RaffleJoinButton raffleId="raffle-1" entryOpen={false} raffleActive />)

    expect(screen.getByText('報名已截止')).toBeTruthy()
  })

  it('shows joined state after a 200 response', async () => {
    mockedJoinRaffle.mockResolvedValue({ status: 200 })
    render(<RaffleJoinButton raffleId="raffle-1" entryOpen raffleActive />)

    fireEvent.click(screen.getByRole('button', { name: '我要抽獎' }))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: '已加入 ✓' })).toBeTruthy()
    })
  })

  it('shows not eligible state after a 403 response', async () => {
    mockedJoinRaffle.mockResolvedValue({ status: 403 })
    render(<RaffleJoinButton raffleId="raffle-1" entryOpen raffleActive />)

    fireEvent.click(screen.getByRole('button', { name: '我要抽獎' }))

    await waitFor(() => {
      expect(screen.getByText('需訂閱才能參加')).toBeTruthy()
    })
  })

  it('shows joined state after a 409 response', async () => {
    mockedJoinRaffle.mockResolvedValue({ status: 409 })
    render(<RaffleJoinButton raffleId="raffle-1" entryOpen raffleActive />)

    fireEvent.click(screen.getByRole('button', { name: '我要抽獎' }))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: '已加入 ✓' })).toBeTruthy()
    })
  })
})
