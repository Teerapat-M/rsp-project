import { describe, expect, it } from 'vitest'
import {
  hostedClientURLError,
  isAbsoluteHttpsURL,
  isAbsoluteWssURL,
  shouldRequireHostedClientURLs,
} from './hostedClientURLs'

describe('shouldRequireHostedClientURLs', () => {
  it('is off for local and CI builds', () => {
    expect(shouldRequireHostedClientURLs({})).toBe(false)
    expect(shouldRequireHostedClientURLs({ VERCEL: '' })).toBe(false)
    expect(shouldRequireHostedClientURLs({ VERCEL: '   ' })).toBe(false)
  })

  it('is on when Vercel injects VERCEL', () => {
    expect(shouldRequireHostedClientURLs({ VERCEL: '1' })).toBe(true)
  })
})

describe('hostedClientURLError', () => {
  it('accepts Render-style absolute URLs', () => {
    expect(
      hostedClientURLError('https://rsp-api.onrender.com', 'wss://rsp-api.onrender.com/ws'),
    ).toBeNull()
  })

  it('rejects missing, relative, and non-TLS URLs', () => {
    expect(hostedClientURLError(undefined, 'wss://api.example/ws')).toMatch(/VITE_API_BASE/)
    expect(hostedClientURLError('/api', 'wss://api.example/ws')).toMatch(/VITE_API_BASE/)
    expect(hostedClientURLError('http://api.example', 'wss://api.example/ws')).toMatch(/VITE_API_BASE/)
    expect(hostedClientURLError('https://api.example', '')).toMatch(/VITE_WS_URL/)
    expect(hostedClientURLError('https://api.example', '/ws')).toMatch(/VITE_WS_URL/)
    expect(hostedClientURLError('https://api.example', 'ws://api.example/ws')).toMatch(/VITE_WS_URL/)
    expect(hostedClientURLError('https://api.example', 'https://api.example/ws')).toMatch(/VITE_WS_URL/)
  })
})

describe('absolute URL helpers', () => {
  it('needs a host', () => {
    expect(isAbsoluteHttpsURL('https://')).toBe(false)
    expect(isAbsoluteWssURL('wss://')).toBe(false)
    expect(isAbsoluteHttpsURL('https://api.example')).toBe(true)
  })
})
