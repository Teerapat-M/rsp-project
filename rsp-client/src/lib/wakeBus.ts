import type { WakeState } from './wake'

type Fn = (s: WakeState) => void

const listeners = new Set<Fn>()
let state: WakeState = { waking: false, attempt: 0 }

export function getWakeState(): WakeState {
  return state
}

export function subscribeWake(fn: Fn): () => void {
  listeners.add(fn)
  fn(state)
  return () => {
    listeners.delete(fn)
  }
}

export function noticeWake(attempt: number): void {
  state = { waking: true, attempt }
  listeners.forEach((l) => l(state))
}

export function clearWake(): void {
  if (!state.waking && state.attempt === 0) return
  state = { waking: false, attempt: 0 }
  listeners.forEach((l) => l(state))
}
