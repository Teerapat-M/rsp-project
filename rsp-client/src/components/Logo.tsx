import { svgAsset } from '../assets/svg'

type Props = {
  size?: 'sm' | 'md' | 'lg'
  stacked?: boolean
  darkWord?: boolean
  variant?: 'mark' | 'badge'
}

const BOX: Record<NonNullable<Props['size']>, number> = { sm: 168, md: 148, lg: 168 }

export function Logo({ size = 'md', stacked, darkWord = true, variant }: Props) {
  const kind = variant ?? (size === 'sm' ? 'badge' : 'mark')
  const px = BOX[size]
  return (
    <div className={stacked ? 'logoStacked' : undefined}>
      <span
        className="logoSvg"
        style={{ width: px, height: px }}
        role="img"
        aria-label="RSP"
        dangerouslySetInnerHTML={{ __html: svgAsset(kind === 'badge' ? 'logo-badge' : 'logo-mark') }}
      />
      {stacked ? <div className={`logoWord ${darkWord ? '' : 'logoWordLight'}`}>RSP</div> : null}
    </div>
  )
}

export function LogoMini() {
  return (
    <span
      className="logoMini"
      role="img"
      aria-label="RSP"
      dangerouslySetInnerHTML={{ __html: svgAsset('logo-mini') }}
    />
  )
}
