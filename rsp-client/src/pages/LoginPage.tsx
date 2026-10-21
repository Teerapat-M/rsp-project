import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Button } from '../components/Button'
import { Input } from '../components/Input'
import { Logo } from '../components/Logo'
import { HeaderWave } from '../components/Wave'
import { IconEye, IconGmail, IconLock, IconUser } from '../components/Icons'
import { useAuth } from '../context/AuthContext'

const googleClientId = import.meta.env.VITE_GOOGLE_CLIENT_ID ?? ''
const showGoogle = Boolean(googleClientId)

function loadGis(): Promise<void> {
  if (window.google?.accounts?.id) return Promise.resolve()
  return new Promise((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>('script[data-gis]')
    if (existing) {
      existing.addEventListener('load', () => resolve())
      existing.addEventListener('error', () => reject(new Error('Could not load Google')))
      return
    }
    const s = document.createElement('script')
    s.src = 'https://accounts.google.com/gsi/client'
    s.async = true
    s.dataset.gis = '1'
    s.onload = () => resolve()
    s.onerror = () => reject(new Error('Could not load Google'))
    document.head.appendChild(s)
  })
}

export function LoginPage() {
  const { login, google } = useAuth()
  const nav = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(false)
  const [show, setShow] = useState(false)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      await login(username, password, remember)
      nav('/lobby')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sign in failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="shell">
      <HeaderWave />
      <main className="pageCenter shellMain">
        <div className="authLogoFloat">
          <Logo size="sm" />
        </div>
        <h1 className="heroTitle">Welcome</h1>
        <p className="heroSub">Sign in to continue</p>
        <form className="authForm" onSubmit={onSubmit}>
          <Input
            icon={<IconUser />}
            placeholder="Username"
            autoComplete="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
          />
          <Input
            icon={<IconLock />}
            placeholder="Password"
            type={show ? 'text' : 'password'}
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            suffix={
              <button type="button" onClick={() => setShow((v) => !v)} aria-label="Toggle password">
                <IconEye />
              </button>
            }
          />
          <div className="rowBetween">
            <label className="check">
              <input type="checkbox" checked={remember} onChange={(e) => setRemember(e.target.checked)} />
              Remember me
            </label>
            <Link to="/forgot">Forgot password?</Link>
          </div>
          {error ? <div className="hint" style={{ color: '#dc2626' }}>{error}</div> : null}
          <Button type="submit" block disabled={busy}>
            Sign In
          </Button>
          <div className="divider">or</div>
          {showGoogle ? (
            <Button
              variant="secondary"
              block
              type="button"
              onClick={async () => {
                setBusy(true)
                setError('')
                try {
                  await loadGis()
                  const idToken = await new Promise<string>((resolve, reject) => {
                    if (!window.google?.accounts?.id) {
                      reject(new Error('Google is not configured'))
                      return
                    }
                    window.google.accounts.id.initialize({
                      client_id: googleClientId,
                      callback: (r) => {
                        if (r.credential) resolve(r.credential)
                        else reject(new Error('Google sign-in cancelled'))
                      },
                    })
                    window.google.accounts.id.prompt()
                  })
                  await google(idToken)
                  nav('/lobby')
                } catch (err) {
                  setError(err instanceof Error ? err.message : 'Google sign-in failed')
                } finally {
                  setBusy(false)
                }
              }}
            >
              <IconGmail /> Continue with Gmail
            </Button>
          ) : null}
          <Button variant="secondary" block to="/guest">
            <IconUser /> Continue as Guest
          </Button>
          <p className="mutedLink">
            Don&apos;t have an account? <Link to="/register">Register</Link>
          </p>
        </form>
      </main>
    </div>
  )
}
