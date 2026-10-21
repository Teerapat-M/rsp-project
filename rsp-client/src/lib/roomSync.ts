import type { Room } from '../types'

export type LobbyReadyPatch = {
  playerId: string
  playerReady: boolean
  seq?: number
}

export function isStaleSeq(prevSeq: number | undefined, nextSeq: number | undefined): boolean {
  return Boolean(prevSeq && nextSeq && nextSeq < prevSeq)
}

export function applyRoomSnapshot(prev: Room | null, next: Room): Room {
  if (prev && isStaleSeq(prev.seq, next.seq)) return prev
  return { ...next, players: next.players.map((p) => ({ ...p })) }
}

export function applyLobbyReady(prev: Room | null, patch: LobbyReadyPatch): Room | null {
  if (!prev) return prev
  if (isStaleSeq(prev.seq, patch.seq)) return prev
  const seq = patch.seq && patch.seq > (prev.seq ?? 0) ? patch.seq : prev.seq
  return {
    ...prev,
    seq,
    players: prev.players.map((p) => (p.id === patch.playerId ? { ...p, ready: patch.playerReady } : p)),
  }
}
