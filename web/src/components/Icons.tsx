import type { CSSProperties } from 'react'
import { svgAsset } from '../assets/svg'

export type IconProps = {
  size?: number
  color?: string
  className?: string
  title?: string
}

export function SvgIcon({
  name,
  size = 18,
  color,
  className,
  title,
}: IconProps & { name: string }) {
  const style: CSSProperties = { width: size, height: size, color }
  return (
    <span
      className={className ? `svgIcon ${className}` : 'svgIcon'}
      role={title ? 'img' : undefined}
      aria-label={title}
      aria-hidden={title ? undefined : true}
      style={style}
      dangerouslySetInnerHTML={{ __html: svgAsset(name) }}
    />
  )
}

export function IconAlert(p: IconProps) {
  return <SvgIcon name="icon-alert" {...p} size={p.size ?? 22} />
}
export function IconArrowLeft(p: IconProps) {
  return <SvgIcon name="icon-arrow-left" {...p} size={p.size ?? 18} />
}
export function IconCheck(p: IconProps) {
  return <SvgIcon name="icon-check" {...p} size={p.size ?? 14} />
}
export function IconChevronRight(p: IconProps) {
  return <SvgIcon name="icon-chevron-right" {...p} size={p.size ?? 16} />
}
export function IconCircleX(p: IconProps) {
  return <SvgIcon name="icon-circle-x" {...p} size={p.size ?? 18} />
}
export function IconClock(p: IconProps) {
  return <SvgIcon name="icon-clock" {...p} size={p.size ?? 16} />
}
export function IconCopy(p: IconProps) {
  return <SvgIcon name="icon-copy" {...p} size={p.size ?? 16} />
}
export function IconEye(p: IconProps) {
  return <SvgIcon name="icon-eye" {...p} size={p.size ?? 18} />
}
export function IconFlame(p: IconProps) {
  return <SvgIcon name="icon-flame" {...p} size={p.size ?? 18} />
}
export function IconGmail(p: IconProps) {
  return <SvgIcon name="icon-gmail" {...p} size={p.size ?? 18} />
}
export function IconHistory(p: IconProps) {
  return <SvgIcon name="icon-history" {...p} size={p.size ?? 18} />
}
export function IconHourglass(p: IconProps) {
  return <SvgIcon name="icon-hourglass" {...p} size={p.size ?? 14} />
}
export function IconKey(p: IconProps) {
  return <SvgIcon name="icon-key" {...p} size={p.size ?? 18} />
}
export function IconLock(p: IconProps) {
  return <SvgIcon name="icon-lock" {...p} size={p.size ?? 18} />
}
export function IconLogout(p: IconProps) {
  return <SvgIcon name="icon-logout" {...p} size={p.size ?? 16} />
}
export function IconMinus(p: IconProps) {
  return <SvgIcon name="icon-minus" {...p} size={p.size ?? 18} />
}
export function IconPercent(p: IconProps) {
  return <SvgIcon name="icon-percent" {...p} size={p.size ?? 18} />
}
export function IconPlus(p: IconProps) {
  return <SvgIcon name="icon-plus" {...p} size={p.size ?? 18} />
}
export function IconRefresh(p: IconProps) {
  return <SvgIcon name="icon-refresh" {...p} size={p.size ?? 16} />
}
export function IconStats(p: IconProps) {
  return <SvgIcon name="icon-stats" {...p} size={p.size ?? 18} />
}
export function IconTrophy(p: IconProps) {
  return <SvgIcon name="icon-trophy" {...p} size={p.size ?? 22} />
}
export function IconUser(p: IconProps) {
  return <SvgIcon name="icon-user" {...p} size={p.size ?? 18} />
}
export function IconUserX(p: IconProps) {
  return <SvgIcon name="icon-user-x" {...p} size={p.size ?? 22} />
}
export function IconUsers(p: IconProps) {
  return <SvgIcon name="icon-users" {...p} size={p.size ?? 18} />
}
export function IconWifiOff(p: IconProps) {
  return <SvgIcon name="icon-wifi-off" {...p} size={p.size ?? 22} />
}
export function IconX(p: IconProps) {
  return <SvgIcon name="icon-x" {...p} size={p.size ?? 18} />
}

/** No Penpot mail asset — Register / Forgot only. */
export function IconMail({ size = 18, color = 'currentColor' }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="1.8" aria-hidden>
      <rect x="3" y="5" width="18" height="14" rx="2" />
      <path d="m4 7 8 6 8-6" />
    </svg>
  )
}
