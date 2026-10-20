import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { AppShell } from '../components/AppShell'
import { Button } from '../components/Button'
import { Input } from '../components/Input'
import { Modal } from '../components/Modal'
import { IconHistory, IconKey, IconPlayers, IconStats, IconWarn } from '../components/Icons'
import { useGame } from '../context/GameContext'
import { formatPasscode, onlyDigits } from '../lib/format'

export function LobbyPage() {
  const nav = useNavigate()
  const { joinRoom, joinError, clearJoinError } = useGame()
  const [code, setCode] = useState('')

  async function onJoin(e: FormEvent) {
    e.preventDefault()
    const room = await joinRoom(onlyDigits(code))
    if (room) nav(`/room/${room.id}`)
  }

  return (
    <AppShell>
      <div className="pageCenter">
        <h1 className="heroTitle">Ready to play?</h1>
        <p className="heroSub">Create a room or join a friend</p>
        <div className="lobbyGrid">
          <div className="panel lobbyCard">
            <h3>Create Room</h3>
            <p>Start a new match and share the passcode</p>
            <Button block to="/create">
              Create Room
            </Button>
          </div>
          <form className="panel lobbyCard" onSubmit={onJoin}>
            <h3>Join Room</h3>
            <p>Enter the 6-digit passcode from your friend</p>
            <div className="joinRow">
              <Input
                icon={<IconKey />}
                placeholder="Enter passcode"
                inputMode="numeric"
                value={formatPasscode(code)}
                onChange={(e) => setCode(onlyDigits(e.target.value))}
              />
              <Button type="submit">Join</Button>
            </div>
          </form>
        </div>
        <div className="lobbyLinks">
          <Button variant="ghost" to="/stats">
            <IconStats /> Stats
          </Button>
          <Button variant="ghost" to="/history">
            <IconHistory /> History
          </Button>
        </div>
      </div>
      <Modal
        open={joinError?.code === 'WRONG_PASSCODE'}
        tone="danger"
        icon={<IconWarn />}
        title="Wrong passcode"
        body={`We couldn't find a room with passcode ${formatPasscode(joinError?.passcode ?? '')}. Check the code with your friend and try again.`}
        primaryLabel="Try again"
        secondaryLabel="Create a room"
        onPrimary={clearJoinError}
        onSecondary={() => {
          clearJoinError()
          nav('/create')
        }}
        onClose={clearJoinError}
      />
      <Modal
        open={joinError?.code === 'ROOM_FULL'}
        tone="warn"
        icon={<IconPlayers />}
        title="Room is full"
        body={`Room ${formatPasscode(joinError?.passcode ?? '')} already has 2 players. Ask your friend for a new code or create your own room.`}
        primaryLabel="Create a room"
        secondaryLabel="Back to lobby"
        onPrimary={() => {
          clearJoinError()
          nav('/create')
        }}
        onSecondary={clearJoinError}
        onClose={clearJoinError}
      />
    </AppShell>
  )
}
