import { afterEach, describe, expect, it, vi } from 'vitest'
import { watchNextRound } from './nextRound'

describe('watchNextRound', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('ticks 3, 2, 1 from nextRoundAt then transitions', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-10-21T00:00:00.000Z'))

    const ticks: number[] = []
    let transitioned = false
    const stop = watchNextRound({
      nextRoundAt: '2026-10-21T00:00:03.000Z',
      onTick: (n) => ticks.push(n),
      onDone: () => {
        transitioned = true
      },
    })

    expect(ticks).toEqual([3])
    expect(transitioned).toBe(false)

    vi.advanceTimersByTime(1000)
    expect(ticks).toEqual([3, 2])
    expect(transitioned).toBe(false)

    vi.advanceTimersByTime(1000)
    expect(ticks).toEqual([3, 2, 1])
    expect(transitioned).toBe(false)

    vi.advanceTimersByTime(1000)
    expect(ticks).toEqual([3, 2, 1])
    expect(transitioned).toBe(true)

    stop()
  })

  it('clears timers on stop so unmount or rematch cannot fire', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-10-21T00:00:00.000Z'))

    const ticks: number[] = []
    let transitioned = false
    const stop = watchNextRound({
      nextRoundAt: '2026-10-21T00:00:03.000Z',
      onTick: (n) => ticks.push(n),
      onDone: () => {
        transitioned = true
      },
    })

    expect(ticks).toEqual([3])
    stop()
    vi.advanceTimersByTime(5000)
    expect(ticks).toEqual([3])
    expect(transitioned).toBe(false)
  })
})
