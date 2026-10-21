export type Move = 'rock' | 'paper' | 'scissors' | 'timeout'
export type PlayableMove = 'rock' | 'paper' | 'scissors'
export type AvatarKind = PlayableMove

export type User = {
  id: string
  displayName: string
  username?: string
  email?: string
  isGuest: boolean
  avatar: AvatarKind
  color: string
}

export type AuthSession = {
  accessToken: string
  refreshToken?: string
  user: User
}

export type RoomStatus = 'waiting' | 'playing' | 'closed'

export type RoomPlayer = {
  id: string
  displayName: string
  avatar: AvatarKind
  color: string
  isHost: boolean
  joined: boolean
  ready?: boolean
  connected?: boolean
}

export type Room = {
  id: string
  passcode: string
  hostId: string
  winsToFinish: number
  turnSeconds: number
  status: RoomStatus
  players: RoomPlayer[]
  seq?: number
}

export type MatchStatus = 'playing' | 'finished'

export type ResolvedRound = {
  number: number
  moves: Record<string, Move>
  autoPicked?: Record<string, boolean>
  winnerId: string | null
  draw: boolean
  pickTimeYouSec?: number
}

export type PlayerView = {
  roomId: string
  matchId: string
  status: MatchStatus
  roundNumber: number
  winsNeeded: number
  turnSeconds: number
  remainingSeconds: number | null
  scores: Record<string, number>
  yourMove: Move | ''
  yourReady: boolean
  opponentId: string
  opponentName: string
  opponentAvatar: AvatarKind
  opponentColor: string
  opponentReady: boolean
  opponentDisconnected: boolean
  reconnectSeconds: number | null
  winnerId: string | null
  finished: boolean
  history: ResolvedRound[]
  you: User
}

export type RoundResultView = {
  roundNumber: number
  winsNeeded: number
  yourMove: Move
  opponentMove: Move
  yourAutoPicked?: boolean
  opponentAutoPicked?: boolean
  youWon: boolean
  draw: boolean
  scores: Record<string, number>
  nextRoundIn: number
  nextRoundAt?: string
  matchOver: boolean
}

export type UserStats = {
  matchesPlayed: number
  wins: number
  losses: number
  draws?: number
  winRate: number
  bestStreak: number
  picks: Record<PlayableMove, number>
  pickWins: Record<PlayableMove, number>
}

export type HistoryItem = {
  id: string
  opponentName: string
  opponentAvatar: AvatarKind
  opponentColor: string
  youScore: number
  opponentScore: number
  won: boolean
  playedAt: string
  winsToFinish: number
  turnSeconds: number
}

export type HistoryPage = {
  items: HistoryItem[]
  total: number
  nextCursor?: string
}

export type MatchDetail = HistoryItem & {
  rounds: Array<{
    number: number
    you: Move
    opponent: Move
    result: 'won' | 'lost' | 'draw'
    pickTimeSec: number
    youAutoPicked?: boolean
    opponentAutoPicked?: boolean
  }>
}

export type ApiErrorCode =
  | 'UNAUTHORIZED'
  | 'NOT_FOUND'
  | 'ROOM_FULL'
  | 'WRONG_PASSCODE'
  | 'VALIDATION'
  | 'CONFLICT'

export type ApiError = {
  code: ApiErrorCode
  message: string
  passcode?: string
}

export type WsEvent =
  | { type: 'room.updated'; room: Room }
  | { type: 'lobby.ready'; playerId: string; playerReady: boolean; ready: number; need: number; seq?: number }
  | { type: 'match.started'; view: PlayerView }
  | { type: 'tick'; remainingSeconds: number }
  | { type: 'view'; view: PlayerView }
  | { type: 'round.resolved'; result: RoundResultView; view: PlayerView; nextRoundAt?: string }
  | { type: 'round.started'; view: PlayerView }
  | { type: 'match.finished'; view: PlayerView }
  | { type: 'opponent.disconnected'; view: PlayerView }
  | { type: 'opponent.reconnected'; view: PlayerView }
  | { type: 'connection.lost' }
  | { type: 'connection.restored' }
  | { type: 'play_again.waiting'; ready: number; need: number }

export type WsClientEvent =
  | { type: 'submit'; move: PlayableMove }
  | { type: 'ready' }
  | { type: 'lobby.ready' }
  | { type: 'lobby.unready' }
  | { type: 'claim_win' }
  | { type: 'play_again' }
  | { type: 'leave' }

export type WsConnection = {
  send: (event: WsClientEvent) => void
  close: () => void
}
