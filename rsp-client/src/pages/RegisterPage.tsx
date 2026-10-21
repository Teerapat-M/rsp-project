import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Button } from '../components/Button'
import { Input } from '../components/Input'
import { Logo } from '../components/Logo'
import { HeaderWave } from '../components/Wave'
import { IconLock, IconMail, IconUser } from '../components/Icons'
import { useAuth } from '../context/AuthContext'

export function RegisterPage() {
  const { register, user } = useAuth()
  const nav = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [email, setEmail] = useState('')
  const [error, setError] = useState('')

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await register(username, password, email || undefined)
      nav('/lobby')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not register')
    }
  }

  return (
    <div className="shell">
      <HeaderWave />
      <main className="pageCenter shellMain">
        <div className="authLogoFloat">
          <Logo size="sm" />
        </div>
        <h1 className="heroTitle">Create account</h1>
        <p className="heroSub">
          {user?.isGuest ? 'Keep this guest id and add a password' : 'Username, password, optional email'}
        </p>
        <form className="authForm" onSubmit={onSubmit}>
          <Input icon={<IconUser />} placeholder="Username" value={username} onChange={(e) => setUsername(e.target.value)} />
          <Input
            icon={<IconLock />}
            placeholder="Password (min 8 characters)"
            type="password"
            minLength={8}
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <Input icon={<IconMail />} placeholder="Email (optional)" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          {error ? <div className="hint" style={{ color: '#dc2626' }}>{error}</div> : null}
          <Button type="submit" block>
            Register
          </Button>
          <p className="mutedLink">
            <Link to="/login">← Back to login</Link>
          </p>
        </form>
      </main>
    </div>
  )
}
