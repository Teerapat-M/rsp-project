const TICK_MS = 1000

export function deadlineMs(nextRoundAt: string | undefined, fallbackSeconds: number | undefined, nowMs: number): number {
  if (nextRoundAt) {
    const at = Date.parse(nextRoundAt)
    if (!Number.isNaN(at)) return at
  }
  return nowMs + Math.max(1, fallbackSeconds || 3) * TICK_MS
}

export function secondsUntil(nextRoundAt: string | undefined, nowMs: number, fallbackSeconds = 3): number {
  return Math.max(0, Math.ceil((deadlineMs(nextRoundAt, fallbackSeconds, nowMs) - nowMs) / TICK_MS))
}

export type WatchNextRoundOpts = {
  nextRoundAt?: string
  nextRoundIn?: number
  now?: () => number
  onTick: (secondsLeft: number) => void
  onDone: () => void
}

/** Tick 3, 2, 1 from the server deadline, then fire onDone. stop() clears both timers. */
export function watchNextRound(opts: WatchNextRoundOpts): () => void {
  const now = opts.now ?? Date.now
  const deadline = deadlineMs(opts.nextRoundAt, opts.nextRoundIn, now())
  let stopped = false
  let timeoutId: ReturnType<typeof setTimeout> | undefined
  let intervalId: ReturnType<typeof setInterval> | undefined

  const stop = () => {
    if (stopped) return
    stopped = true
    if (timeoutId !== undefined) clearTimeout(timeoutId)
    if (intervalId !== undefined) clearInterval(intervalId)
  }

  const finish = () => {
    if (stopped) return
    stop()
    opts.onDone()
  }

  const tick = () => {
    if (stopped) return
    const left = Math.max(0, Math.ceil((deadline - now()) / TICK_MS))
    if (left <= 0) {
      finish()
      return
    }
    opts.onTick(left)
  }

  tick()
  if (stopped) return stop
  timeoutId = setTimeout(finish, Math.max(0, deadline - now()))
  intervalId = setInterval(tick, TICK_MS)
  return stop
}
