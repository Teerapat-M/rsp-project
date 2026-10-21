import type { AvatarKind, PlayerView, Room, RoomPlayer, User } from '../types'

export function asAvatar(value: string | undefined | null): AvatarKind {
  if (value === 'paper' || value === 'scissors' || value === 'rock') return value
  return 'rock'
}

function fromUser(user: User): Pick<User, 'id' | 'displayName' | 'avatar' | 'color'> {
  return {
    id: user.id,
    displayName: user.displayName,
    avatar: asAvatar(user.avatar),
    color: user.color,
  }
}

function fromSeat(p: RoomPlayer): Pick<User, 'id' | 'displayName' | 'avatar' | 'color'> {
  return {
    id: p.id,
    displayName: p.displayName,
    avatar: asAvatar(p.avatar),
    color: p.color,
  }
}

/** Self identity for in-room screens: hub snapshot, then seated room player, then profile. */
export function youInRoom(
  user: User,
  view: PlayerView | null | undefined,
  room: Room | null | undefined,
): Pick<User, 'id' | 'displayName' | 'avatar' | 'color'> {
  if (view?.you?.id) {
    return {
      id: view.you.id,
      displayName: view.you.displayName || user.displayName,
      avatar: asAvatar(view.you.avatar || user.avatar),
      color: view.you.color || user.color,
    }
  }
  const seated = room?.players.find((p) => p.id === user.id)
  if (seated) {
    return {
      ...fromSeat(seated),
      displayName: seated.displayName || user.displayName,
      avatar: asAvatar(seated.avatar || user.avatar),
      color: seated.color || user.color,
    }
  }
  return fromUser(user)
}

export function opponentInRoom(
  selfId: string,
  view: PlayerView | null | undefined,
  room: Room | null | undefined,
): Pick<User, 'id' | 'displayName' | 'avatar' | 'color'> | null {
  if (view?.opponentId) {
    return {
      id: view.opponentId,
      displayName: view.opponentName,
      avatar: asAvatar(view.opponentAvatar),
      color: view.opponentColor,
    }
  }
  const seated = room?.players.find((p) => p.id !== selfId)
  if (!seated) return null
  return fromSeat(seated)
}

export function seatedAvatar(player: RoomPlayer, self?: User | null): AvatarKind {
  if (player.avatar) return asAvatar(player.avatar)
  if (self && player.id === self.id) return asAvatar(self.avatar)
  return asAvatar(player.avatar)
}

export function seatedColor(player: RoomPlayer, self?: User | null): string {
  if (player.color) return player.color
  if (self && player.id === self.id) return self.color
  return player.color
}
