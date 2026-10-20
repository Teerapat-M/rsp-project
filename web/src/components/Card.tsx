import type { CSSProperties, ReactNode } from 'react'
import type { PlayableMove } from '../types'
import { Hand, moveLabel } from './Hand'
import styles from './Card.module.css'

type State = 'normal' | 'selected' | 'disabled'

type Props = {
  move: PlayableMove
  state?: State
  dragging?: boolean
  ghost?: boolean
  mini?: boolean
  size?: number
  style?: CSSProperties
  listeners?: object
  attributes?: object
}

export function Card({
  move,
  state = 'normal',
  dragging,
  ghost,
  mini,
  size,
  style,
  listeners,
  attributes,
}: Props) {
  const cls = [
    styles.card,
    state === 'selected' ? styles.selected : '',
    state === 'disabled' ? styles.disabled : '',
    dragging ? styles.dragging : '',
    ghost ? styles.ghost : '',
    mini ? styles.mini : '',
  ]
    .filter(Boolean)
    .join(' ')
  const iconColor = state === 'disabled' ? '#A0A6B1' : undefined
  return (
    <div className={cls} style={style} {...attributes} {...listeners}>
      <Hand move={move} color={iconColor} size={size ?? (mini ? 36 : 52)} />
      <span className={styles.label}>{moveLabel(move)}</span>
    </div>
  )
}

export function FaceDown({ dim, children }: { dim?: boolean; children?: ReactNode }) {
  return <div className={`${styles.faceDown} ${dim ? styles.faceDownDim : ''}`}>{children ?? '?'}</div>
}
