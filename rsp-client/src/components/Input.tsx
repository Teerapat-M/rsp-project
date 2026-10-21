import type { InputHTMLAttributes, ReactNode } from 'react'
import { useId } from 'react'
import styles from './Input.module.css'

type Props = InputHTMLAttributes<HTMLInputElement> & {
  label?: string
  hint?: string
  error?: string
  icon?: ReactNode
  suffix?: ReactNode
}

export function Input({ label, hint, error, icon, suffix, className, id, ...rest }: Props) {
  const autoId = useId()
  const inputId = id ?? autoId
  return (
    <label className={`${styles.wrap} ${error ? styles.error : ''} ${className ?? ''}`} htmlFor={inputId}>
      {label ? <span className={styles.label}>{label}</span> : null}
      <span className={styles.field}>
        {icon ? <span className={styles.icon}>{icon}</span> : null}
        <input id={inputId} className={styles.input} {...rest} />
        {suffix ? <span className={styles.suffix}>{suffix}</span> : null}
      </span>
      {error ? <span className={styles.err}>{error}</span> : hint ? <span className={styles.hint}>{hint}</span> : null}
    </label>
  )
}
