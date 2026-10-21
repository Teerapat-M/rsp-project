import type { ReactNode } from 'react'
import { Button } from './Button'
import { IconX } from './Icons'
import styles from './Modal.module.css'

type Tone = 'danger' | 'warn' | 'info'

type Props = {
  open: boolean
  title: string
  body: ReactNode
  icon?: ReactNode
  tone?: Tone
  status?: ReactNode
  primaryLabel: string
  secondaryLabel?: string
  onPrimary: () => void
  onSecondary?: () => void
  onClose?: () => void
}

export function Modal({
  open,
  title,
  body,
  icon,
  tone = 'danger',
  status,
  primaryLabel,
  secondaryLabel,
  onPrimary,
  onSecondary,
  onClose,
}: Props) {
  if (!open) return null
  return (
    <div className={styles.overlay} role="presentation" onClick={onClose}>
      <div
        className={styles.dialog}
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-title"
        onClick={(e) => e.stopPropagation()}
      >
        {onClose ? (
          <button className={styles.close} type="button" onClick={onClose} aria-label="Close">
            <IconX size={18} />
          </button>
        ) : null}
        {icon ? <div className={`${styles.icon} ${styles[tone]}`}>{icon}</div> : null}
        <h2 id="modal-title" className={styles.title}>
          {title}
        </h2>
        <div className={styles.body}>{body}</div>
        {status ? <div className={styles.status}>{status}</div> : null}
        <div className={styles.actions}>
          <Button onClick={onPrimary}>{primaryLabel}</Button>
          {secondaryLabel ? (
            <Button variant="secondary" onClick={onSecondary}>
              {secondaryLabel}
            </Button>
          ) : null}
        </div>
      </div>
    </div>
  )
}
