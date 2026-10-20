import type {
  AuthSession,
  HistoryPage,
  MatchDetail,
  Room,
  User,
  UserStats,
  WsConnection,
  WsEvent,
} from '../types'
import type { ApiClient, WsHandlers } from './client'
import type { CreateRoomBody, GuestBody, LoginBody, RegisterBody } from './contract'

const base = import.meta.env.VITE_API_BASE ?? '/api'

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const token = localStorage.getItem('rsp.access')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(init?.headers as Record<string, string> | undefined),
  }
  if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(`${base}${path}`, { ...init, headers })
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { code?: string; message?: string }
    const err = new Error(body.message ?? res.statusText) as Error & { code: string }
    err.code = body.code ?? 'HTTP'
    throw err
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

function persist(session: AuthSession): AuthSession {
  localStorage.setItem('rsp.access', session.accessToken)
  localStorage.setItem('rsp.user', JSON.stringify(session.user))
  return session
}

export const httpApi: ApiClient = {
  register: (body: RegisterBody) => req<AuthSession>('/auth/register', { method: 'POST', body: JSON.stringify(body) }).then(persist),
  login: (body: LoginBody) => req<AuthSession>('/auth/login', { method: 'POST', body: JSON.stringify(body) }).then(persist),
  loginGoogle: () => req<AuthSession>('/auth/google', { method: 'POST' }).then(persist),
  loginGuest: (body: GuestBody) => req<AuthSession>('/auth/guest', { method: 'POST', body: JSON.stringify(body) }).then(persist),
  forgotPassword: (email: string) =>
    req<{ message: string; resetUrl?: string }>('/auth/forgot', { method: 'POST', body: JSON.stringify({ email }) }),
  logout: async () => {
    await req('/auth/logout', { method: 'POST' }).catch(() => undefined)
    localStorage.removeItem('rsp.access')
    localStorage.removeItem('rsp.user')
  },
  me: () => req<User>('/auth/me'),
  createRoom: (body: CreateRoomBody) => req<Room>('/rooms', { method: 'POST', body: JSON.stringify(body) }),
  joinRoom: (passcode: string) => req<Room>('/rooms/join', { method: 'POST', body: JSON.stringify({ passcode }) }),
  leaveRoom: (roomId: string) => req<void>(`/rooms/${roomId}/leave`, { method: 'POST' }),
  startGame: (roomId: string) => req<void>(`/rooms/${roomId}/start`, { method: 'POST' }),
  getRoom: (roomId: string) => req<Room>(`/rooms/${roomId}`),
  getStats: () => req<UserStats>('/stats'),
  getHistory: (cursor?: string) => req<HistoryPage>(`/matches${cursor ? `?cursor=${cursor}` : ''}`),
  getMatch: (id: string) => req<MatchDetail>(`/matches/${id}`),
  connectMatch(roomId: string, handlers: WsHandlers): WsConnection {
    const token = localStorage.getItem('rsp.access') ?? ''
    const wsUrl = import.meta.env.VITE_WS_URL ?? 'ws://localhost:8080/ws'
    const ws = new WebSocket(`${wsUrl}?token=${encodeURIComponent(token)}&room=${encodeURIComponent(roomId)}`)
    ws.onmessage = (ev) => {
      try {
        handlers.onEvent(JSON.parse(String(ev.data)) as WsEvent)
      } catch (err) {
        handlers.onError?.(err instanceof Error ? err : new Error(String(err)))
      }
    }
    ws.onerror = () => handlers.onError?.(new Error('WebSocket error'))
    return {
      send: (event) => {
        if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify(event))
      },
      close: () => ws.close(),
    }
  },
}
