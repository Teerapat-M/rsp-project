export const ACTIVE_ROOM_KEY = 'rsp.activeRoom'

type Reader = Pick<Storage, 'getItem'>
type Writer = Pick<Storage, 'setItem' | 'removeItem'>

export type ActiveRoomStore = {
  userId?: string | null
  storage?: Reader & Partial<Writer>
}

export function playPath(roomId: string): string {
  return `/play?room=${encodeURIComponent(roomId)}`
}

export function activeRoomKey(userId: string): string {
  return `${ACTIVE_ROOM_KEY}.${userId}`
}

export function readActiveRoom(opts: ActiveRoomStore | undefined = undefined): string | null {
  const uid = opts?.userId ?? currentUserId()
  const storage = opts?.storage ?? defaultLocal()
  if (!uid || !storage) return null
  try {
    const v = storage.getItem(activeRoomKey(uid))?.trim()
    return v || null
  } catch {
    return null
  }
}

export function writeActiveRoom(
  id: string | null,
  opts: ActiveRoomStore | undefined = undefined,
): void {
  const uid = opts?.userId ?? currentUserId()
  const storage = (opts?.storage as Writer | undefined) ?? defaultLocal()
  if (!uid || !storage) return
  try {
    const key = activeRoomKey(uid)
    if (id) storage.setItem(key, id)
    else storage.removeItem(key)
  } catch {
    // private mode / missing storage
  }
}

/** Prefer the URL, then the persisted room, then in-memory state. */
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

function currentUserId(): string | null {
  try {
    if (typeof localStorage === 'undefined') return null
    const raw = localStorage.getItem('rsp.user')
    if (!raw) return null
    const u = JSON.parse(raw) as { id?: string }
    return u.id?.trim() || null
  } catch {
    return null
  }
}

function defaultLocal(): Storage | undefined {
  try {
    return typeof localStorage === 'undefined' ? undefined : localStorage
  } catch {
    return undefined
  }
}
