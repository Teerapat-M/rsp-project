import { useEffect, useState } from 'react'
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import { CSS } from '@dnd-kit/utilities'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { AppShell } from '../components/AppShell'
import { Avatar } from '../components/Avatar'
import { Button } from '../components/Button'
import { Card, FaceDown } from '../components/Card'
import { HAND_ORDER, Hand, moveLabel } from '../components/Hand'
import { Modal } from '../components/Modal'
import { Timer } from '../components/Timer'
import { IconCheck, IconClock, IconHourglass, IconLock, IconRefresh, IconTrophy, IconWifiOff } from '../components/Icons'
import { useAuth } from '../context/AuthContext'
import { useGame } from '../context/GameContext'
import type { PlayableMove } from '../types'

export function GamePage() {
  const { user } = useAuth()
  const game = useGame()
  const { attach } = game
  const nav = useNavigate()
  const [params] = useSearchParams()
  const roomQ = params.get('room')
  const demo = params.get('demo')

  useEffect(() => {
    if (roomQ) attach(roomQ)
  }, [roomQ, attach])

  useEffect(() => {
    if (game.view?.finished && !game.roundResult) nav('/result')
  }, [game.view, game.roundResult, nav])

  if (!user) return null
  if (game.roundResult) return <RoundResult />
  if (!game.view && !game.room) {
    return (
      <AppShell header={false}>
        <p className="hint">Connecting…</p>
      </AppShell>
    )
  }

  return <PlaySurface demo={demo} />
}

function PlaySurface({ demo }: { demo: string | null }) {
  const { user } = useAuth()
  const game = useGame()
  const view = game.view
  const nav = useNavigate()
  const [active, setActive] = useState<PlayableMove | null>(null)
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }))

  if (!user) return null

  const placed = (view?.yourMove && view.yourMove !== 'timeout' ? view.yourMove : '') as PlayableMove | ''
  const locked = Boolean(view?.yourReady)
  const dc = Boolean(view?.opponentDisconnected)
  const timeup = game.remaining === 0 && !placed && !locked && !dc
  const over = Boolean(active)

  function onDragStart(e: DragStartEvent) {
    if (locked || dc || timeup) return
    setActive(e.active.id as PlayableMove)
  }

  function onDragEnd(e: DragEndEvent) {
    setActive(null)
    if (locked || dc || timeup) return
    if (e.over?.id === 'slot' && typeof e.active.id === 'string') {
      game.submitMove(e.active.id as PlayableMove)
    }
  }

  const hint = dc
    ? `If ${view?.opponentName ?? 'they'} doesn’t return in 0:${String(view?.reconnectSeconds ?? 24).padStart(2, '0')} you can claim the win`
    : locked
      ? `You’re ready — cards reveal when ${view?.opponentName ?? 'the opponent'} locks in`
      : timeup
        ? 'Round lost on time — next round starts in 3…'
        : placed
          ? 'Press Ready to lock in your pick'
          : over
            ? 'Drag a card into the drop zone, then press Ready'
            : 'Drag a card into the drop zone, then press Ready'

  return (
    <AppShell header={false} wide>
      <DndContext sensors={sensors} onDragStart={onDragStart} onDragEnd={onDragEnd} onDragCancel={() => setActive(null)}>
        {view ? <ScoreBar /> : null}
        <div className="board">
          <div className="oppRow">
            <div className="oppCards">
              <FaceDown dim={dc} />
              <FaceDown dim={dc} />
              <FaceDown dim={dc} />
            </div>
            {dc ? (
              <span className="statusChip statusBad">
                <IconWifiOff size={14} /> Opponent disconnected
              </span>
            ) : view?.opponentReady ? (
              <span className="statusChip statusOk">
                <IconCheck /> Opponent is ready
              </span>
            ) : (
              <span className="statusChip statusWait">
                <IconHourglass /> Opponent is choosing…
              </span>
            )}
          </div>

          <DropSlot placed={placed} locked={locked} over={over} timeup={timeup} dc={dc} opponentName={view?.opponentName} reconnect={view?.reconnectSeconds} />

          <div className="handRow">
            {HAND_ORDER.map((m) => {
              const gone = placed === m
              if (gone) return <div key={m} className="handSlot" />
              return (
                <DraggableCard
                  key={m}
                  move={m}
                  disabled={locked || dc || timeup}
                  selected={false}
                  hidden={active === m}
                />
              )
            })}
          </div>

          {dc ? (
            <div className="actionsRow" style={{ width: 'min(420px, 100%)' }}>
              <Button block onClick={() => game.claimWin()}>
                Claim win
              </Button>
              <Button
                variant="secondary"
                block
                onClick={async () => {
                  await game.leaveRoom()
                  nav('/lobby')
                }}
              >
                Leave match
              </Button>
            </div>
          ) : (
            <div className="readyBtn">
              <Button
                block
                disabled={!placed || locked || timeup || over}
                onClick={() => game.readyUp()}
              >
                {timeup ? 'Revealing…' : locked ? 'Waiting for opponent…' : 'Ready'}
              </Button>
            </div>
          )}
          <p className="hint">{hint}</p>
        </div>
        <DragOverlay>{active ? <Card move={active} dragging /> : null}</DragOverlay>
      </DndContext>
      <Modal
        open={game.connectionLost || demo === 'connection'}
        tone="danger"
        icon={<IconWifiOff />}
        title="Connection lost"
        body="You're offline. Your match is paused and we'll keep trying to reconnect for 30 seconds."
        status={
          <>
            <IconRefresh /> Reconnecting… attempt {game.reconnectAttempt} of 5 • 0:21
          </>
        }
        primaryLabel="Retry now"
        secondaryLabel="Leave match"
        onPrimary={() => game.retryConnection()}
        onSecondary={async () => {
          await game.leaveRoom()
          nav('/lobby')
        }}
      />
    </AppShell>
  )
}

function DropSlot({
  placed,
  locked,
  over,
  timeup,
  dc,
  opponentName,
  reconnect,
}: {
  placed: PlayableMove | ''
  locked: boolean
  over: boolean
  timeup: boolean
  dc: boolean
  opponentName?: string
  reconnect?: number | null
}) {
  const { setNodeRef, isOver } = useDroppable({ id: 'slot' })
  const hot = over || isOver
  let cls = 'drop'
  if (dc) cls += ' dropDc'
  else if (timeup) cls += ' dropTime'
  else if (locked) cls += ' dropLocked'
  else if (placed) cls += ' dropPlaced'
  else if (hot) cls += ' dropOver'

  let title = 'Drag a card here'
  let sub = 'Pick Rock, Paper or Scissors'
  if (hot && !placed) {
    title = 'Release to play'
    sub = moveLabel(placed) || 'Scissors'
  }
  if (placed && !locked) {
    title = `${moveLabel(placed)} selected`
    sub = 'Drag another card here to swap it'
  }
  if (locked && placed) {
    title = `Locked in: ${moveLabel(placed)}`
    sub = 'Hidden until both players are ready'
  }
  if (timeup) {
    title = "Time's up!"
    sub = 'No card played — this round goes to your opponent'
  }
  if (dc) {
    title = `${opponentName ?? 'Opponent'} lost connection`
    sub = `Waiting for them to reconnect… 0:${String(reconnect ?? 24).padStart(2, '0')}`
  }

  return (
    <div ref={setNodeRef} className={cls}>
      {placed && !timeup && !dc ? (
        <>
          <Card move={placed} mini state={locked ? 'normal' : 'selected'} />
          <div>
            <h3>
              {locked ? <IconLock size={14} color="#16A34A" /> : null} {title}
            </h3>
            <p>{sub}</p>
          </div>
        </>
      ) : (
        <div>
          <h3>
            {timeup ? <IconClock size={16} color="#DC2626" /> : dc ? <IconWifiOff size={16} /> : null} {title}
          </h3>
          <p>{sub}</p>
        </div>
      )}
    </div>
  )
}

function DraggableCard({
  move,
  disabled,
  selected,
  hidden,
}: {
  move: PlayableMove
  disabled: boolean
  selected: boolean
  hidden: boolean
}) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: move,
    disabled,
  })
  const style = transform ? { transform: CSS.Translate.toString(transform) } : undefined
  return (
    <div ref={setNodeRef} style={style}>
      <Card
        move={move}
        state={disabled ? 'disabled' : selected ? 'selected' : 'normal'}
        ghost={hidden || isDragging}
        listeners={disabled ? undefined : listeners}
        attributes={attributes}
      />
    </div>
  )
}

function ScoreBar() {
  const { user } = useAuth()
  const { view, remaining, roundResult } = useGame()
  if (!user || !view) return null
  const youScore = view.scores[user.id] ?? 0
  const oppScore = view.scores[view.opponentId] ?? 0
  return (
    <div className="scoreBar">
      <div className="scoreSide">
        <Avatar kind={user.avatar} color={user.color} size={44} />
        <div className="scoreName">
          <strong>You</strong>
          <span>{user.displayName}</span>
        </div>
        <div className="scoreNum">{youScore}</div>
      </div>
      <div className="scoreMid">
        {roundResult ? null : <Timer remaining={remaining} total={view.turnSeconds} />}
        <div className="roundLabel">
          <strong>{roundResult ? `Round ${roundResult.roundNumber} result` : `Round ${view.roundNumber}`}</strong>
          <span>First to {view.winsNeeded}</span>
        </div>
      </div>
      <div className="scoreSide right">
        <div className="scoreNum">{oppScore}</div>
        <div className="scoreName">
          <strong>Opponent</strong>
          <span>{view.opponentName}</span>
        </div>
        <Avatar kind={view.opponentAvatar} color={view.opponentColor} size={44} />
      </div>
    </div>
  )
}

function RoundResult() {
  const { user } = useAuth()
  const { roundResult, view } = useGame()
  if (!user || !roundResult || !view) return null
  const title = roundResult.draw ? "It's a draw" : roundResult.youWon ? 'You win this round!' : 'You lose this round'
  const bannerCls = roundResult.draw ? 'banner bannerDraw' : roundResult.youWon ? 'banner' : 'banner bannerLose'
  return (
    <AppShell header={false}>
      <ScoreBar />
      <div className="pageCenter">
        <div className={bannerCls}>
          {roundResult.youWon ? <IconTrophy size={22} color="#F2B01E" /> : null} {title}
        </div>
        <div className="vsRow">
          <div className={`vsPick ${roundResult.youWon ? 'win' : ''}`}>
            <span>You</span>
            <div className="bigCard">
              {roundResult.yourMove === 'timeout' ? (
                <span>Timeout</span>
              ) : (
                <>
                  <Hand move={roundResult.yourMove as PlayableMove} size={72} />
                  {moveLabel(roundResult.yourMove)}
                </>
              )}
            </div>
            {roundResult.youWon ? <span className="badge badgeOk">Winner</span> : null}
          </div>
          <div className="vs">VS</div>
          <div className={`vsPick ${!roundResult.youWon && !roundResult.draw ? 'win' : ''}`}>
            <span>Opponent</span>
            <div className="bigCard">
              {roundResult.opponentMove === 'timeout' ? (
                <span>Timeout</span>
              ) : (
                <>
                  <Hand move={roundResult.opponentMove as PlayableMove} size={72} />
                  {moveLabel(roundResult.opponentMove)}
                </>
              )}
            </div>
            {!roundResult.youWon && !roundResult.draw ? (
              <span className="badge badgeOk">Winner</span>
            ) : (
              <span className="badge" style={{ background: '#fde8e8', color: '#dc2626' }}>
                Lost
              </span>
            )}
          </div>
        </div>
        <p className="hint">Next round in {roundResult.nextRoundIn}...</p>
      </div>
    </AppShell>
  )
}
