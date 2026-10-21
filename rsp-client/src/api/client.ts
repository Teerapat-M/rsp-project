import type {
  AuthSession,
  CreateRoomBody,
  GuestBody,
  HistoryPage,
  LoginBody,
  MatchDetail,
  RegisterBody,
  Room,
  User,
  UserStats,
  WsConnection,
  WsEvent,
} from './contract'

export type { CreateRoomBody, GuestBody, LoginBody, RegisterBody } from './contract'

export type WsHandlers = {
  onEvent: (event: WsEvent) => void
  onError?: (err: Error) => void
}

export type ApiClient = {
  register: (body: RegisterBody) => Promise<AuthSession>
  login: (body: LoginBody) => Promise<AuthSession>
  loginGoogle: () => Promise<AuthSession>
  loginGuest: (body: GuestBody) => Promise<AuthSession>
  forgotPassword: (email: string) => Promise<{ message: string; resetUrl?: string }>
  logout: () => Promise<void>
  me: () => Promise<User>
  createRoom: (body: CreateRoomBody) => Promise<Room>
  joinRoom: (passcode: string) => Promise<Room>
  leaveRoom: (roomId: string) => Promise<void>
  startGame: (roomId: string) => Promise<void>
  getRoom: (roomId: string) => Promise<Room>
  getStats: () => Promise<UserStats>
  getHistory: (cursor?: string) => Promise<HistoryPage>
  getMatch: (id: string) => Promise<MatchDetail>
  connectMatch: (roomId: string, handlers: WsHandlers) => WsConnection
  applyDemo?: (id: string) => Promise<string>
}

export function isMock(): boolean {
  return import.meta.env.VITE_USE_MOCK !== 'false'
}
