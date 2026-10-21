import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { api } from '../api'
import { secondsUntil, watchNextRound } from '../lib/nextRound'
import { writeActiveRoom } from '../lib/playRoom'
import { RECONNECT_GRACE_SEC, RECONNECT_MAX_ATTEMPTS, reconnectDelayMs } from '../lib/reconnect'
import { applyLobbyReady, applyRoomSnapshot } from '../lib/roomSync'
import type { PlayableMove, PlayerView, Room, RoundResultView, WsConnection } from '../types'

type JoinError = { code: 'WRONG_PASSCODE' | 'ROOM_FULL'; passcode: string; message: string } | null

type GameCtx = {
  room: Room | null
  view: PlayerView | null
  remaining: number | null
  roundResult: RoundResultView | null
  nextRoundLeft: number | null
  connectionLost: boolean
  reconnectAttempt: number
  reconnectGraceLeft: number
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
  setLobbyReady: (ready: boolean) => void
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
  const [nextRoundLeft, setNextRoundLeft] = useState<number | null>(null)
  const [connectionLost, setConnectionLost] = useState(false)
  const [reconnectAttempt, setReconnectAttempt] = useState(0)
  const [reconnectGraceLeft, setReconnectGraceLeft] = useState(RECONNECT_GRACE_SEC)
  const [playAgainReady, setPlayAgainReady] = useState(0)
  const [joinError, setJoinError] = useState<JoinError>(null)
  const conn = useRef<WsConnection | null>(null)
  const roomId = useRef<string | null>(null)
  const retryRef = useRef(0)
  const retryTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const attachRef = useRef<(id: string) => void>(() => {})
  const liveRef = useRef(false)
  const connectingRef = useRef(false)

  const attach = useCallback((id: string) => {
    if (roomId.current === id && conn.current && (liveRef.current || connectingRef.current)) {
      writeActiveRoom(id)
      return
    }
    conn.current?.close()
    roomId.current = id
    liveRef.current = false
    connectingRef.current = true
    writeActiveRoom(id)
    conn.current = api.connectMatch(id, {
      onEvent: (ev) => {
        if (ev.type === 'room.updated') {
          setRoom((prev) => applyRoomSnapshot(prev, ev.room))
        }
        if (ev.type === 'lobby.ready') {
          setRoom((prev) => applyLobbyReady(prev, ev))
        }
        if (ev.type === 'match.started') {
          if (roomId.current) writeActiveRoom(roomId.current)
          setView(ev.view)
          setRoundResult(null)
          setNextRoundLeft(null)
          setRemaining(ev.view.remainingSeconds)
          setPlayAgainReady(0)
        }
        if (ev.type === 'tick') setRemaining(ev.remainingSeconds)
        if (ev.type === 'view') {
          setView(ev.view)
          setRemaining(ev.view.remainingSeconds)
        }
        if (ev.type === 'round.resolved') {
          const result = {
            ...ev.result,
            nextRoundAt: ev.result.nextRoundAt ?? ev.nextRoundAt,
          }
          setRoundResult(result)
          setView(ev.view)
          setNextRoundLeft(secondsUntil(result.nextRoundAt, Date.now(), result.nextRoundIn))
        }
        if (ev.type === 'round.started') {
          setView(ev.view)
          setRoundResult(null)
          setNextRoundLeft(null)
          setRemaining(ev.view.remainingSeconds)
        }
        if (ev.type === 'match.finished') {
          writeActiveRoom(null)
          setView(ev.view)
          setRoundResult(null)
          setNextRoundLeft(null)
        }
        if (ev.type === 'opponent.disconnected' || ev.type === 'opponent.reconnected') {
          setView(ev.view)
        }
        if (ev.type === 'connection.lost') {
          liveRef.current = false
          connectingRef.current = false
          setConnectionLost(true)
          setReconnectGraceLeft(RECONNECT_GRACE_SEC)
          const next = retryRef.current + 1
          retryRef.current = next
          setReconnectAttempt(next)
          if (retryTimer.current) clearTimeout(retryTimer.current)
          if (next <= RECONNECT_MAX_ATTEMPTS && roomId.current) {
            const rid = roomId.current
            retryTimer.current = setTimeout(() => attachRef.current(rid), reconnectDelayMs(next - 1))
          }
        }
        if (ev.type === 'connection.restored') {
          liveRef.current = true
          connectingRef.current = false
          setConnectionLost(false)
          setReconnectGraceLeft(RECONNECT_GRACE_SEC)
          retryRef.current = 0
          setReconnectAttempt(0)
          if (retryTimer.current) {
            clearTimeout(retryTimer.current)
            retryTimer.current = null
          }
        }
        if (ev.type === 'play_again.waiting') setPlayAgainReady(ev.ready)
      },
    })
  }, [])

  useEffect(() => {
    attachRef.current = attach
  })

  useEffect(() => {
    if (!connectionLost) return
    const id = setInterval(() => {
      setReconnectGraceLeft((n) => Math.max(0, n - 1))
    }, 1000)
    return () => clearInterval(id)
  }, [connectionLost])

  useEffect(
    () => () => {
      if (retryTimer.current) clearTimeout(retryTimer.current)
      conn.current?.close()
    },
    [],
  )

  // Server broadcasts round.started / match.finished after the result pause.
  // Tick from nextRoundAt so both clients show the same 3, 2, 1. If that event
  // is late, drop the overlay when the deadline passes.
  useEffect(() => {
    if (!roundResult) return
    const roundNo = roundResult.roundNumber
    return watchNextRound({
      nextRoundAt: roundResult.nextRoundAt,
      nextRoundIn: roundResult.nextRoundIn,
      onTick: setNextRoundLeft,
      onDone: () => {
        setRoundResult((cur) => (cur && cur.roundNumber === roundNo ? null : cur))
        setNextRoundLeft(null)
      },
    })
  }, [roundResult])

  const value = useMemo<GameCtx>(
    () => ({
      room,
      view,
      remaining,
      roundResult,
      nextRoundLeft,
      connectionLost,
      reconnectAttempt,
      reconnectGraceLeft,
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
        const id = roomId.current
        roomId.current = null
        liveRef.current = false
        connectingRef.current = false
        writeActiveRoom(null)
        if (id) await api.leaveRoom(id)
        conn.current?.close()
        conn.current = null
        setRoom(null)
        setView(null)
        setRoundResult(null)
        setNextRoundLeft(null)
      },
      startGame: async () => {
        if (room) await api.startGame(room.id)
      },
      attach,
      loadRoom: async (id) => {
        const r = await api.getRoom(id)
        setRoom((prev) => applyRoomSnapshot(prev, r))
        attach(id)
        return r
      },
      submitMove: (move) => conn.current?.send({ type: 'submit', move }),
      readyUp: () => conn.current?.send({ type: 'ready' }),
      setLobbyReady: (ready) => conn.current?.send({ type: ready ? 'lobby.ready' : 'lobby.unready' }),
      claimWin: () => conn.current?.send({ type: 'claim_win' }),
      playAgain: () => conn.current?.send({ type: 'play_again' }),
      retryConnection: () => {
        liveRef.current = false
        connectingRef.current = false
        retryRef.current = 0
        setReconnectAttempt(0)
        setConnectionLost(false)
        if (roomId.current) attach(roomId.current)
      },
    }),
    [room, view, remaining, roundResult, nextRoundLeft, connectionLost, reconnectAttempt, reconnectGraceLeft, playAgainReady, joinError, attach],
  )

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useGame(): GameCtx {
  const ctx = useContext(Ctx)
  if (!ctx) throw new Error('useGame outside provider')
  return ctx
}
