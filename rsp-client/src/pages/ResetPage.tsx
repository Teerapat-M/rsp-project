import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { Button } from '../components/Button'
import { Input } from '../components/Input'
import { Logo } from '../components/Logo'
import { HeaderWave } from '../components/Wave'
import { IconLock } from '../components/Icons'
import { api } from '../api'

export function ResetPage() {
  const [params] = useSearchParams()
  const nav = useNavigate()
  const token = params.get('token') ?? ''
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    try {
      await api.resetPassword(token, password)
      setDone(true)
      setTimeout(() => nav('/login'), 800)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not reset password')
    }
  }

  return (
    <div className="shell">
      <HeaderWave />
      <main className="pageCenter shellMain">
        <div className="authLogoFloat">
          <Logo size="sm" />
        </div>
        <h1 className="heroTitle">Reset password</h1>
        <p className="heroSub">Pick a new password for this account</p>
        <form className="authForm" onSubmit={onSubmit}>
          <Input
            icon={<IconLock />}
            placeholder="New password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          {error ? <div className="hint" style={{ color: '#dc2626' }}>{error}</div> : null}
          {done ? <p className="hint">Password updated. Taking you to sign in…</p> : null}
          <Button type="submit" block disabled={!token || done}>
            Update password
          </Button>
          <p className="mutedLink">
            <Link to="/login">← Back to login</Link>
          </p>
        </form>
      </main>
    </div>
  )
}
