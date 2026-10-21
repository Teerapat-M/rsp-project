import type { AvatarKind } from '../types'
import { Hand } from './Hand'

type Props = {
  kind: AvatarKind
  color: string
  size?: number
  ring?: boolean
}

export function Avatar({ kind, color, size = 40, ring }: Props) {
  const pad = Math.round(size * 0.22)
  return (
    <div
      className={`avatar ${ring ? 'avatarRing' : ''}`}
      style={{
        width: size,
        height: size,
        background: color,
        boxShadow: ring ? `0 0 0 3px #fff, 0 0 0 5px #16a34a` : undefined,
      }}
    >
      <Hand move={kind} tone="white" size={size - pad} />
    </div>
  )
}
