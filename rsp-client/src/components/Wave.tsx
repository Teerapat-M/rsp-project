export function HeaderWave() {
  return (
    <svg className="headerWave" viewBox="0 0 1440 240" preserveAspectRatio="none" aria-hidden>
      <defs>
        <linearGradient id="hdr" x1="0" y1="0" x2="1" y2="0">
          <stop offset="0" stopColor="#C3D5FD" />
          <stop offset="1" stopColor="#B39DEA" />
        </linearGradient>
      </defs>
      <path fill="url(#hdr)" d="M0 0h1440v150c-180 70-420 20-720 40C420 210 180 230 0 190V0z" />
    </svg>
  )
}

export function SideWave() {
  return (
    <svg className="sideWave" viewBox="0 0 280 810" preserveAspectRatio="none" aria-hidden>
      <defs>
        <linearGradient id="side" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#B39DEA" />
          <stop offset="1" stopColor="#C2D3FD" />
        </linearGradient>
      </defs>
      <path fill="url(#side)" d="M280 0c-40 80-20 160-90 250-50 64-30 140-70 220-40 80-20 160 20 250 18 40 40 70 140 90V0z" />
    </svg>
  )
}

export function CornerWave() {
  return (
    <svg className="cornerWave" viewBox="0 0 200 200" preserveAspectRatio="none" aria-hidden>
      <defs>
        <linearGradient id="crn" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#B39DEA" />
          <stop offset="1" stopColor="#C2D3FD" />
        </linearGradient>
      </defs>
      <path fill="url(#crn)" d="M200 0c-40 10-80 40-90 90C96 150 40 180 0 200h200V0z" />
    </svg>
  )
}
