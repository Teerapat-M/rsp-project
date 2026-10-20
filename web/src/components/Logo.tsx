import { Hand } from './Hand'

type Props = {
  size?: 'sm' | 'md' | 'lg'
  stacked?: boolean
  darkWord?: boolean
}

const wrap: Record<NonNullable<Props['size']>, number> = { sm: 52, md: 86, lg: 148 }
const hand: Record<NonNullable<Props['size']>, number> = { sm: 18, md: 28, lg: 44 }

export function Logo({ size = 'md', stacked, darkWord = true }: Props) {
  const box = wrap[size]
  const hs = hand[size]
  return (
    <div className={`logo ${stacked ? 'logoStacked' : ''}`}>
      <div className="logoMark" style={{ width: box, height: box, borderRadius: box * 0.28 }}>
        <Hand move="rock" size={hs} />
        <Hand move="scissors" size={hs} />
        <Hand move="paper" size={hs} />
      </div>
      {stacked ? (
        <div className={`logoWord ${darkWord ? '' : 'logoWordLight'}`}>RSP</div>
      ) : null}
    </div>
  )
}

export function LogoMini() {
  return (
    <div className="logoMini" aria-label="RSP">
      <Hand move="rock" size={36} />
      <Hand move="scissors" size={36} />
      <Hand move="paper" size={36} />
    </div>
  )
}
