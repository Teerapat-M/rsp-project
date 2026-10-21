import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { TURN_PRESETS } from '../theme'
import { AppShell } from '../components/AppShell'
import { Button } from '../components/Button'
import { IconMinus, IconPlus } from '../components/Icons'
import { useGame } from '../context/GameContext'

export function CreateRoomPage() {
  const [wins, setWins] = useState(3)
  const [seconds, setSeconds] = useState(15)
  const [error, setError] = useState('')
  const { createRoom } = useGame()
  const nav = useNavigate()

  return (
    <AppShell>
      <div className="pageCenter">
        <h1 className="heroTitle">Create Room</h1>
        <p className="heroSub">Set up your match, then share the passcode</p>
        <div className="panel settingsCard">
          <div className="settingsRow">
            <div className="settingsCopy">
              <h4>Rounds to win</h4>
              <p>First player to win this many rounds</p>
            </div>
            <div className="stepper">
              <button type="button" onClick={() => setWins((n) => Math.max(1, n - 1))} aria-label="Decrease">
                <IconMinus size={18} />
              </button>
              <strong>{wins}</strong>
              <button type="button" onClick={() => setWins((n) => Math.min(9, n + 1))} aria-label="Increase">
                <IconPlus size={18} />
              </button>
            </div>
          </div>
          <h4>Time per round (seconds)</h4>
          <div className="seg">
            {TURN_PRESETS.map((n) => (
              <button key={n} type="button" className={seconds === n ? 'on' : ''} onClick={() => setSeconds(n)}>
                {n}
              </button>
            ))}
          </div>
          {error ? <p className="hint" style={{ color: '#dc2626' }}>{error}</p> : null}
          <Button
            block
            onClick={async () => {
              setError('')
              try {
                const room = await createRoom(wins, seconds)
                nav(`/room/${room.id}`)
              } catch (err) {
                setError(err instanceof Error ? err.message : 'Could not create room')
              }
            }}
          >
            Create Room
          </Button>
        </div>
        <div style={{ marginTop: 18 }}>
          <Button to="/lobby">← Back to lobby</Button>
        </div>
      </div>
    </AppShell>
  )
}
