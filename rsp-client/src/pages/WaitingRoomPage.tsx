import { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { AppShell } from '../components/AppShell'
import { Avatar } from '../components/Avatar'
import { Button } from '../components/Button'
import { IconCheck, IconClock, IconCopy, IconLogout, IconUsers } from '../components/Icons'
import { useAuth } from '../context/AuthContext'
import { useGame } from '../context/GameContext'
import { formatPasscode } from '../lib/format'

export function WaitingRoomPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const { room, loadRoom, startGame, leaveRoom, view } = useGame()
  const nav = useNavigate()
  const [copied, setCopied] = useState(false)
  const loadRoomRef = useRef(loadRoom)
  useEffect(() => {
    loadRoomRef.current = loadRoom
  })

  useEffect(() => {
    if (!id) return
    loadRoomRef.current(id).catch(() => nav('/lobby'))
  }, [id, nav])

  useEffect(() => {
    if (view && !view.finished) nav('/play')
  }, [view, nav])

  if (!room || !user) return null
  const isHost = room.hostId === user.id
  const canStart = isHost && room.players.length === 2

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
                <Avatar kind={p.avatar} color={p.color} size={36} />
                <strong>{p.displayName}</strong>
                {p.isHost ? (
                  <span className="badge badgeHost">Host</span>
                ) : (
                  <span className="badge badgeOk">
                    <IconCheck color="#16A34A" /> Joined
                  </span>
                )}
              </div>
            ))}
          </div>
        </div>
        <div style={{ width: minWidth, marginTop: 22 }}>
          <Button block disabled={!canStart} onClick={() => startGame()}>
            Start Game
          </Button>
          <p className="hint">Only the host can start</p>
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
