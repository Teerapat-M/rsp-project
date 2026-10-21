import { AppShell } from '../components/AppShell'
import { Avatar } from '../components/Avatar'
import { Button } from '../components/Button'
import { Hand, RandomBadge } from '../components/Hand'
import { IconTrophy } from '../components/Icons'
import { useEffect } from 'react'
import { useAuth } from '../context/AuthContext'
import { useGame } from '../context/GameContext'
import { useNavigate } from 'react-router-dom'
import type { PlayableMove } from '../types'

export function MatchResultPage() {
  const { user } = useAuth()
  const { view, playAgain, playAgainReady, leaveRoom } = useGame()
  const nav = useNavigate()

  useEffect(() => {
    if (view && !view.finished) nav('/play')
  }, [view, nav])
  if (!user || !view) {
    return (
      <AppShell>
        <p className="hint">No match result. Head back to the lobby.</p>
        <Button to="/lobby">Back to Lobby</Button>
      </AppShell>
    )
  }
  const youScore = view.scores[user.id] ?? 0
  const oppScore = view.scores[view.opponentId] ?? 0
  const won = view.winnerId === user.id
  return (
    <AppShell>
      <div className="pageCenter">
        <div className="resultPage">
          <div className="winnerCol">
            <Avatar kind={user.avatar} color={user.color} size={88} ring={won} />
            <h1 className="heroTitle" style={{ marginTop: 12, display: 'inline-flex', alignItems: 'center', gap: 8 }}>
              {won ? 'You Win!' : 'You Lose'} {won ? <IconTrophy color="#F2B01E" /> : null}
            </h1>
            <div style={{ color: '#5f6675' }}>{user.displayName}</div>
            <div className="scoreLine">
              {youScore} – {oppScore}
            </div>
          </div>
          <div className="panel">
            <h3 style={{ marginBottom: 12 }}>Match summary</h3>
            <div className="summaryList">
              {view.history.map((h) => {
                const yours = h.moves[user.id]
                const opp = h.moves[view.opponentId]
                const w = h.winnerId === user.id
                return (
                  <div className="summaryRow" key={h.number}>
                    <span>Round {h.number}</span>
                    {yours && yours !== 'timeout' ? <Hand move={yours as PlayableMove} size={22} /> : '—'}
                    <RandomBadge show={h.autoPicked?.[user.id]} />
                    <span style={{ color: '#8a909c' }}>vs</span>
                    {opp && opp !== 'timeout' ? <Hand move={opp as PlayableMove} size={22} /> : '—'}
                    <RandomBadge show={h.autoPicked?.[view.opponentId]} />
                    <span className={`wl ${w ? 'wlW' : 'wlL'}`}>{w ? 'W' : h.draw ? 'D' : 'L'}</span>
                  </div>
                )
              })}
            </div>
          </div>
        </div>
        <div className="actionsRow" style={{ marginTop: 24 }}>
          <Button
            onClick={() => {
              playAgain()
            }}
          >
            Play Again
          </Button>
          <Button
            variant="secondary"
            onClick={async () => {
              await leaveRoom()
              nav('/lobby')
            }}
          >
            Back to Lobby
          </Button>
        </div>
        {playAgainReady > 0 ? (
          <p className="hint">Waiting for opponent... ({playAgainReady}/2 ready)</p>
        ) : (
          <p className="hint">Waiting for opponent... (1/2 ready)</p>
        )}
      </div>
    </AppShell>
  )
}
