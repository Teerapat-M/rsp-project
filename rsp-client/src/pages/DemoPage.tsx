import { useNavigate } from 'react-router-dom'
import { AppShell } from '../components/AppShell'
import { Button } from '../components/Button'
import { api } from '../api'

const LINKS: Array<{ to: string; label: string; demo?: string }> = [
  { to: '/', label: '01 Splash' },
  { to: '/login', label: '02 Login' },
  { to: '/guest', label: '03 Guest name' },
  { to: '/lobby', label: '04 Lobby' },
  { to: '/create', label: '05 Create room' },
  { to: '/register', label: 'Register' },
  { to: '/forgot', label: 'Forgot password' },
  { to: '/stats', label: '10 Stats' },
  { to: '/history', label: '11 History' },
]

const GAME: Array<{ id: string; label: string }> = [
  { id: 'game', label: '07 Game board' },
  { id: 'game-placed', label: '07b Card placed' },
  { id: 'game-ready', label: '07c Ready / waiting' },
  { id: 'game-timeup', label: '07d Time up' },
  { id: 'game-dc', label: '07e Opponent disconnected' },
  { id: 'connection', label: '12c Connection lost' },
  { id: 'round', label: '08 Round result' },
  { id: 'match', label: '09 Match result' },
]

export function DemoPage() {
  const nav = useNavigate()
  return (
    <AppShell>
      <div className="pageCenter">
        <h1 className="heroTitle">Screen gallery</h1>
        <p className="heroSub">Mock-only shortcuts so every board is one click away</p>
        <div className="demoGrid">
          {LINKS.map((l) => (
            <Button key={l.label} variant="secondary" to={l.to}>
              {l.label}
            </Button>
          ))}
          <Button
            variant="secondary"
            onClick={async () => {
              const r = await api.createRoom({ winsToFinish: 3, turnSeconds: 15 })
              nav(`/room/${r.id}`)
            }}
          >
            06 Waiting room
          </Button>
          {GAME.map((g) => (
            <button
              key={g.id}
              type="button"
              onClick={async () => {
                const path = api.applyDemo ? await api.applyDemo(g.id) : '/play'
                nav(path)
              }}
            >
              {g.label}
            </button>
          ))}
          <Button
            variant="secondary"
            onClick={() => {
              nav('/lobby')
              queueMicrotask(() => {
                /* passcodes documented in README */
              })
            }}
          >
            Lobby + passcode 482910 / 111111 for modals
          </Button>
        </div>
        <p className="hint">Join passcode 482910 = wrong · 111111 = room full</p>
        <div style={{ marginTop: 16 }}>
          <Button to="/lobby">← Back to lobby</Button>
        </div>
      </div>
    </AppShell>
  )
}
