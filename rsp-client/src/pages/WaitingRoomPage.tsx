import { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { AppShell } from '../components/AppShell'
import { Avatar } from '../components/Avatar'
import { Button } from '../components/Button'
import { PageStatus } from '../components/PageStatus'
import { IconCheck, IconClock, IconCopy, IconLogout, IconUsers } from '../components/Icons'
import { useAuth } from '../context/AuthContext'
import { useGame } from '../context/GameContext'
import { formatPasscode } from '../lib/format'
import { playPath } from '../lib/playRoom'
import { seatedAvatar, seatedColor } from '../lib/profile'

export function WaitingRoomPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const { room, loadRoom, startGame, leaveRoom, view, setLobbyReady } = useGame()
  const nav = useNavigate()
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState('')
  const loadRoomRef = useRef(loadRoom)
  useEffect(() => {
    loadRoomRef.current = loadRoom
  })

  useEffect(() => {
    if (!id) return
    loadRoomRef.current(id).catch((err: unknown) => {
      setError(err instanceof Error ? err.message : 'Room not found')
    })
  }, [id])

  useEffect(() => {
    if (view && !view.finished) nav(playPath(view.roomId), { replace: true })
  }, [view, nav])

  if (error) {
    return (
      <PageStatus>
        {error}{' '}
        <Button variant="ghost" onClick={() => nav('/lobby')}>
          Back to lobby
        </Button>
      </PageStatus>
    )
  }
  if (!room || !user) return <PageStatus>Connecting…</PageStatus>
  const isHost = room.hostId === user.id
  const canStart = isHost && room.players.length === 2
  const me = room.players.find((p) => p.id === user.id)
  const myReady = Boolean(me?.ready)

  return (
    <AppShell>
      <div className="pageCenter">
        <h1 className="heroTitle">Waiting Room</h1>
        <div className="pill">
          <IconClock color="#4B4FD0" /> First to {room.winsToFinish} wins • {room.turnSeconds}s per round
        </div>
        <div className="waitingGrid">
          <div className="panel passcodeBox">
            <div className="label">Room passcode</div>
            <div className="passcode">{formatPasscode(room.passcode)}</div>
            <Button
              variant="secondary"
              size="sm"
              onClick={async () => {
                await navigator.clipboard.writeText(room.passcode)
                setCopied(true)
                setTimeout(() => setCopied(false), 1200)
              }}
            >
              <IconCopy /> {copied ? 'Copied' : 'Copy'}
            </Button>
            <p className="hint">Share this code with a friend to join</p>
          </div>
          <div className="panel">
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontWeight: 600 }}>
              <IconUsers /> Players ({room.players.length}/2)
            </div>
            {room.players.map((p) => (
              <div className="playerRow" key={p.id}>
                <Avatar kind={seatedAvatar(p, user)} color={seatedColor(p, user)} size={36} />
                <strong>{p.displayName}</strong>
                {p.isHost ? <span className="badge badgeHost">Host</span> : null}
                {p.ready ? (
                  <span className="badge badgeOk">
                    <IconCheck color="#16A34A" /> Ready
                  </span>
                ) : (
                  <span className="badge">Waiting</span>
                )}
              </div>
            ))}
          </div>
        </div>
        <div style={{ width: minWidth, marginTop: 22 }}>
          <Button block variant="secondary" onClick={() => setLobbyReady(!myReady)}>
            {myReady ? 'Unready' : 'Ready'}
          </Button>
          <Button block disabled={!canStart} onClick={() => startGame()}>
            Start Game
          </Button>
          <p className="hint">Only the host can start. Ready is optional.</p>
          <div style={{ textAlign: 'center' }}>
            <Button
              variant="dangerText"
              onClick={async () => {
                await leaveRoom()
                nav('/lobby')
              }}
            >
              <IconLogout color="#DC2626" /> Leave room
            </Button>
          </div>
        </div>
      </div>
    </AppShell>
  )
}

const minWidth = 'min(420px, 100%)'
