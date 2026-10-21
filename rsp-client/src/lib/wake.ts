/** Render free-tier cold start: first request can take ~30–60s. */

export type WakeState = { waking: boolean; attempt: number }

export const WAKE_MAX_ATTEMPTS = 12

/** Backoff before retry N (0-based). Immediate, then 1s, 2s, 4s, 8s cap. */
export function wakeDelayMs(attempt: number): number {
  if (attempt <= 0) return 0
  return Math.min(1000 * 2 ** (attempt - 1), 8000)
}

export function isWakeNetworkError(err: unknown): boolean {
  if (!(err instanceof Error)) return false
  const m = err.message.toLowerCase()
  return (
    err.name === 'TypeError' ||
    m.includes('failed to fetch') ||
    m.includes('networkerror') ||
    m.includes('load failed') ||
    m.includes('network request failed')
  )
}

/** 502/504 are the proxy. 503 without our JSON `code` is the host waking. */
export function isWakeHTTP(status: number, body?: { code?: string }): boolean {
  if (status === 502 || status === 504) return true
  if (status === 503 && !body?.code) return true
  return false
}

export function wakeCopy(attempt: number, max = WAKE_MAX_ATTEMPTS): string {
  return `Waking the server… attempt ${Math.max(1, attempt)} of ${max}. Free-tier hosts sleep after idle — this can take about a minute.`
}
