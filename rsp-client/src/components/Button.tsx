import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { Link } from 'react-router-dom'
import styles from './Button.module.css'

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'dangerText'

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: Variant
  block?: boolean
  size?: 'md' | 'sm'
  to?: string
  children: ReactNode
}

export function Button({
  variant = 'primary',
  block,
  size = 'md',
  to,
  className,
  disabled,
  children,
  type = 'button',
  ...rest
}: Props) {
  const cls = [
    styles.btn,
    styles[variant],
    block ? styles.block : '',
    size === 'sm' ? styles.sm : '',
    disabled ? styles.disabled : '',
    className ?? '',
  ]
    .filter(Boolean)
    .join(' ')

  if (to && !disabled) {
    return (
      <Link className={cls} to={to}>
        {children}
      </Link>
    )
  }

  return (
    <button type={type} className={cls} disabled={disabled} {...rest}>
      {children}
    </button>
  )
}
