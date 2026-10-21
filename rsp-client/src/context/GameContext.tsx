import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { api } from '../api'
import type { PlayableMove, PlayerView, Room, RoundResultView, WsConnection } from '../types'

type JoinError = { code: 'WRONG_PASSCODE' | 'ROOM_FULL'; passcode: string; message: string } | null

type GameCtx = {
  room: Room | null
  view: PlayerView | null
  remaining: number | null
  roundResult: RoundResultView | null
  connectionLost: boolean
  reconnectAttempt: number
  playAgainReady: number
  joinError: JoinError
  createRoom: (winsToFinish: number, turnSeconds: number) => Promise<Room>
  joinRoom: (passcode: string) => Promise<Room | null>
  clearJoinError: () => void
  leaveRoom: () => Promise<void>
  startGame: () => Promise<void>
  attach: (roomId: string) => void
  loadRoom: (roomId: string) => Promise<Room>
  submitMove: (move: PlayableMove) => void
  readyUp: () => void
  claimWin: () => void
  playAgain: () => void
  retryConnection: () => void
}

const Ctx = createContext<GameCtx | null>(null)

export function GameProvider({ children }: { children: ReactNode }) {
  const [room, setRoom] = useState<Room | null>(null)
  const [view, setView] = useState<PlayerView | null>(null)
  const [remaining, setRemaining] = useState<number | null>(null)
  const [roundResult, setRoundResult] = useState<RoundResultView | null>(null)
  const [connectionLost, setConnectionLost] = useState(false)
  const [reconnectAttempt, setReconnectAttempt] = useState(2)
  const [playAgainReady, setPlayAgainReady] = useState(0)
  const [joinError, setJoinError] = useState<JoinError>(null)
  const conn = useRef<WsConnection | null>(null)
  const roomId = useRef<string | null>(null)

  const attach = useCallback((id: string) => {
    conn.current?.close()
    roomId.current = id
    conn.current = api.connectMatch(id, {
      onEvent: (ev) => {
        if (ev.type === 'room.updated') {
          setRoom({ ...ev.room, players: ev.room.players.map((p) => ({ ...p })) })
        }
        if (ev.type === 'match.started') {
          setView(ev.view)
          setRoundResult(null)
          setRemaining(ev.view.remainingSeconds)
          setPlayAgainReady(0)
        }
        if (ev.type === 'tick') setRemaining(ev.remainingSeconds)
        if (ev.type === 'view') {
          setView(ev.view)
          setRemaining(ev.view.remainingSeconds)
        }
        if (ev.type === 'round.resolved') {
          setRoundResult(ev.result)
          setView(ev.view)
        }
        if (ev.type === 'round.started') {
          setView(ev.view)
          setRoundResult(null)
          setRemaining(ev.view.remainingSeconds)
        }
        if (ev.type === 'match.finished') {
          setView(ev.view)
          setRoundResult(null)
        }
        if (ev.type === 'opponent.disconnected' || ev.type === 'opponent.reconnected') {
          setView(ev.view)
        }
        if (ev.type === 'connection.lost') setConnectionLost(true)
        if (ev.type === 'connection.restored') setConnectionLost(false)
        if (ev.type === 'play_again.waiting') setPlayAgainReady(ev.ready)
      },
    })
  }, [])

  useEffect(() => () => conn.current?.close(), [])

  // Server broadcasts round.started / match.finished after the result pause.
  // If that event is late, drop the overlay locally so both players can still
  // return to play (or the result screen) after nextRoundIn seconds.
  useEffect(() => {
    if (!roundResult) return
    const roundNo = roundResult.roundNumber
    const ms = Math.max(1, roundResult.nextRoundIn || 3) * 1000
    const t = window.setTimeout(() => {
      setRoundResult((cur) => (cur && cur.roundNumber === roundNo ? null : cur))
    }, ms)
    return () => window.clearTimeout(t)
  }, [roundResult])

  const value = useMemo<GameCtx>(
    () => ({
      room,
      view,
      remaining,
      roundResult,
      connectionLost,
      reconnectAttempt,
      playAgainReady,
      joinError,
      createRoom: async (winsToFinish, turnSeconds) => {
        const r = await api.createRoom({ winsToFinish, turnSeconds })
        setRoom(r)
        attach(r.id)
        return r
      },
      joinRoom: async (passcode) => {
        try {
          const r = await api.joinRoom(passcode)
          setJoinError(null)
          setRoom(r)
          attach(r.id)
          return r
        } catch (e) {
          const err = e as Error & { code?: string; passcode?: string }
          if (err.code === 'WRONG_PASSCODE' || err.code === 'ROOM_FULL') {
            setJoinError({
              code: err.code,
              passcode: err.passcode ?? passcode,
              message: err.message,
            })
            return null
          }
          throw e
        }
      },
      clearJoinError: () => setJoinError(null),
      leaveRoom: async () => {
        if (roomId.current) await api.leaveRoom(roomId.current)
        conn.current?.close()
        setRoom(null)
        setView(null)
        setRoundResult(null)
      },
      startGame: async () => {
        if (room) await api.startGame(room.id)
      },
      attach,
      loadRoom: async (id) => {
        const r = await api.getRoom(id)
        setRoom(r)
        attach(id)
        return r
      },
      submitMove: (move) => conn.current?.send({ type: 'submit', move }),
      readyUp: () => conn.current?.send({ type: 'ready' }),
      claimWin: () => conn.current?.send({ type: 'claim_win' }),
      playAgain: () => conn.current?.send({ type: 'play_again' }),
      retryConnection: () => {
        setReconnectAttempt((n) => n + 1)
        setConnectionLost(false)
        if (roomId.current) attach(roomId.current)
      },
    }),
    [room, view, remaining, roundResult, connectionLost, reconnectAttempt, playAgainReady, joinError, attach],
  )

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useGame(): GameCtx {
  const ctx = useContext(Ctx)
  if (!ctx) throw new Error('useGame outside provider')
  return ctx
}
