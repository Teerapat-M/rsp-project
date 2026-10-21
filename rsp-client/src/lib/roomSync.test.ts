import { describe, expect, it } from 'vitest'
import { applyLobbyReady, applyRoomSnapshot, isStaleSeq } from './roomSync'
import type { Room, RoomPlayer } from '../types'

function player(id: string, ready = false): RoomPlayer {
  return {
    id,
    displayName: id,
    avatar: 'rock',
    color: '#000',
    isHost: id === 'a',
    joined: true,
    ready,
  }
}

function room(seq: number, readyA: boolean, readyB: boolean): Room {
  return {
    id: 'r1',
    passcode: '123456',
    hostId: 'a',
    winsToFinish: 3,
    turnSeconds: 15,
    status: 'waiting',
    seq,
    players: [player('a', readyA), player('b', readyB)],
  }
}

describe('isStaleSeq', () => {
  it('drops older frames and keeps equal/newer', () => {
    expect(isStaleSeq(3, 2)).toBe(true)
    expect(isStaleSeq(3, 3)).toBe(false)
    expect(isStaleSeq(3, 4)).toBe(false)
    expect(isStaleSeq(undefined, 1)).toBe(false)
  })
})

describe('applyRoomSnapshot', () => {
  it('skips a stale dual-socket snapshot after 2/2', () => {
    const both = applyRoomSnapshot(null, room(2, true, true))
    const stale = applyRoomSnapshot(both, room(1, true, false))
    expect(stale.players.map((p) => p.ready)).toEqual([true, true])
    expect(stale.seq).toBe(2)
  })
})

describe('applyLobbyReady', () => {
  it('patches the second ready and ignores a late 1/2 frame', () => {
    let cur: Room | null = room(1, true, false)
    cur = applyLobbyReady(cur, { playerId: 'b', playerReady: true, seq: 2 })
    expect(cur?.players.map((p) => p.ready)).toEqual([true, true])
    cur = applyLobbyReady(cur, { playerId: 'a', playerReady: true, seq: 1 })
    expect(cur?.players.map((p) => p.ready)).toEqual([true, true])
    cur = applyRoomSnapshot(cur, room(1, true, false))
    expect(cur.players.map((p) => p.ready)).toEqual([true, true])
  })
})
