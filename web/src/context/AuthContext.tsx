import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api } from '../api'
import { selfCheck } from '../game/rules'
import type { AvatarKind, User } from '../types'

if (import.meta.env.DEV) selfCheck()

type AuthCtx = {
  user: User | null
  ready: boolean
  login: (username: string, password: string, rememberMe: boolean) => Promise<User>
  register: (username: string, password: string, email?: string) => Promise<User>
  google: () => Promise<User>
  guest: (displayName: string, avatar: AvatarKind, color: string) => Promise<User>
  logout: () => Promise<void>
  refresh: () => Promise<void>
}

const Ctx = createContext<AuthCtx | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [ready, setReady] = useState(false)

  useEffect(() => {
    const raw = localStorage.getItem('rsp.user')
    if (raw) {
      try {
        setUser(JSON.parse(raw) as User)
      } catch {
        localStorage.removeItem('rsp.user')
      }
    }
    setReady(true)
  }, [])

  const value = useMemo<AuthCtx>(
    () => ({
      user,
      ready,
      login: async (username, password, rememberMe) => {
        const s = await api.login({ username, password, rememberMe })
        setUser(s.user)
        return s.user
      },
      register: async (username, password, email) => {
        const s = await api.register({ username, password, email })
        setUser(s.user)
        return s.user
      },
      google: async () => {
        const s = await api.loginGoogle()
        setUser(s.user)
        return s.user
      },
      guest: async (displayName, avatar, color) => {
        const s = await api.loginGuest({ displayName, avatar, color })
        setUser(s.user)
        return s.user
      },
      logout: async () => {
        await api.logout()
        setUser(null)
      },
      refresh: async () => {
        try {
          setUser(await api.me())
        } catch {
          setUser(null)
        }
      },
    }),
    [user, ready],
  )

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export function useAuth(): AuthCtx {
  const ctx = useContext(Ctx)
  if (!ctx) throw new Error('useAuth outside provider')
  return ctx
}
