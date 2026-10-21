/** Vercel SPA rewrites `/(.*)` to index.html. A relative `/api` then 200s HTML. */

export function isAbsoluteHttpsURL(value: string): boolean {
  return isAbsoluteURL(value, 'https:')
}

export function isAbsoluteWssURL(value: string): boolean {
  return isAbsoluteURL(value, 'wss:')
}

function isAbsoluteURL(value: string, protocol: string): boolean {
  const raw = value.trim()
  if (!raw) return false
  try {
    const u = new URL(raw)
    return u.protocol === protocol && Boolean(u.hostname)
  } catch {
    return false
  }
}

export function hostedClientURLError(apiBase?: string, wsURL?: string): string | null {
  if (!isAbsoluteHttpsURL(apiBase ?? '')) {
    return 'VITE_API_BASE must be an absolute https:// URL on Vercel (relative /api is rewritten to index.html).'
  }
  if (!isAbsoluteWssURL(wsURL ?? '')) {
    return 'VITE_WS_URL must be an absolute wss:// URL on Vercel.'
  }
  return null
}

/** Vercel sets VERCEL=1. Local `vite` / `npm run build` leave it unset. */
export function shouldRequireHostedClientURLs(env: { VERCEL?: string }): boolean {
  return Boolean(env.VERCEL?.trim())
}
