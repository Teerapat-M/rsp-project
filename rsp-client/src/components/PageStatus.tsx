import type { ReactNode } from 'react'
import { AppShell } from './AppShell'

export function PageStatus({ children }: { children: ReactNode }) {
  return (
    <AppShell>
      <p className="hint" role="status">
        {children}
      </p>
    </AppShell>
  )
}
