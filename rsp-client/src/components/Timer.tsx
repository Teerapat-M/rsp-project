import styles from './Timer.module.css'

type Props = {
  remaining: number | null
  total: number
}

export function Timer({ remaining, total }: Props) {
  const low = remaining !== null && remaining <= 3
  const r = 22
  const c = 2 * Math.PI * r
  const ratio = remaining === null || total <= 0 ? 1 : Math.max(0, Math.min(1, remaining / total))
  const dash = `${c * ratio} ${c}`
  const color = low ? '#DC2626' : '#5B5FE6'
  const text = remaining === null ? '–' : `${remaining}s`
  return (
    <div className={`${styles.wrap} ${low ? styles.low : ''}`} aria-label={`Time remaining ${text}`}>
      <svg className={styles.svg} viewBox="0 0 56 56">
        <circle cx="28" cy="28" r={r} fill="none" stroke="#E6E8FD" strokeWidth="5" />
        <circle
          cx="28"
          cy="28"
          r={r}
          fill="none"
          stroke={color}
          strokeWidth="5"
          strokeLinecap="round"
          strokeDasharray={dash}
        />
      </svg>
      <span className={styles.label}>{text}</span>
    </div>
  )
}
