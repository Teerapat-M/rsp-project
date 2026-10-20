import type {
  AuthSession,
  HistoryPage,
  MatchDetail,
  Room,
  User,
  UserStats,
  WsConnection,
  WsEvent,
  AvatarKind,
} from '../types'

export type {
  AuthSession,
  HistoryPage,
  MatchDetail,
  Room,
  User,
  UserStats,
  WsConnection,
  WsEvent,
}

export type RegisterBody = {
  username: string
  password: string
  email?: string
}

export type LoginBody = {
  username: string
  password: string
  rememberMe: boolean
}

export type GuestBody = {
  displayName: string
  avatar: AvatarKind
  color: string
}

export type CreateRoomBody = {
  winsToFinish: number
  turnSeconds: number
}

export type ApiClient = unknown
