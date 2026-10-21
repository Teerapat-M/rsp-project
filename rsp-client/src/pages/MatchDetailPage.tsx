import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { AppShell } from '../components/AppShell'
import { Button } from '../components/Button'
import { Hand, moveLabel, RandomBadge } from '../components/Hand'
import { IconArrowLeft } from '../components/Icons'
import { api } from '../api'
import { useAuth } from '../context/AuthContext'
import { useGame } from '../context/GameContext'
import type { MatchDetail, PlayableMove } from '../types'

export function MatchDetailPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const { createRoom } = useGame()
  const nav = useNavigate()
  const [m, setM] = useState<MatchDetail | null>(null)

  useEffect(() => {
    if (id) api.getMatch(id).then(setM)
  }, [id])

  if (!m || !user) return <AppShell />

  return (
    <AppShell>
      <div className="pageCenter">
        <Link to="/history" style={{ alignSelf: 'flex-start', marginBottom: 8, display: 'inline-flex', alignItems: 'center', gap: 6 }}>
          <IconArrowLeft size={16} /> Match History
        </Link>
        <h1 className="heroTitle">
          {user.displayName} vs {m.opponentName}
        </h1>
        <div className="pill" style={{ borderColor: '#16a34a', color: '#16a34a', background: '#e3f5e9' }}>
          {m.won ? 'Win' : 'Loss'} • {m.youScore} – {m.opponentScore} • First to {m.winsToFinish} • {m.turnSeconds}s/round • {m.playedAt}
        </div>
        <div className="panel table">
          <div className="tableHead">
            <span>Round</span>
            <span>You</span>
            <span>Opponent</span>
            <span>Result</span>
            <span>Pick time</span>
          </div>
          {m.rounds.map((r) => (
            <div className="tableRow" key={r.number}>
              <strong>Round {r.number}</strong>
              <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                {r.you !== 'timeout' ? <Hand move={r.you as PlayableMove} size={20} /> : null}
                {moveLabel(r.you)}
                <RandomBadge show={r.youAutoPicked} />
              </span>
              <span style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                {r.opponent !== 'timeout' ? <Hand move={r.opponent as PlayableMove} size={20} /> : null}
                {moveLabel(r.opponent)}
                <RandomBadge show={r.opponentAutoPicked} />
              </span>
              <span className={`badge ${r.result === 'won' ? 'badgeOk' : ''}`} style={r.result === 'lost' ? { background: '#fde8e8', color: '#dc2626' } : undefined}>
                {r.result === 'won' ? 'Won' : r.result === 'lost' ? 'Lost' : 'Draw'}
              </span>
              <span style={{ color: '#5f6675' }}>{r.pickTimeSec.toFixed(1)}s</span>
            </div>
          ))}
        </div>
        <div className="actionsRow" style={{ marginTop: 18 }}>
          <Button
            onClick={async () => {
              const room = await createRoom(m.winsToFinish, m.turnSeconds)
              nav(`/room/${room.id}`)
            }}
          >
            Rematch
          </Button>
          <Button variant="secondary" to="/history">
            Back to History
          </Button>
        </div>
      </div>
    </AppShell>
  )
}
