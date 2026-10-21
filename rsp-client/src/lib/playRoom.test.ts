import { describe, expect, it } from 'vitest'
import { ACTIVE_ROOM_KEY, playPath, readActiveRoom, resolvePlayRoom, writeActiveRoom } from './playRoom'

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
  it('round-trips and clears', () => {
    const s = memStorage()
    writeActiveRoom('room-1', s)
    expect(s.data[ACTIVE_ROOM_KEY]).toBe('room-1')
    expect(readActiveRoom(s)).toBe('room-1')
    writeActiveRoom(null, s)
    expect(readActiveRoom(s)).toBe(null)
  })
})
