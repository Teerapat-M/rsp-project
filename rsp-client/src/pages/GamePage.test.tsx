/** @vitest-environment jsdom */
import React, { StrictMode } from 'react'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes, useLocation, useParams } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { GameCtx } from '../context/GameContext'
import type { Room, User } from '../types'
import { GamePage } from './GamePage'

const { api, useAuth, useGame, attach } = vi.hoisted(() => {
  const attach = vi.fn()
  return {
    attach,
    api: { getActiveRoom: vi.fn() },
    useAuth: vi.fn(),
    useGame: vi.fn(),
  }
})

vi.mock('../api', () => ({ api }))
vi.mock('../context/AuthContext', () => ({ useAuth }))
vi.mock('../context/GameContext', () => ({ useGame }))

const user: User = {
  id: 'user-1',
  displayName: 'Ada',
  isGuest: true,
  avatar: 'rock',
  color: '#4B4FD0',
}

function room(id: string, status: Room['status']): Room {
  return {
    id,
    passcode: '123456',
    hostId: user.id,
    winsToFinish: 3,
    turnSeconds: 15,
    status,
    players: [],
  }
}

function stubGame(): GameCtx {
  return {
    room: null,
    view: null,
    remaining: null,
    roundResult: null,
    nextRoundLeft: null,
    connectionLost: false,
    reconnectAttempt: 0,
    reconnectGraceLeft: 20,
    playAgainReady: 0,
    joinError: null,
    createRoom: vi.fn(),
    joinRoom: vi.fn(),
    clearJoinError: vi.fn(),
    leaveRoom: vi.fn(),
    startGame: vi.fn(),
    attach,
    loadRoom: vi.fn(),
    submitMove: vi.fn(),
    readyUp: vi.fn(),
    setLobbyReady: vi.fn(),
    claimWin: vi.fn(),
    playAgain: vi.fn(),
    retryConnection: vi.fn(),
  }
}

function PathProbe() {
  const loc = useLocation()
  return <div data-testid="loc">{`${loc.pathname}${loc.search}`}</div>
}

function WaitingProbe() {
  const { id } = useParams()
  return <div data-testid="waiting">{id}</div>
}

function renderPlay() {
  return render(
    <StrictMode>
      <MemoryRouter initialEntries={['/play']}>
        <PathProbe />
        <Routes>
          <Route path="/play" element={<GamePage />} />
          <Route path="/lobby" element={<div data-testid="lobby">lobby</div>} />
          <Route path="/room/:id" element={<WaitingProbe />} />
        </Routes>
      </MemoryRouter>
    </StrictMode>,
  )
}

function mockActiveRoom(value: Room | null) {
  // Resolved-but-async so StrictMode cleanup can cancel the first run
  // before the second run applies the response.
  vi.mocked(api.getActiveRoom).mockImplementation(() => Promise.resolve(value))
}

describe('GamePage bare /play restore under StrictMode', () => {
  afterEach(() => {
    cleanup()
  })

  beforeEach(() => {
    attach.mockReset()
    vi.mocked(api.getActiveRoom).mockReset()
    vi.mocked(useAuth).mockReturnValue({
      user,
      ready: true,
      login: vi.fn(),
      register: vi.fn(),
      google: vi.fn(),
      guest: vi.fn(),
      logout: vi.fn(),
      refresh: vi.fn(),
    })
    vi.mocked(useGame).mockReturnValue(stubGame())
  })

  it('restores a playing room onto /play?room=', async () => {
    mockActiveRoom(room('mid-game', 'playing'))
    renderPlay()
    expect(screen.getAllByText('Connecting…').length).toBeGreaterThan(0)
    await waitFor(() => {
      expect(screen.getByTestId('loc').textContent).toBe('/play?room=mid-game')
    })
    await waitFor(() => {
      expect(attach).toHaveBeenCalledWith('mid-game')
    })
    expect(screen.queryByTestId('lobby')).toBeNull()
  })

  it('restores a waiting room onto /room/:id', async () => {
    mockActiveRoom(room('wait-room', 'waiting'))
    renderPlay()
    expect(screen.getAllByText('Connecting…').length).toBeGreaterThan(0)
    await waitFor(() => {
      expect(screen.getByTestId('waiting').textContent).toBe('wait-room')
    })
    expect(screen.getByTestId('loc').textContent).toBe('/room/wait-room')
    expect(screen.queryByTestId('lobby')).toBeNull()
    expect(attach).not.toHaveBeenCalled()
  })

  it('redirects to /lobby when there is no active room', async () => {
    mockActiveRoom(null)
    renderPlay()
    expect(screen.getAllByText('Connecting…').length).toBeGreaterThan(0)
    await waitFor(() => {
      expect(screen.getByTestId('lobby').textContent).toBe('lobby')
    })
    expect(screen.getByTestId('loc').textContent).toBe('/lobby')
    expect(attach).not.toHaveBeenCalled()
  })
})
