import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { isMock } from '../api'
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
          {isMock() ? (
            <>
              <button type="button" onClick={() => { setOpen(false); nav('/demo') }}>
                Screen gallery
              </button>
              <button
                type="button"
                onClick={() => {
                  setOpen(false)
                  window.dispatchEvent(new CustomEvent('rsp-simulate', { detail: 'opponent_dc' }))
                }}
              >
                Simulate opponent DC
              </button>
              <button
                type="button"
                onClick={() => {
                  setOpen(false)
                  window.dispatchEvent(new CustomEvent('rsp-simulate', { detail: 'connection_lost' }))
                }}
              >
                Simulate connection lost
              </button>
            </>
          ) : null}
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
