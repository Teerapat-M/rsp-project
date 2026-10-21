import { describe, expect, it } from 'vitest'
import { formatGrace, reconnectDelayMs } from './reconnect'

describe('reconnectDelayMs', () => {
  it('backs off then caps', () => {
    expect(reconnectDelayMs(0)).toBe(0)
    expect(reconnectDelayMs(1)).toBe(1000)
    expect(reconnectDelayMs(2)).toBe(2000)
    expect(reconnectDelayMs(3)).toBe(4000)
    expect(reconnectDelayMs(4)).toBe(4000)
  })
})

describe('formatGrace', () => {
  it('pads whole seconds', () => {
    expect(formatGrace(20)).toBe('0:20')
    expect(formatGrace(9)).toBe('0:09')
    expect(formatGrace(-1)).toBe('0:00')
  })
})
