import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import type { User } from '../types'
import { Avatar } from './Avatar'

export function PlayerChip({ user }: { user: User }) {
  const [open, setOpen] = useState(false)
  const { logout } = useAuth()
  const nav = useNavigate()
  return (
    <div className="chipWrap">
      <button type="button" className="playerChip" onClick={() => setOpen((v) => !v)}>
        <Avatar kind={user.avatar} color={user.color} size={28} />
        <span>{user.displayName}</span>
      </button>
      {open ? (
        <div className="chipMenu">
          <button type="button" onClick={() => { setOpen(false); nav('/stats') }}>
            Stats
          </button>
          <button type="button" onClick={() => { setOpen(false); nav('/history') }}>
            History
          </button>
          <button
            type="button"
            onClick={async () => {
              setOpen(false)
              await logout()
              nav('/login')
            }}
          >
            Log out
          </button>
        </div>
      ) : null}
    </div>
  )
}
