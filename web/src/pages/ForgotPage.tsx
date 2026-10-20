import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { Button } from '../components/Button'
import { Input } from '../components/Input'
import { Logo } from '../components/Logo'
import { HeaderWave } from '../components/Wave'
import { IconMail } from '../components/Icons'
import { api } from '../api'

export function ForgotPage() {
  const [email, setEmail] = useState('')
  const [done, setDone] = useState('')
  const [resetUrl, setResetUrl] = useState('')

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    const res = await api.forgotPassword(email)
    setDone(res.message)
    setResetUrl(res.resetUrl ?? '')
  }

  return (
    <div className="shell">
      <HeaderWave />
      <main className="pageCenter shellMain">
        <div className="authLogoFloat">
          <Logo size="sm" />
        </div>
        <h1 className="heroTitle">Forgot password</h1>
        <p className="heroSub">We&apos;ll send a reset link if that email is on an account</p>
        <form className="authForm" onSubmit={onSubmit}>
          <Input icon={<IconMail />} placeholder="Email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          <Button type="submit" block>
            Send reset link
          </Button>
          {done ? <p className="hint">{done}</p> : null}
          {resetUrl ? (
            <p className="hint">
              Dev reset URL: <a href={resetUrl}>{resetUrl}</a>
            </p>
          ) : null}
          <p className="mutedLink">
            <Link to="/login">← Back to login</Link>
          </p>
        </form>
      </main>
    </div>
  )
}
