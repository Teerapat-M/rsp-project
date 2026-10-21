export const RECONNECT_GRACE_SEC = 20
export const RECONNECT_MAX_ATTEMPTS = 5

/** Backoff before retry N (0-based). Immediate, then 1s, 2s, 4s, 4s cap. */
export function reconnectDelayMs(attempt: number): number {
  if (attempt <= 0) return 0
  const ms = 1000 * 2 ** (attempt - 1)
  return Math.min(ms, 4000)
}

export function formatGrace(seconds: number): string {
  const n = Math.max(0, Math.floor(seconds))
  return `0:${String(n).padStart(2, '0')}`
}
