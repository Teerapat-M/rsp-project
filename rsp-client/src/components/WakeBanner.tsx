import { useEffect, useState } from 'react'
import { WAKE_MAX_ATTEMPTS, wakeCopy, type WakeState } from '../lib/wake'
import { getWakeState, subscribeWake } from '../lib/wakeBus'

export function WakeBanner() {
  const [state, setState] = useState<WakeState>(getWakeState)
  useEffect(() => subscribeWake(setState), [])
  if (!state.waking) return null
  return (
    <div className="wakeBanner" role="status">
      {wakeCopy(state.attempt, WAKE_MAX_ATTEMPTS)}
    </div>
  )
}
