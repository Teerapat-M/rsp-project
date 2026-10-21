import { Hand, moveLabel, RandomBadge } from './Hand'
import type { PlayableMove, RoundResultView } from '../types'

export type RoundPicks = Pick<
  RoundResultView,
  'youWon' | 'draw' | 'yourMove' | 'opponentMove' | 'yourAutoPicked' | 'opponentAutoPicked'
>

export function roundSideBadge(youWon: boolean, draw: boolean, side: 'you' | 'opponent'): 'Winner' | 'Lost' | null {
  if (draw) return null
  const won = side === 'you' ? youWon : !youWon
  return won ? 'Winner' : 'Lost'
}

function SideBadge({ kind }: { kind: 'Winner' | 'Lost' | null }) {
  if (!kind) return null
  if (kind === 'Winner') return <span className="badge badgeOk">Winner</span>
  return <span className="badge badgeLost">Lost</span>
}

export function RoundResultPicks({ roundResult }: { roundResult: RoundPicks }) {
  const youBadge = roundSideBadge(roundResult.youWon, roundResult.draw, 'you')
  const oppBadge = roundSideBadge(roundResult.youWon, roundResult.draw, 'opponent')
  return (
    <div className="vsRow">
      <div className={`vsPick ${youBadge === 'Winner' ? 'win' : ''}`} data-side="you">
        <span>You</span>
        <div className="bigCard">
          <Hand move={roundResult.yourMove as PlayableMove} size={72} />
          {moveLabel(roundResult.yourMove)}
        </div>
        <RandomBadge show={roundResult.yourAutoPicked} />
        <SideBadge kind={youBadge} />
      </div>
      <div className="vs">VS</div>
      <div className={`vsPick ${oppBadge === 'Winner' ? 'win' : ''}`} data-side="opponent">
        <span>Opponent</span>
        <div className="bigCard">
          <Hand move={roundResult.opponentMove as PlayableMove} size={72} />
          {moveLabel(roundResult.opponentMove)}
        </div>
        <RandomBadge show={roundResult.opponentAutoPicked} />
        <SideBadge kind={oppBadge} />
      </div>
    </div>
  )
}
