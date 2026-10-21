import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { AppShell } from '../components/AppShell'
import { Avatar } from '../components/Avatar'
import { Button } from '../components/Button'
import { IconChevronRight } from '../components/Icons'
import { api } from '../api'
import type { HistoryItem } from '../types'

export function HistoryPage() {
  const [items, setItems] = useState<HistoryItem[]>([])
  const [total, setTotal] = useState(0)
  const [cursor, setCursor] = useState<string | undefined>(undefined)

  async function load(next?: string) {
    const page = await api.getHistory(next)
    setItems((prev) => (next ? [...prev, ...page.items] : page.items))
    setTotal(page.total)
    setCursor(page.nextCursor)
  }

  useEffect(() => {
    load()
  }, [])

  return (
    <AppShell>
      <div className="pageCenter">
        <h1 className="heroTitle">Match History</h1>
        <p className="heroSub">Your recent matches</p>
        <div className="panel historyList">
          {items.map((m, i) => (
            <Link key={m.id} className={`historyRow ${i === 0 ? 'active' : ''}`} to={`/history/${m.id}`}>
              <Avatar kind={m.opponentAvatar} color={m.opponentColor} size={40} />
              <div className="meta">
                <strong>vs {m.opponentName}</strong>
                <span>{m.playedAt}</span>
              </div>
              <div className="historyScore">
                {m.youScore} – {m.opponentScore}
              </div>
              <span className={`badge ${m.won ? 'badgeOk' : ''}`} style={m.won ? undefined : { background: '#fde8e8', color: '#dc2626' }}>
                {m.won ? 'Win' : 'Loss'}
              </span>
              <IconChevronRight color="#8a909c" />
            </Link>
          ))}
        </div>
        <div className="actionsRow" style={{ marginTop: 18 }}>
          <Button disabled={!cursor} onClick={() => cursor && load(cursor)}>
            Load more
          </Button>
          <Button to="/lobby">← Back to lobby</Button>
        </div>
        <p className="hint">
          Showing {items.length} of {total} matches
        </p>
      </div>
    </AppShell>
  )
}
