export const ACTIVE_ROOM_KEY = 'rsp.activeRoom'

type Reader = Pick<Storage, 'getItem'>
type Writer = Pick<Storage, 'setItem' | 'removeItem'>

export function playPath(roomId: string): string {
  return `/play?room=${encodeURIComponent(roomId)}`
}

export function readActiveRoom(storage: Reader | undefined = defaultSession()): string | null {
  if (!storage) return null
  try {
    const v = storage.getItem(ACTIVE_ROOM_KEY)?.trim()
    return v || null
  } catch {
    return null
  }
}

export function writeActiveRoom(
  id: string | null,
  storage: Writer | undefined = defaultSession(),
): void {
  if (!storage) return
  try {
    if (id) storage.setItem(ACTIVE_ROOM_KEY, id)
    else storage.removeItem(ACTIVE_ROOM_KEY)
  } catch {
    // private mode / missing storage
  }
}

/** Prefer the URL, then the persisted tab room, then in-memory state. */
export function resolvePlayRoom(opts: {
  query: string | null | undefined
  stored: string | null | undefined
  live: string | null | undefined
}): string | null {
  for (const raw of [opts.query, opts.stored, opts.live]) {
    const v = raw?.trim()
    if (v) return v
  }
  return null
}

function defaultSession(): Storage | undefined {
  try {
    return typeof sessionStorage === 'undefined' ? undefined : sessionStorage
  } catch {
    return undefined
  }
}
