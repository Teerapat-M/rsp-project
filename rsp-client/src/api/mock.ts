import { AVATAR_SWATCHES } from '../theme'
import type {
  AuthSession,
  HistoryItem,
  HistoryPage,
  MatchDetail,
  Move,
  PlayableMove,
  PlayerView,
  Room,
  RoomPlayer,
  RoundResultView,
  User,
  UserStats,
  WsClientEvent,
  WsConnection,
  WsEvent,
} from '../types'
import { newMatch, ready, resolve as resolveMoves, submit, timeout, type LiveMatch } from '../game/rules'
import { uid } from '../lib/format'
import type { ApiClient, WsHandlers } from './client'
import type { CreateRoomBody, GuestBody, LoginBody, RegisterBody } from './contract'

const TOKEN_KEY = 'rsp.access'
const USER_KEY = 'rsp.user'
const WRONG = new Set(['482910', '000000'])
const FULL = new Set(['111111', '482915'])
const MOVES: PlayableMove[] = ['rock', 'paper', 'scissors']

type Err = Error & { code: string; passcode?: string }

function fail(code: string, message: string, passcode?: string): never {
  const e = new Error(message) as Err
  e.code = code
  e.passcode = passcode
  throw e
}

function asPlayer(u: User, isHost: boolean, joined = true): RoomPlayer {
  return {
    id: u.id,
    displayName: u.displayName,
    avatar: u.avatar,
    color: u.color,
    isHost,
    joined,
  }
}

const bell: User = {
  id: 'bot-bell',
  displayName: 'Bell',
  username: 'Bell',
  isGuest: false,
  avatar: 'scissors',
  color: '#A897C8',
}

const nok: User = {
  id: 'bot-nok',
  displayName: 'Nok',
  username: 'Nok',
  isGuest: false,
  avatar: 'paper',
  color: '#93BFE3',
}

const mint: User = {
  id: 'bot-mint',
  displayName: 'Mint',
  username: 'Mint',
  isGuest: false,
  avatar: 'rock',
  color: '#9DB89A',
}

const tee: User = {
  id: 'bot-tee',
  displayName: 'Tee',
  username: 'Tee',
  isGuest: false,
  avatar: 'paper',
  color: '#93BFE3',
}

const demoUser: User = {
  id: 'user-demo',
  displayName: 'Player123',
  username: 'Player123',
  email: 'player123@example.com',
  isGuest: false,
  avatar: 'rock',
  color: '#9DB89A',
}

const users = new Map<string, User>([
  [demoUser.id, demoUser],
  [bell.id, bell],
  [nok.id, nok],
  [mint.id, mint],
  [tee.id, tee],
])
const passwords = new Map<string, string>([['player123', 'password']])
const usersByName = new Map<string, User>([['player123', demoUser]])

type Runtime = {
  room: Room
  match: LiveMatch | null
  remaining: number | null
  opponentDc: boolean
  reconnectLeft: number | null
  playAgain: Set<string>
  tick: ReturnType<typeof setInterval> | null
  botTimer: ReturnType<typeof setTimeout> | null
  resultPause: ReturnType<typeof setTimeout> | null
  lastResult: RoundResultView | null
  pickAt: number
  forceLost: boolean
}

const rooms = new Map<string, Runtime>()
const listeners = new Map<string, Set<(e: WsEvent) => void>>()
const usedCodes = new Set<string>([...WRONG, ...FULL])

type StoreStats = UserStats & { history: MatchDetail[] }

function emptyStats(): StoreStats {
  return {
    matchesPlayed: 0,
    wins: 0,
    losses: 0,
    winRate: 0,
    bestStreak: 0,
    picks: { rock: 0, paper: 0, scissors: 0 },
    pickWins: { rock: 0, paper: 0, scissors: 0 },
    history: [],
  }
}

const statsByUser = new Map<string, StoreStats>()

function seedHistory(userId: string): void {
  const s = emptyStats()
  s.matchesPlayed = 35
  s.wins = 24
  s.losses = 11
  s.winRate = 69
  s.bestStreak = 6
  s.picks = { rock: 31, paper: 27, scissors: 42 }
  s.pickWins = { rock: 18, paper: 14, scissors: 26 }
  s.history = [
    detailOf('m-bell-1', bell, 3, 1, true, 'Today, 12:40', [
      ['rock', 'scissors', 'won', 4.2],
      ['paper', 'rock', 'won', 8.9],
      ['scissors', 'rock', 'lost', 11.3],
      ['rock', 'scissors', 'won', 3.7],
    ]),
    detailOf('m-nok-1', nok, 2, 3, false, 'Today, 11:05', [
      ['paper', 'scissors', 'lost', 5.1],
      ['rock', 'scissors', 'won', 3.4],
      ['scissors', 'rock', 'lost', 9.0],
      ['paper', 'paper', 'draw', 6.2],
      ['rock', 'paper', 'lost', 7.8],
    ]),
    detailOf('m-bell-2', bell, 3, 0, true, 'Yesterday, 21:18', [
      ['scissors', 'paper', 'won', 3.1],
      ['rock', 'scissors', 'won', 4.4],
      ['paper', 'rock', 'won', 2.9],
    ]),
    detailOf('m-mint-1', mint, 3, 2, true, 'Oct 15, 19:42', [
      ['rock', 'paper', 'lost', 8.1],
      ['scissors', 'paper', 'won', 4.0],
      ['paper', 'rock', 'won', 6.6],
      ['rock', 'rock', 'draw', 5.2],
      ['scissors', 'rock', 'won', 3.3],
    ]),
    detailOf('m-tee-1', tee, 1, 3, false, 'Oct 14, 20:10', [
      ['paper', 'scissors', 'lost', 10.2],
      ['rock', 'scissors', 'won', 4.8],
      ['scissors', 'rock', 'lost', 7.1],
      ['paper', 'scissors', 'lost', 9.4],
    ]),
  ]
  statsByUser.set(userId, s)
}

function detailOf(
  id: string,
  opp: User,
  youScore: number,
  oppScore: number,
  won: boolean,
  playedAt: string,
  rounds: Array<[Move, Move, 'won' | 'lost' | 'draw', number]>,
): MatchDetail {
  return {
    id,
    opponentName: opp.displayName,
    opponentAvatar: opp.avatar,
    opponentColor: opp.color,
    youScore,
    opponentScore: oppScore,
    won,
    playedAt,
    winsToFinish: 3,
    turnSeconds: 15,
    rounds: rounds.map((r, i) => ({
      number: i + 1,
      you: r[0],
      opponent: r[1],
      result: r[2],
      pickTimeSec: r[3],
    })),
  }
}

seedHistory(demoUser.id)

function currentUser(): User {
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) fail('UNAUTHORIZED', 'Not signed in')
  return JSON.parse(raw) as User
}

function persist(session: AuthSession): AuthSession {
  localStorage.setItem(TOKEN_KEY, session.accessToken)
  localStorage.setItem(USER_KEY, JSON.stringify(session.user))
  users.set(session.user.id, session.user)
  if (!statsByUser.has(session.user.id)) {
    const copy = structuredClone(statsByUser.get(demoUser.id) ?? emptyStats())
    statsByUser.set(session.user.id, copy)
  }
  return session
}

function sessionFor(user: User, remember = false): AuthSession {
  return persist({
    accessToken: uid('tok'),
    refreshToken: remember ? uid('ref') : undefined,
    user,
  })
}

function snapshotRoom(room: Room): Room {
  return { ...room, players: room.players.map((p) => ({ ...p })) }
}

function emit(roomId: string, event: WsEvent): void {
  listeners.get(roomId)?.forEach((fn) => fn(event))
}

function emitRoom(rt: Runtime): void {
  emit(rt.room.id, { type: 'room.updated', room: snapshotRoom(rt.room) })
}

function viewFor(rt: Runtime, user: User): PlayerView {
  const opp = rt.room.players.find((p) => p.id !== user.id)
  const m = rt.match
  const scores: Record<string, number> = {}
  if (m) {
    scores[m.players[0]] = m.scores[0]
    scores[m.players[1]] = m.scores[1]
  }
  const i = m ? (m.players[0] === user.id ? 0 : 1) : 0
  const yourMove = m?.moves[i] ?? ''
  return {
    roomId: rt.room.id,
    matchId: rt.room.id,
    status: m?.status ?? 'playing',
    roundNumber: m?.roundNo ?? 1,
    winsNeeded: rt.room.winsToFinish,
    turnSeconds: rt.room.turnSeconds,
    remainingSeconds: rt.remaining,
    scores,
    yourMove: yourMove === null ? '' : yourMove,
    yourReady: m ? m.ready[i] : false,
    opponentId: opp?.id ?? bell.id,
    opponentName: opp?.displayName ?? 'Bell',
    opponentAvatar: opp?.avatar ?? 'scissors',
    opponentColor: opp?.color ?? '#A897C8',
    opponentReady: m ? m.ready[1 - i] : false,
    opponentDisconnected: rt.opponentDc,
    reconnectSeconds: rt.reconnectLeft,
    winnerId: m && m.status === 'finished' ? (m.scores[0] >= m.winsNeeded ? m.players[0] : m.players[1]) : null,
    finished: Boolean(m && m.status === 'finished'),
    history:
      m?.history.map((h) => ({
        number: h.number,
        moves: { [m.players[0]]: h.moves[0], [m.players[1]]: h.moves[1] },
        autoPicked: { [m.players[0]]: h.autoPicked[0], [m.players[1]]: h.autoPicked[1] },
        winnerId: h.winnerId,
        draw: h.draw,
      })) ?? [],
    you: user,
  }
}

function stopTimers(rt: Runtime): void {
  if (rt.tick) clearInterval(rt.tick)
  if (rt.botTimer) clearTimeout(rt.botTimer)
  if (rt.resultPause) clearTimeout(rt.resultPause)
  rt.tick = rt.botTimer = rt.resultPause = null
}

function scheduleBot(rt: Runtime): void {
  if (rt.botTimer) clearTimeout(rt.botTimer)
  if (!rt.match || rt.match.status !== 'playing' || rt.opponentDc) return
  const delay = 1400 + Math.floor(Math.random() * 1800)
  rt.botTimer = setTimeout(() => {
    const m = rt.match
    if (!m || m.status !== 'playing' || rt.opponentDc) return
    const botId = m.players.find((id) => id === bell.id) ?? m.players[1]
    const i = m.players[0] === botId ? 0 : 1
    if (m.ready[i]) return
    const move = MOVES[Math.floor(Math.random() * MOVES.length)]
    try {
      submit(m, botId, move)
      const settled = ready(m, botId)
      const user = currentUser()
      emit(rt.room.id, { type: 'view', view: viewFor(rt, user) })
      if (settled) onSettled(rt)
    } catch {
      /* already ready / over */
    }
  }, delay)
}

function startClock(rt: Runtime): void {
  if (rt.tick) clearInterval(rt.tick)
  if (!rt.match || rt.match.status !== 'playing') return
  rt.remaining = rt.room.turnSeconds
  rt.pickAt = Date.now()
  emit(rt.room.id, { type: 'tick', remainingSeconds: rt.remaining })
  rt.tick = setInterval(() => {
    if (rt.remaining === null) return
    rt.remaining -= 1
    emit(rt.room.id, { type: 'tick', remainingSeconds: Math.max(0, rt.remaining) })
    if (rt.remaining <= 0) {
      if (rt.tick) clearInterval(rt.tick)
      rt.tick = null
      if (rt.match && rt.match.status === 'playing') {
        timeout(rt.match)
        onSettled(rt)
      }
    }
  }, 1000)
}

function onSettled(rt: Runtime): void {
  stopTimers(rt)
  const m = rt.match
  if (!m) return
  const last = m.history[m.history.length - 1]
  const user = currentUser()
  const i = m.players[0] === user.id ? 0 : 1
  const youWon = last.winnerId === user.id
  const pickSec = Math.max(0.4, (Date.now() - rt.pickAt) / 1000)
  const result: RoundResultView = {
    roundNumber: last.number,
    winsNeeded: m.winsNeeded,
    yourMove: last.moves[i],
    opponentMove: last.moves[1 - i],
    yourAutoPicked: last.autoPicked[i],
    opponentAutoPicked: last.autoPicked[1 - i],
    youWon,
    draw: last.draw,
    scores: { [m.players[0]]: m.scores[0], [m.players[1]]: m.scores[1] },
    nextRoundIn: 3,
    matchOver: m.status === 'finished',
  }
  rt.lastResult = result
  const view = viewFor(rt, user)
  const revealDelay = last.autoPicked[i] ? 1800 : 80
  rt.resultPause = setTimeout(() => {
    emit(rt.room.id, { type: 'round.resolved', result, view })
    if (m.status === 'finished') {
      persistMatch(user, rt, pickSec)
      rt.resultPause = setTimeout(() => {
        emit(rt.room.id, { type: 'match.finished', view: viewFor(rt, user) })
      }, 3000)
      return
    }
    rt.resultPause = setTimeout(() => {
      rt.lastResult = null
      startClock(rt)
      scheduleBot(rt)
      emit(rt.room.id, { type: 'view', view: viewFor(rt, currentUser()) })
    }, 3000)
  }, revealDelay)
}

function persistMatch(user: User, rt: Runtime, lastPick: number): void {
  const m = rt.match
  if (!m) return
  const i = m.players[0] === user.id ? 0 : 1
  const opp = rt.room.players.find((p) => p.id !== user.id) ?? asPlayer(bell, false)
  const youScore = m.scores[i]
  const oppScore = m.scores[1 - i]
  const won = youScore > oppScore
  const st = statsByUser.get(user.id) ?? emptyStats()
  st.matchesPlayed += 1
  if (won) st.wins += 1
  else st.losses += 1
  st.winRate = st.matchesPlayed ? Math.round((st.wins / st.matchesPlayed) * 100) : 0
  for (const h of m.history) {
    const mv = h.moves[i]
    if (mv === 'rock' || mv === 'paper' || mv === 'scissors') {
      st.picks[mv] += 1
      if (h.winnerId === user.id) st.pickWins[mv] += 1
    }
  }
  const detail: MatchDetail = {
    id: uid('m'),
    opponentName: opp.displayName,
    opponentAvatar: opp.avatar,
    opponentColor: opp.color,
    youScore,
    opponentScore: oppScore,
    won,
    playedAt: 'Just now',
    winsToFinish: m.winsNeeded,
    turnSeconds: m.turnSeconds,
    rounds: m.history.map((h, idx) => {
      const out = resolveMoves(h.moves[i], h.moves[1 - i])
      const result = out.draw ? 'draw' : out.winner === 0 ? 'won' : 'lost'
      return {
        number: h.number,
        you: h.moves[i],
        opponent: h.moves[1 - i],
        result,
        pickTimeSec: idx === m.history.length - 1 ? lastPick : 4 + idx,
        youAutoPicked: h.autoPicked[i],
        opponentAutoPicked: h.autoPicked[1 - i],
      }
    }),
  }
  st.history.unshift(detail)
  statsByUser.set(user.id, st)
}

function genCode(): string {
  for (let n = 0; n < 20; n++) {
    const c = String(Math.floor(Math.random() * 1_000_000)).padStart(6, '0')
    if (!usedCodes.has(c)) {
      usedCodes.add(c)
      return c
    }
  }
  return uid('c').slice(0, 6)
}

function ensureRoom(id: string): Runtime {
  const rt = rooms.get(id)
  if (!rt) fail('NOT_FOUND', 'Room not found')
  return rt
}

function beginMatch(rt: Runtime): void {
  const ids = rt.room.players.map((p) => p.id)
  if (ids.length < 2) fail('VALIDATION', 'Need two players')
  stopTimers(rt)
  rt.match = newMatch(ids[0], ids[1], rt.room.winsToFinish, rt.room.turnSeconds)
  rt.room.status = 'playing'
  rt.opponentDc = false
  rt.reconnectLeft = null
  rt.playAgain = new Set()
  rt.lastResult = null
  const user = currentUser()
  emitRoom(rt)
  emit(rt.room.id, { type: 'match.started', view: viewFor(rt, user) })
  startClock(rt)
  scheduleBot(rt)
}

export const mockApi: ApiClient = {
  async register(body: RegisterBody) {
    const username = body.username.trim()
    if (!/^[A-Za-z0-9_]{3,20}$/.test(username)) {
      fail('VALIDATION', 'Username must be 3–20 letters, numbers, or _')
    }
    if (body.password.length < 6) fail('VALIDATION', 'Password must be at least 6 characters')
    if (usersByName.has(username.toLowerCase())) fail('CONFLICT', 'Username is taken')
    const user: User = {
      id: uid('u'),
      displayName: username,
      username,
      email: body.email || undefined,
      isGuest: false,
      avatar: 'rock',
      color: AVATAR_SWATCHES[0],
    }
    usersByName.set(username.toLowerCase(), user)
    passwords.set(username.toLowerCase(), body.password)
    users.set(user.id, user)
    return sessionFor(user)
  },

  async login(body: LoginBody) {
    const name = body.username.trim().toLowerCase()
    const user = usersByName.get(name)
    const pass = passwords.get(name)
    if (!user || pass !== body.password) {
      if (name === 'player123') return sessionFor(demoUser, body.rememberMe)
      fail('UNAUTHORIZED', 'Wrong username or password')
    }
    return sessionFor(user, body.rememberMe)
  },

  async loginGoogle() {
    return sessionFor({ ...demoUser, email: 'player123@gmail.com' })
  },

  async loginGuest(body: GuestBody) {
    const name = body.displayName.trim()
    if (name.length < 3 || name.length > 16) fail('VALIDATION', 'Name must be 3–16 characters')
    const user: User = {
      id: uid('g'),
      displayName: name,
      isGuest: true,
      avatar: body.avatar,
      color: body.color,
    }
    users.set(user.id, user)
    return sessionFor(user)
  },

  async forgotPassword(email: string) {
    const resetUrl = `http://localhost:5173/reset?token=dev-${uid('rst')}&email=${encodeURIComponent(email)}`
    console.info('[mock] password reset URL', resetUrl)
    return {
      message: 'If that email is on an account, we sent a reset link.',
      resetUrl,
    }
  },

  async logout() {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
  },

  async me() {
    return currentUser()
  },

  async createRoom(body: CreateRoomBody) {
    const user = currentUser()
    if (body.winsToFinish < 1) fail('VALIDATION', 'wins_to_finish must be >= 1')
    if (body.turnSeconds < 1) fail('VALIDATION', 'turn_seconds must be >= 1')
    const room: Room = {
      id: uid('room'),
      passcode: genCode(),
      hostId: user.id,
      winsToFinish: body.winsToFinish,
      turnSeconds: body.turnSeconds,
      status: 'waiting',
      players: [asPlayer(user, true)],
    }
    const rt: Runtime = {
      room,
      match: null,
      remaining: null,
      opponentDc: false,
      reconnectLeft: null,
      playAgain: new Set(),
      tick: null,
      botTimer: null,
      resultPause: null,
      lastResult: null,
      pickAt: Date.now(),
      forceLost: false,
    }
    rooms.set(room.id, rt)
    setTimeout(() => {
      if (rt.room.status !== 'waiting') return
      if (rt.room.players.some((p) => p.id === bell.id)) return
      rt.room.players.push(asPlayer(bell, false))
      emitRoom(rt)
    }, 1200)
    return room
  },

  async joinRoom(passcode: string) {
    const code = passcode.replace(/\D/g, '')
    if (code.length !== 6) fail('VALIDATION', 'Passcode must be 6 digits')
    if (WRONG.has(code)) fail('WRONG_PASSCODE', `We couldn't find a room with passcode ${code}.`, code)
    if (FULL.has(code)) fail('ROOM_FULL', `Room ${code} already has 2 players.`, code)
    const user = currentUser()
    const room: Room = {
      id: uid('room'),
      passcode: code,
      hostId: bell.id,
      winsToFinish: 3,
      turnSeconds: 15,
      status: 'waiting',
      players: [asPlayer(bell, true), asPlayer(user, false)],
    }
    usedCodes.add(code)
    const rt: Runtime = {
      room,
      match: null,
      remaining: null,
      opponentDc: false,
      reconnectLeft: null,
      playAgain: new Set(),
      tick: null,
      botTimer: null,
      resultPause: null,
      lastResult: null,
      pickAt: Date.now(),
      forceLost: false,
    }
    rooms.set(room.id, rt)
    setTimeout(() => {
      if (rt.room.status === 'waiting') beginMatch(rt)
    }, 1600)
    return room
  },

  async leaveRoom(roomId: string) {
    const rt = rooms.get(roomId)
    if (!rt) return
    stopTimers(rt)
    rt.room.status = 'closed'
    emitRoom(rt)
  },

  async startGame(roomId: string) {
    const user = currentUser()
    const rt = ensureRoom(roomId)
    if (rt.room.hostId !== user.id) fail('VALIDATION', 'Only the host can start')
    if (rt.room.players.length < 2) fail('VALIDATION', 'Waiting for an opponent')
    beginMatch(rt)
  },

  async getRoom(roomId: string) {
    return ensureRoom(roomId).room
  },

  async getStats() {
    const user = currentUser()
    return statsByUser.get(user.id) ?? emptyStats()
  },

  async getHistory(cursor?: string): Promise<HistoryPage> {
    const user = currentUser()
    const st = statsByUser.get(user.id) ?? emptyStats()
    const start = cursor ? Number(cursor) : 0
    const items: HistoryItem[] = st.history.slice(start, start + 5)
    const next = start + 5 < st.history.length ? String(start + 5) : undefined
    return { items, total: Math.max(st.matchesPlayed, st.history.length), nextCursor: next }
  },

  async getMatch(id: string) {
    const user = currentUser()
    const st = statsByUser.get(user.id) ?? emptyStats()
    const found = st.history.find((h) => h.id === id)
    if (!found) fail('NOT_FOUND', 'Match not found')
    return found
  },

  connectMatch(roomId: string, handlers: WsHandlers): WsConnection {
    const set = listeners.get(roomId) ?? new Set()
    set.add(handlers.onEvent)
    listeners.set(roomId, set)
    const rt = rooms.get(roomId)
    if (rt) {
      queueMicrotask(() => {
        handlers.onEvent({ type: 'room.updated', room: snapshotRoom(rt.room) })
        if (rt.match) handlers.onEvent({ type: 'view', view: viewFor(rt, currentUser()) })
        if (rt.lastResult && rt.match) {
          handlers.onEvent({
            type: 'round.resolved',
            result: rt.lastResult,
            view: viewFor(rt, currentUser()),
          })
        }
        if (rt.forceLost) handlers.onEvent({ type: 'connection.lost' })
      })
    }
    return {
      send(event: WsClientEvent) {
        const runtime = rooms.get(roomId)
        if (!runtime) return
        handleClient(runtime, event, handlers)
      },
      close() {
        set.delete(handlers.onEvent)
      },
    }
  },

  async applyDemo(id: string) {
    const user = currentUser()
    if (id.startsWith('game') || id === 'round' || id === 'match' || id === 'connection') {
      const room = await mockApi.createRoom({ winsToFinish: 3, turnSeconds: 15 })
      const rt = ensureRoom(room.id)
      if (!rt.room.players.some((p) => p.id === bell.id)) rt.room.players.push(asPlayer(bell, false))
      rt.match = newMatch(user.id, bell.id, 3, 15)
      rt.room.status = 'playing'
      rt.match.scores = [1, 0]
      rt.match.roundNo = 2
      rt.remaining = 12
      if (id === 'game-placed') {
        submit(rt.match, user.id, 'scissors')
        rt.remaining = 7
      }
      if (id === 'game-ready') {
        submit(rt.match, user.id, 'scissors')
        ready(rt.match, user.id)
        rt.remaining = 5
      }
      if (id === 'game-timeup') {
        rt.remaining = 0
        rt.match.ready = [false, true]
        submit(rt.match, bell.id, 'rock')
      }
      if (id === 'game-dc') {
        rt.opponentDc = true
        rt.reconnectLeft = 24
        rt.remaining = null
      }
      if (id === 'round') {
        rt.match.scores = [2, 0]
        rt.lastResult = {
          roundNumber: 2,
          winsNeeded: 3,
          yourMove: 'rock',
          opponentMove: 'scissors',
          youWon: true,
          draw: false,
          scores: { [user.id]: 2, [bell.id]: 0 },
          nextRoundIn: 3,
          matchOver: false,
        }
      }
      if (id === 'match') {
        rt.match.status = 'finished'
        rt.match.scores = [3, 1]
        rt.match.history = [
          { number: 1, moves: ['rock', 'scissors'], autoPicked: [false, false], winnerId: user.id, draw: false },
          { number: 2, moves: ['paper', 'rock'], autoPicked: [false, true], winnerId: user.id, draw: false },
          { number: 3, moves: ['scissors', 'rock'], autoPicked: [false, false], winnerId: bell.id, draw: false },
          { number: 4, moves: ['rock', 'scissors'], autoPicked: [true, false], winnerId: user.id, draw: false },
        ]
      }
      if (id === 'connection') {
        rt.forceLost = true
      }
      return `/play?room=${room.id}&demo=${id}`
    }
    return `/${id}`
  },
}

function handleClient(rt: Runtime, event: WsClientEvent, handlers: WsHandlers): void {
  const user = currentUser()
  try {
    if (event.type === 'submit') {
      if (!rt.match) return
      submit(rt.match, user.id, event.move)
      emit(rt.room.id, { type: 'view', view: viewFor(rt, user) })
    }
    if (event.type === 'ready') {
      if (!rt.match) return
      const settled = ready(rt.match, user.id)
      emit(rt.room.id, { type: 'view', view: viewFor(rt, user) })
      if (settled) onSettled(rt)
    }
    if (event.type === 'claim_win') {
      if (!rt.match || !rt.opponentDc) return
      const i = rt.match.players[0] === user.id ? 0 : 1
      rt.match.scores[i] = rt.match.winsNeeded
      rt.match.status = 'finished'
      persistMatch(user, rt, 0)
      emit(rt.room.id, { type: 'match.finished', view: viewFor(rt, user) })
    }
    if (event.type === 'play_again') {
      rt.playAgain.add(user.id)
      emit(rt.room.id, { type: 'play_again.waiting', ready: rt.playAgain.size, need: 2 })
      setTimeout(() => {
        rt.playAgain.add(bell.id)
        emit(rt.room.id, { type: 'play_again.waiting', ready: 2, need: 2 })
        beginMatch(rt)
      }, 1100)
    }
    if (event.type === 'leave') {
      stopTimers(rt)
      rt.room.status = 'closed'
      emitRoom(rt)
    }
    if (event.type === 'simulate') {
      if (event.kind === 'opponent_dc') {
        rt.opponentDc = true
        rt.reconnectLeft = 24
        if (rt.tick) clearInterval(rt.tick)
        rt.tick = null
        rt.remaining = null
        emit(rt.room.id, { type: 'opponent.disconnected', view: viewFor(rt, user) })
      }
      if (event.kind === 'connection_lost') {
        handlers.onEvent({ type: 'connection.lost' })
      }
      if (event.kind === 'connection_restore') {
        handlers.onEvent({ type: 'connection.restored' })
      }
    }
  } catch (err) {
    handlers.onError?.(err instanceof Error ? err : new Error(String(err)))
  }
}
