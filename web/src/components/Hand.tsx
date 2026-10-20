import type { CSSProperties } from 'react'
import type { Move, PlayableMove } from '../types'

const FILL: Record<PlayableMove, string> = {
  rock: '#9DB89A',
  scissors: '#A897C8',
  paper: '#93BFE3',
}

const SRC: Record<PlayableMove, string> = {
  rock: '/hands/rock.png',
  scissors: '/hands/scissors.png',
  paper: '/hands/paper.png',
}

type Props = {
  move: PlayableMove | 'hidden'
  color?: string
  size?: number
  title?: string
}

function imgFilter(color?: string): string | undefined {
  if (!color) return undefined
  const c = color.toLowerCase()
  if (c === '#fff' || c === '#ffffff' || c === 'white') return 'brightness(0) invert(1)'
  if (c === '#a0a6b1' || c === '#4a5263' || c === '#c9cdd4') return 'grayscale(1) opacity(0.55)'
  return undefined
}

export function Hand({ move, color, size = 48, title }: Props) {
  const style: CSSProperties = { width: size, height: size, display: 'block', objectFit: 'contain' }
  if (move === 'hidden') {
    return (
      <svg viewBox="0 0 64 64" style={style}>
        <rect x="10" y="8" width="44" height="48" rx="10" fill={color ?? '#5B5FE6'} />
        <text x="32" y="40" textAnchor="middle" fontSize="28" fontWeight="800" fill="#fff" fontFamily="Inter Tight, sans-serif">
          ?
        </text>
      </svg>
    )
  }
  return (
    <img
      src={SRC[move]}
      alt={title ?? move}
      width={size}
      height={size}
      style={{ ...style, filter: imgFilter(color) }}
      draggable={false}
    />
  )
}

export function moveLabel(move: Move | '' | 'hidden'): string {
  if (move === 'rock') return 'Rock'
  if (move === 'paper') return 'Paper'
  if (move === 'scissors') return 'Scissors'
  if (move === 'timeout') return 'Timeout'
  if (move === 'hidden') return '?'
  return ''
}

export const HAND_ORDER: PlayableMove[] = ['rock', 'scissors', 'paper']
export const HAND_FILL = FILL
