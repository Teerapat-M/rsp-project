import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { AVATAR_SWATCHES } from '../theme'
import { Button } from '../components/Button'
import { Input } from '../components/Input'
import { Avatar } from '../components/Avatar'
import { HAND_ORDER, Hand } from '../components/Hand'
import { HeaderWave } from '../components/Wave'
import { IconUser } from '../components/Icons'
import { useAuth } from '../context/AuthContext'
import type { AvatarKind } from '../types'

export function GuestPage() {
  const { guest } = useAuth()
  const nav = useNavigate()
  const [name, setName] = useState('')
  const [avatar, setAvatar] = useState<AvatarKind>('rock')
  const [color, setColor] = useState<string>(AVATAR_SWATCHES[0])
  const [error, setError] = useState('')

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await guest(name, avatar, color)
      nav('/lobby')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not continue')
    }
  }

  return (
    <div className="shell">
      <HeaderWave />
      <main className="pageCenter shellMain">
        <form className="guestLayout authForm" onSubmit={onSubmit} style={{ marginTop: 48 }}>
          <div className="heroHands">
            <Hand move="rock" size={64} />
            <Hand move="scissors" size={64} />
            <Hand move="paper" size={64} />
          </div>
          <h1 className="heroTitle">Welcome, Guest!</h1>
          <p className="heroSub">Choose a name to start playing</p>
          <Input
            label="Display name"
            icon={<IconUser />}
            placeholder="Enter your name"
            hint="3–16 characters"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <div className="avatarPick">
            <div>
              <Avatar kind={avatar} color={color} size={92} />
              <div className="hint">Preview</div>
            </div>
            <div>
              <div style={{ fontWeight: 500, marginBottom: 8 }}>Pick an avatar</div>
              <div style={{ display: 'flex', gap: 12 }}>
                {HAND_ORDER.map((m) => (
                  <button
                    key={m}
                    type="button"
                    className={`pickBtn ${avatar === m ? 'on' : ''}`}
                    onClick={() => setAvatar(m)}
                  >
                    <Hand move={m} tone={avatar === m ? 'color' : 'slate'} size={28} />
                  </button>
                ))}
              </div>
              <div style={{ display: 'flex', gap: 16, fontSize: 13, marginTop: 4 }}>
                {HAND_ORDER.map((m) => (
                  <span key={m} style={{ fontWeight: avatar === m ? 700 : 400, color: avatar === m ? '#111b3d' : '#5f6675' }}>
                    {m[0].toUpperCase() + m.slice(1)}
                  </span>
                ))}
              </div>
            </div>
          </div>
          <div style={{ fontWeight: 500 }}>Pick a color</div>
          <div className="swatches">
            {AVATAR_SWATCHES.map((c) => (
              <button
                key={c}
                type="button"
                className={`swatch ${color === c ? 'on' : ''}`}
                style={{ background: c }}
                onClick={() => setColor(c)}
                aria-label={c}
              />
            ))}
          </div>
          {error ? <div className="hint" style={{ color: '#dc2626' }}>{error}</div> : null}
          <Button type="submit" block>
            Start Playing
          </Button>
          <p className="mutedLink">
            <Link to="/login">← Back to login</Link>
          </p>
        </form>
      </main>
    </div>
  )
}
