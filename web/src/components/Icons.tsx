type IconProps = { size?: number; color?: string }

export function IconUser({ size = 18, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8">
      <circle cx="12" cy="8" r="3.2" />
      <path d="M5 19c1.6-3.2 4-4.8 7-4.8S17.4 15.8 19 19" strokeLinecap="round" />
    </svg>
  )
}

export function IconLock({ size = 18, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8">
      <rect x="5" y="11" width="14" height="10" rx="2" />
      <path d="M8 11V8a4 4 0 0 1 8 0v3" strokeLinecap="round" />
    </svg>
  )
}

export function IconEye({ size = 18, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8">
      <path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12z" />
      <circle cx="12" cy="12" r="2.5" />
    </svg>
  )
}

export function IconKey({ size = 18, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8">
      <circle cx="8" cy="14" r="4" />
      <path d="M11.5 12.5 20 5v3M16 5v3" strokeLinecap="round" />
    </svg>
  )
}

export function IconMail({ size = 18, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8">
      <rect x="3" y="5" width="18" height="14" rx="2" />
      <path d="m4 7 8 6 8-6" />
    </svg>
  )
}

export function IconCopy({ size = 16, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8">
      <rect x="9" y="9" width="11" height="11" rx="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </svg>
  )
}

export function IconStats({ size = 18, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8">
      <path d="M4 19V9M12 19V5M20 19v-7" strokeLinecap="round" />
    </svg>
  )
}

export function IconHistory({ size = 18, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8">
      <path d="M12 7v6l4 2" strokeLinecap="round" />
      <path d="M4.5 12a7.5 7.5 0 1 0 2-5.2L4 9" strokeLinecap="round" />
    </svg>
  )
}

export function IconGoogle({ size = 18 }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24">
      <path fill="#EA4335" d="M12 10.2v3.6h5.1c-.2 1.2-1.5 3.6-5.1 3.6-3.1 0-5.6-2.5-5.6-5.6S8.9 6.2 12 6.2c1.7 0 2.9.7 3.6 1.4l2.4-2.4C16.5 3.7 14.4 2.8 12 2.8 6.9 2.8 2.8 6.9 2.8 12S6.9 21.2 12 21.2c5.4 0 9-3.8 9-9.1 0-.6-.1-1.1-.2-1.9H12z" />
    </svg>
  )
}

export function IconClock({ size = 16, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8">
      <circle cx="12" cy="12" r="8" />
      <path d="M12 8v5l3 2" strokeLinecap="round" />
    </svg>
  )
}

export function IconPlayers({ size = 18, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8">
      <circle cx="9" cy="8" r="2.6" />
      <circle cx="16" cy="9" r="2.2" />
      <path d="M4.5 18c.8-3 2.6-4.5 4.5-4.5s3.7 1.5 4.5 4.5M14 13.6c1.4 0 3 .9 3.8 3.4" strokeLinecap="round" />
    </svg>
  )
}

export function IconLeave({ size = 16, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8">
      <path d="M10 6H7a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h3M13 12h8M18 9l3 3-3 3" strokeLinecap="round" />
    </svg>
  )
}

export function IconCheck({ size = 14, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="2.4">
      <path d="m5 12 5 5 9-10" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

export function IconWifiOff({ size = 22, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8">
      <path d="m2 8 20 12M7 11a9 9 0 0 1 3-1.2M5 16a6 6 0 0 1 3.2-1.5M12 20h.01M17 11.5a9 9 0 0 0-1.6-.7" strokeLinecap="round" />
    </svg>
  )
}

export function IconWarn({ size = 22, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8">
      <path d="M12 4 3 20h18L12 4z" strokeLinejoin="round" />
      <path d="M12 10v5M12 17.5h.01" strokeLinecap="round" />
    </svg>
  )
}

export function IconTrophy({ size = 22, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8">
      <path d="M8 5h8v5a4 4 0 0 1-8 0V5z" />
      <path d="M8 7H5a3 3 0 0 0 3 4M16 7h3a3 3 0 0 1-3 4M12 14v2M9 19h6" strokeLinecap="round" />
    </svg>
  )
}

export function IconLockSmall({ size = 14, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="2">
      <rect x="6" y="11" width="12" height="9" rx="2" />
      <path d="M8.5 11V8.5a3.5 3.5 0 0 1 7 0V11" />
    </svg>
  )
}
