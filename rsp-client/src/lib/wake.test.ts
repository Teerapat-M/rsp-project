import { describe, expect, it } from 'vitest'
import { isWakeHTTP, isWakeNetworkError, wakeCopy, wakeDelayMs, WAKE_MAX_ATTEMPTS } from './wake'

describe('wakeDelayMs', () => {
  it('backs off then caps at 8s', () => {
    expect(wakeDelayMs(0)).toBe(0)
    expect(wakeDelayMs(1)).toBe(1000)
    expect(wakeDelayMs(2)).toBe(2000)
    expect(wakeDelayMs(3)).toBe(4000)
    expect(wakeDelayMs(4)).toBe(8000)
    expect(wakeDelayMs(5)).toBe(8000)
  })
})

describe('isWakeHTTP', () => {
  it('retries proxy errors, not our JSON 503', () => {
    expect(isWakeHTTP(502)).toBe(true)
    expect(isWakeHTTP(504)).toBe(true)
    expect(isWakeHTTP(503)).toBe(true)
    expect(isWakeHTTP(503, { code: 'UNAVAILABLE' })).toBe(false)
    expect(isWakeHTTP(401)).toBe(false)
    expect(isWakeHTTP(409, { code: 'ROOM_FULL' })).toBe(false)
  })
})

describe('isWakeNetworkError', () => {
  it('treats fetch failures as a sleeping host', () => {
    expect(isWakeNetworkError(new TypeError('Failed to fetch'))).toBe(true)
    expect(isWakeNetworkError(new Error('NetworkError when attempting to fetch'))).toBe(true)
    expect(isWakeNetworkError(new Error('WRONG_PASSCODE'))).toBe(false)
  })
})

describe('wakeCopy', () => {
  it('names the attempt', () => {
    expect(wakeCopy(2)).toContain('attempt 2 of ' + WAKE_MAX_ATTEMPTS)
    expect(wakeCopy(1)).toContain('Waking the server')
  })
})
