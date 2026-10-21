import type { Move, PlayableMove } from '../types'

export function playable(move: Move): move is PlayableMove {
  return move === 'rock' || move === 'paper' || move === 'scissors'
}

function beats(a: Move, b: Move): boolean {
  return (
    (a === 'rock' && b === 'scissors') ||
    (a === 'paper' && b === 'rock') ||
    (a === 'scissors' && b === 'paper')
  )
}

export type Outcome = { draw: true; winner: null } | { draw: false; winner: 0 | 1 }

export function resolve(a: Move, b: Move): Outcome {
  if (a === b) return { draw: true, winner: null }
  if (a === 'timeout') return { draw: false, winner: 1 }
  if (b === 'timeout') return { draw: false, winner: 0 }
  if (beats(a, b)) return { draw: false, winner: 0 }
  return { draw: false, winner: 1 }
}

export type LiveMatch = {
  winsNeeded: number
  turnSeconds: number
  players: [string, string]
  scores: [number, number]
  roundNo: number
  status: 'playing' | 'finished'
  moves: [Move | null, Move | null]
  ready: [boolean, boolean]
  history: Array<{
    number: number
    moves: [Move, Move]
    winnerId: string | null
    draw: boolean
  }>
  pickStartedAt: number
}

export function newMatch(
  a: string,
  b: string,
  winsNeeded: number,
  turnSeconds: number,
): LiveMatch {
  return {
    winsNeeded,
    turnSeconds,
    players: [a, b],
    scores: [0, 0],
    roundNo: 1,
    status: 'playing',
    moves: [null, null],
    ready: [false, false],
    history: [],
    pickStartedAt: Date.now(),
  }
}

function side(m: LiveMatch, id: string): 0 | 1 {
  if (m.players[0] === id) return 0
  if (m.players[1] === id) return 1
  throw new Error('unknown player')
}

export function submit(m: LiveMatch, id: string, move: PlayableMove): void {
  if (m.status !== 'playing') throw new Error('match over')
  const i = side(m, id)
  if (m.ready[i]) throw new Error('already ready')
  m.moves[i] = move
}

export function ready(m: LiveMatch, id: string): boolean {
  if (m.status !== 'playing') throw new Error('match over')
  const i = side(m, id)
  if (m.ready[i]) throw new Error('already ready')
  if (!m.moves[i]) throw new Error('no move')
  m.ready[i] = true
  if (m.ready[0] && m.ready[1]) {
    settle(m)
    return true
  }
  return false
}

export function timeout(m: LiveMatch): void {
  if (m.status !== 'playing') return
  settle(m)
}

function settle(m: LiveMatch): void {
  const pair: [Move, Move] = [m.moves[0] ?? 'timeout', m.moves[1] ?? 'timeout']
  const out = resolve(pair[0], pair[1])
  let winnerId: string | null = null
  if (!out.draw) {
    winnerId = m.players[out.winner]
    m.scores[out.winner] += 1
  }
  m.history.push({
    number: m.roundNo,
    moves: pair,
    winnerId,
    draw: out.draw,
  })
  if (m.scores[0] >= m.winsNeeded || m.scores[1] >= m.winsNeeded) {
    m.status = 'finished'
  } else {
    m.roundNo += 1
    m.moves = [null, null]
    m.ready = [false, false]
    m.pickStartedAt = Date.now()
  }
}

export function selfCheck(): void {
  const r1 = resolve('rock', 'scissors')
  if (r1.draw || r1.winner !== 0) throw new Error('rock should beat scissors')
  const r2 = resolve('paper', 'rock')
  if (r2.draw || r2.winner !== 0) throw new Error('paper should beat rock')
  const r3 = resolve('scissors', 'paper')
  if (r3.draw || r3.winner !== 0) throw new Error('scissors should beat paper')
  const r4 = resolve('rock', 'timeout')
  if (r4.draw || r4.winner !== 0) throw new Error('throw beats timeout')
  const r5 = resolve('timeout', 'timeout')
  if (!r5.draw) throw new Error('double timeout is a draw')
  const m = newMatch('a', 'b', 2, 10)
  submit(m, 'a', 'rock')
  submit(m, 'b', 'scissors')
  ready(m, 'a')
  ready(m, 'b')
  if (m.scores[0] !== 1) throw new Error('first round should score')
  submit(m, 'a', 'paper')
  timeout(m)
  if (m.scores[0] < 2) throw new Error('timeout should award the second win')
}
