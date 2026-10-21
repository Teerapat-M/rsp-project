import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { LogoMini } from './Logo'
import { PlayerChip } from './PlayerChip'
import { CornerWave, HeaderWave, SideWave } from './Wave'

type Props = {
  children?: ReactNode
  wave?: 'side' | 'header' | 'none'
  header?: boolean
  wide?: boolean
}

export function AppShell({ children, wave = 'side', header = true, wide }: Props) {
  const { user } = useAuth()
  return (
    <div className={`shell ${wide ? 'shellWide' : ''}`}>
      {wave === 'header' ? <HeaderWave /> : null}
      {wave === 'side' ? (
        <>
          <SideWave />
          <CornerWave />
        </>
      ) : null}
      {header && user ? (
        <header className="topbar">
          <Link to="/lobby" className="topbarLogo">
            <LogoMini />
          </Link>
          <PlayerChip user={user} />
        </header>
      ) : null}
      <main className="shellMain">{children}</main>
    </div>
  )
}
