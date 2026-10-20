import { useEffect, useState } from 'react'
import { AppShell } from '../components/AppShell'
import { Avatar } from '../components/Avatar'
import { Button } from '../components/Button'
import { Hand, HAND_FILL } from '../components/Hand'
import { useAuth } from '../context/AuthContext'
import { api } from '../api'
import type { UserStats } from '../types'

export function StatsPage() {
  const { user } = useAuth()
  const [stats, setStats] = useState<UserStats | null>(null)

  useEffect(() => {
    api.getStats().then(setStats)
  }, [])

  if (!user || !stats) return <AppShell />
  const totalPicks = stats.picks.rock + stats.picks.paper + stats.picks.scissors || 1
  const kinds: Array<'rock' | 'paper' | 'scissors'> = ['scissors', 'rock', 'paper']
  const most = kinds.reduce((a, b) => (stats.picks[b] > stats.picks[a] ? b : a))
  const mostPct = Math.round((stats.picks[most] / totalPicks) * 100)
  const mostWin = stats.picks[most] ? Math.round((stats.pickWins[most] / stats.picks[most]) * 100) : 0
  const kind = user.isGuest ? 'Guest player' : 'Player'

  return (
    <AppShell>
      <div className="pageCenter">
        <Avatar kind={user.avatar} color={user.color} size={72} />
        <h1 className="heroTitle" style={{ marginTop: 10 }}>
          {user.displayName}
        </h1>
        <p className="heroSub">
          {kind} · {stats.matchesPlayed} matches played
        </p>
        <div className="statsGrid">
          <div className="panel statCard">
            <span>🏆 Wins</span>
            <strong className="statWin">{stats.wins}</strong>
          </div>
          <div className="panel statCard">
            <span>✕ Losses</span>
            <strong className="statLose">{stats.losses}</strong>
          </div>
          <div className="panel statCard">
            <span>% Win rate</span>
            <strong className="statRate">{stats.winRate}%</strong>
          </div>
          <div className="panel statCard">
            <span>🔥 Best streak</span>
            <strong className="statStreak">{stats.bestStreak}</strong>
          </div>
        </div>
        <div className="statsLower">
          <div className="panel mostPlayed">
            <div className="mostIcon">
              <Hand move={most} size={56} />
            </div>
            <div>
              <div style={{ fontWeight: 600, marginBottom: 4 }}>Most played card</div>
              <h2>{most[0].toUpperCase() + most.slice(1)}</h2>
              <p className="hint" style={{ textAlign: 'left' }}>
                Picked in {mostPct}% of rounds
              </p>
              <div style={{ color: '#16a34a', fontWeight: 600 }}>Win rate with it: {mostWin}%</div>
            </div>
          </div>
          <div className="panel">
            <div style={{ fontWeight: 600, marginBottom: 8 }}>Card picks</div>
            {(['rock', 'scissors', 'paper'] as const).map((m) => {
              const pct = Math.round((stats.picks[m] / totalPicks) * 100)
              return (
                <div className="barRow" key={m}>
                  <Hand move={m} size={18} />
                  <span>{m[0].toUpperCase() + m.slice(1)}</span>
                  <div className="bar">
                    <i style={{ width: `${pct}%`, background: HAND_FILL[m] }} />
                  </div>
                  <span>{pct}%</span>
                </div>
              )
            })}
          </div>
        </div>
        <div style={{ marginTop: 22 }}>
          <Button to="/lobby">← Back to lobby</Button>
        </div>
      </div>
    </AppShell>
  )
}
