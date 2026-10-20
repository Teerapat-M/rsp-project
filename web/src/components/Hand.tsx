import type { CSSProperties } from 'react'
import { svgAsset } from '../assets/svg'
import type { Move, PlayableMove } from '../types'

const FILL: Record<PlayableMove, string> = {
  rock: '#9DB89A',
  scissors: '#A897C8',
  paper: '#93BFE3',
}

export type HandTone = 'color' | 'white' | 'gray' | 'slate'

const ART: Record<PlayableMove, Record<HandTone, string>> = {
  rock: {
    color: 'card-art-rock-sage',
    white: 'card-art-rock-white',
    gray: 'card-art-rock-gray',
    slate: 'card-art-rock-gray', // no slate export; gray is the closest unselected fill
  },
  scissors: {
    color: 'card-art-scissors-lavender',
    white: 'card-art-scissors-white',
    gray: 'card-art-scissors-gray',
    slate: 'card-art-scissors-slate',
  },
  paper: {
    color: 'card-art-paper-blue',
    white: 'card-art-paper-white',
    gray: 'card-art-paper-gray',
    slate: 'card-art-paper-slate',
  },
}

type Props = {
  move: PlayableMove | 'hidden'
  tone?: HandTone
  size?: number
  title?: string
}

export function Hand({ move, tone = 'color', size = 48, title }: Props) {
  const style: CSSProperties = { width: size, height: size }
  if (move === 'hidden') {
    return (
      <svg viewBox="0 0 64 64" style={{ ...style, display: 'block' }}>
        <rect x="10" y="8" width="44" height="48" rx="10" fill="#5B5FE6" />
        <text x="32" y="40" textAnchor="middle" fontSize="28" fontWeight="800" fill="#fff" fontFamily="Inter Tight, sans-serif">
          ?
        </text>
      </svg>
    )
  }
  return (
    <span
      className="svgIcon"
      role="img"
      aria-label={title ?? move}
      style={style}
      dangerouslySetInnerHTML={{ __html: svgAsset(ART[move][tone]) }}
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
