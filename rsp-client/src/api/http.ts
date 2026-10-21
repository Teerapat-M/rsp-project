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

type ApiErr = Error & { code: string; passcode?: string }

function persist(session: AuthSession): AuthSession {
  localStorage.setItem('rsp.access', session.accessToken)
  localStorage.setItem('rsp.user', JSON.stringify(session.user))
  if (session.refreshToken) localStorage.setItem('rsp.refresh', session.refreshToken)
  return session
}

function clearSession(): void {
  localStorage.removeItem('rsp.access')
  localStorage.removeItem('rsp.user')
  localStorage.removeItem('rsp.refresh')
}

function skipRefresh(path: string): boolean {
  return (
    path.startsWith('/auth/login') ||
    path.startsWith('/auth/register') ||
    path.startsWith('/auth/refresh') ||
    path.startsWith('/auth/guest') ||
    path.startsWith('/auth/google') ||
    path.startsWith('/auth/forgot') ||
    path.startsWith('/auth/reset')
  )
}

async function refreshAccess(): Promise<boolean> {
  const refresh = localStorage.getItem('rsp.refresh')
  if (!refresh) return false
  const res = await fetch(`${base}/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refreshToken: refresh }),
  })
  if (!res.ok) {
    clearSession()
    return false
  }
  persist((await res.json()) as AuthSession)
  return true
}

async function req<T>(path: string, init?: RequestInit, retried = false): Promise<T> {
  const token = localStorage.getItem('rsp.access')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(init?.headers as Record<string, string> | undefined),
  }
  if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(`${base}${path}`, { ...init, headers })
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { code?: string; message?: string; passcode?: string }
    if (res.status === 401 && !retried && !skipRefresh(path) && (await refreshAccess())) {
      return req<T>(path, init, true)
    }
    const err = new Error(body.message ?? res.statusText) as ApiErr
    err.code = body.code ?? 'HTTP'
    err.passcode = body.passcode
    throw err
  }
  if (res.status === 204) return undefined as T
  return (await res.json()) as T
}

function wsURL(): string {
  if (import.meta.env.VITE_WS_URL) return import.meta.env.VITE_WS_URL
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${window.location.host}/ws`
}

export const httpApi: ApiClient = {
  register: (body: RegisterBody) => req<AuthSession>('/auth/register', { method: 'POST', body: JSON.stringify(body) }).then(persist),
  login: (body: LoginBody) => req<AuthSession>('/auth/login', { method: 'POST', body: JSON.stringify(body) }).then(persist),
  loginGoogle: (idToken?: string) =>
    req<AuthSession>('/auth/google', { method: 'POST', body: JSON.stringify({ idToken }) }).then(persist),
  loginGuest: (body: GuestBody) => req<AuthSession>('/auth/guest', { method: 'POST', body: JSON.stringify(body) }).then(persist),
  forgotPassword: (email: string) =>
    req<{ message: string; resetUrl?: string }>('/auth/forgot', { method: 'POST', body: JSON.stringify({ email }) }),
  resetPassword: (token: string, password: string) =>
    req<void>('/auth/reset', { method: 'POST', body: JSON.stringify({ token, password }) }),
  upgrade: (body: RegisterBody) => req<AuthSession>('/auth/upgrade', { method: 'POST', body: JSON.stringify(body) }).then(persist),
  logout: async () => {
    await req('/auth/logout', {
      method: 'POST',
      body: JSON.stringify({ refreshToken: localStorage.getItem('rsp.refresh') ?? '' }),
    }).catch(() => undefined)
    clearSession()
  },
  me: () => req<User>('/auth/me'),
  createRoom: (body: CreateRoomBody) => req<Room>('/rooms', { method: 'POST', body: JSON.stringify(body) }),
  joinRoom: (passcode: string) => req<Room>('/rooms/join', { method: 'POST', body: JSON.stringify({ passcode }) }),
  leaveRoom: (roomId: string) => req<void>(`/rooms/${roomId}/leave`, { method: 'POST' }),
  startGame: (roomId: string) => req<void>(`/rooms/${roomId}/start`, { method: 'POST' }),
  getRoom: (roomId: string) => req<Room>(`/rooms/${roomId}`),
  getStats: () => req<UserStats>('/stats'),
  getHistory: (cursor?: string) => req<HistoryPage>(`/matches${cursor ? `?cursor=${encodeURIComponent(cursor)}` : ''}`),
  getMatch: (id: string) => req<MatchDetail>(`/matches/${id}`),
  connectMatch(roomId: string, handlers: WsHandlers): WsConnection {
    const token = localStorage.getItem('rsp.access') ?? ''
    const ws = new WebSocket(`${wsURL()}?token=${encodeURIComponent(token)}&room=${encodeURIComponent(roomId)}`)
    let closedByUs = false
    ws.onmessage = (ev) => {
      try {
        handlers.onEvent(JSON.parse(String(ev.data)) as WsEvent)
      } catch (err) {
        handlers.onError?.(err instanceof Error ? err : new Error(String(err)))
      }
    }
    ws.onerror = () => handlers.onError?.(new Error('WebSocket error'))
    ws.onclose = () => {
      if (!closedByUs) handlers.onEvent({ type: 'connection.lost' })
    }
    return {
      send: (event) => {
        if (ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify(event))
      },
      close: () => {
        closedByUs = true
        ws.close()
      },
    }
  },
}
