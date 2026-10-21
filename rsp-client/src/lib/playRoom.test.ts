import { describe, expect, it } from 'vitest'
import {
  ACTIVE_ROOM_KEY,
  activeRoomKey,
  playPath,
  readActiveRoom,
  resolvePlayRoom,
  writeActiveRoom,
} from './playRoom'

function memStorage(init?: Record<string, string>) {
  const data = { ...(init ?? {}) }
  return {
    getItem: (k: string) => (k in data ? data[k] : null),
    setItem: (k: string, v: string) => {
      data[k] = v
    },
    removeItem: (k: string) => {
      delete data[k]
    },
    data,
  }
}

describe('playPath', () => {
  it('puts the room on the query string', () => {
    expect(playPath('abc-123')).toBe('/play?room=abc-123')
  })
})

describe('resolvePlayRoom', () => {
  it('prefers the URL, then storage, then live state', () => {
    expect(resolvePlayRoom({ query: 'from-url', stored: 'from-store', live: 'from-mem' })).toBe('from-url')
    expect(resolvePlayRoom({ query: '', stored: 'from-store', live: 'from-mem' })).toBe('from-store')
    expect(resolvePlayRoom({ query: null, stored: null, live: 'from-mem' })).toBe('from-mem')
    expect(resolvePlayRoom({ query: '  ', stored: '', live: null })).toBe(null)
  })
})

describe('active room storage', () => {
  it('keys by user id so a new tab can restore', () => {
    const s = memStorage()
    writeActiveRoom('room-1', { userId: 'u1', storage: s })
    expect(s.data[activeRoomKey('u1')]).toBe('room-1')
    expect(s.data[ACTIVE_ROOM_KEY]).toBeUndefined()
    expect(readActiveRoom({ userId: 'u1', storage: s })).toBe('room-1')
    expect(readActiveRoom({ userId: 'u2', storage: s })).toBe(null)
    writeActiveRoom(null, { userId: 'u1', storage: s })
    expect(readActiveRoom({ userId: 'u1', storage: s })).toBe(null)
  })
})
