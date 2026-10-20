import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Logo } from '../components/Logo'
import { useAuth } from '../context/AuthContext'

export function SplashPage() {
  const nav = useNavigate()
  const { user, ready } = useAuth()

  useEffect(() => {
    if (!ready) return
    const t = setTimeout(() => nav(user ? '/lobby' : '/login', { replace: true }), 1400)
    return () => clearTimeout(t)
  }, [ready, user, nav])

  return (
    <div className="splash">
      <div className="splashInner">
        <Logo size="lg" stacked darkWord={false} />
        <div className="splashTag">Rock · Scissors · Paper</div>
        <div className="dots" aria-hidden>
          <span className="on" />
          <span />
          <span />
        </div>
      </div>
    </div>
  )
}
