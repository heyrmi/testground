import { useCallback, useEffect, useRef, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/kanban',
  component: Kanban,
})

interface Card {
  id: string
  title: string
  column: string
}

interface Column {
  id: string
  title: string
  limit: number
  cards: Card[]
}

interface Board {
  rev: number
  columns: Column[]
  watchers: number
}

interface Move {
  card: string
  column: string
  index: number
}

interface Refusal extends Move {
  reason: string
}

interface BoardEvent {
  seq: number
  kind: string
  data: Board
}

interface Place {
  column: string
  index: number
}

function socketURL(path: string): string {
  const url = new URL(path, window.location.href)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  return url.toString()
}

/** Where a card sits now, which is what a release is compared against. */
function placeOf(board: Board, cardId: string): Place | null {
  for (const column of board.columns) {
    const index = column.cards.findIndex((card) => card.id === cardId)
    if (index >= 0) return { column: column.id, index }
  }
  return null
}

/**
 * The offline board. It enforces none of the server's rules on purpose: the
 * column limit and the one-way done column are conditions of the board as the
 * server holds it, and a browser that has stopped talking to the server cannot
 * know either one still holds. Applying the move anyway is what makes the
 * offline board a plausible fiction rather than an obviously broken one.
 */
function applyLocally(board: Board, move: Move): Board {
  const card = board.columns.flatMap((column) => column.cards).find((one) => one.id === move.card)
  if (!card) return board

  const columns = board.columns.map((column) => ({
    ...column,
    cards: column.cards.filter((one) => one.id !== move.card),
  }))
  const target = columns.find((column) => column.id === move.column)
  if (!target) return board

  const at = Math.min(Math.max(move.index, 0), target.cards.length)
  target.cards.splice(at, 0, { ...card, column: target.id })
  return { ...board, columns }
}

function Kanban() {
  const [board, setBoard] = useState<Board | null>(null)
  const [online, setOnline] = useState(true)
  const [socketOpen, setSocketOpen] = useState(false)
  const [lastEvent, setLastEvent] = useState('')
  const [queue, setQueue] = useState<Move[]>([])
  const [refused, setRefused] = useState<Refusal[]>([])
  const [flushNote, setFlushNote] = useState('')
  const [refusal, setRefusal] = useState('')
  const [drag, setDrag] = useState<(Place & { card: string }) | null>(null)
  const [lastDrop, setLastDrop] = useState('')
  const [changedMidDrag, setChangedMidDrag] = useState('')

  // The socket's message handler is created once per connection and would
  // otherwise close over the drag as it stood when the socket opened, which is
  // never the drag that is actually in progress.
  const dragging = useRef<(Place & { card: string }) | null>(null)

  // Every snapshot the server hands out is complete and carries the sequence
  // number it was taken at, so the newest one wins outright. Without this an
  // in-flight reply to our own write can land after a broadcast that already
  // superseded it and put the board back an edit.
  const lastSeq = useRef(-1)
  const apply = useCallback((seq: number, next: Board) => {
    if (seq < lastSeq.current) return
    lastSeq.current = seq
    setBoard(next)
  }, [])

  useEffect(() => {
    if (!online) return

    const socket = new WebSocket(socketURL('/api/app/kanban/socket'))
    socket.addEventListener('open', () => setSocketOpen(true))
    socket.addEventListener('close', () => setSocketOpen(false))
    socket.addEventListener('message', (message) => {
      const event = JSON.parse(message.data as string) as BoardEvent
      setLastEvent(event.kind)
      if (dragging.current && event.kind === 'board') {
        setChangedMidDrag(`the board moved to revision ${event.data.rev} while this card was held`)
      }
      apply(event.seq, event.data)
    })

    return () => socket.close()
  }, [online, apply])

  // A first read for the case where the socket never opens, guarded by the
  // same sequence number so it cannot overwrite anything the socket already
  // delivered.
  useEffect(() => {
    fetch('/api/app/kanban/board')
      .then((res) => res.json() as Promise<{ seq: number; board: Board }>)
      .then((body) => apply(body.seq, body.board))
      .catch(() => undefined)
  }, [apply])

  async function send(move: Move) {
    const res = await fetch('/api/app/kanban/moves', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(move),
    })
    const body = (await res.json()) as { seq: number; board: Board; error?: string }
    setRefusal(res.ok ? '' : String(body.error))
    apply(body.seq, body.board)
  }

  function commit(move: Move) {
    setLastDrop(`${move.card} to ${move.column} ${move.index}`)
    if (online) {
      void send(move)
      return
    }
    setQueue((current) => [...current, move])
    setBoard((current) => (current ? applyLocally(current, move) : current))
  }

  function goOffline() {
    setOnline(false)
    setRefusal('')
    setFlushNote('')
    setRefused([])
  }

  async function reconnect() {
    const pending = queue
    setQueue([])
    setOnline(true)
    if (pending.length === 0) return

    const res = await fetch('/api/app/kanban/queue', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ moves: pending }),
    })
    const body = (await res.json()) as {
      applied: number
      refused: Refusal[]
      seq: number
      board: Board
    }
    setRefused(body.refused)
    setFlushNote(
      `${body.applied} of ${pending.length} queued moves applied, ${body.refused.length} refused`,
    )
    apply(body.seq, body.board)
  }

  function hold(event: React.PointerEvent<HTMLLIElement>, card: Card, index: number) {
    event.currentTarget.setPointerCapture(event.pointerId)
    setChangedMidDrag('')
    const started = { card: card.id, column: card.column, index }
    dragging.current = started
    setDrag(started)
  }

  // The target is recomputed on every move and then left alone until the
  // release. That is the hazard: a card arriving in the target column between
  // the last move and the release shifts everything below it, and the index
  // captured here now names a different gap.
  function track(event: React.PointerEvent<HTMLLIElement>) {
    const held = dragging.current
    if (!held || !board) return

    const under = document.elementFromPoint(event.clientX, event.clientY)
    const columnId = under?.closest<HTMLElement>('[data-testid="column"]')?.dataset.column
    if (!columnId) return

    const overId = under?.closest<HTMLElement>('[data-testid="card"]')?.dataset.cardId
    if (overId === held.card) return

    const cards = (board.columns.find((column) => column.id === columnId)?.cards ?? []).filter(
      (card) => card.id !== held.card,
    )
    const found = overId === undefined ? -1 : cards.findIndex((card) => card.id === overId)
    const next = { card: held.card, column: columnId, index: found >= 0 ? found : cards.length }
    dragging.current = next
    setDrag(next)
  }

  function release() {
    const held = dragging.current
    dragging.current = null
    setDrag(null)
    if (!held || !board) return

    // A press and release with no travel is not a move, and sending one would
    // advance the revision every watcher is using to tell changes apart.
    const now = placeOf(board, held.card)
    if (now && now.column === held.column && now.index === held.index) return
    commit({ card: held.card, column: held.column, index: held.index })
  }

  const connection = !online ? 'offline' : socketOpen ? 'online' : 'connecting'

  return (
    <ChallengePage id="kanban">
      <p className="stage__label">
        connection <b data-testid="connection-state">{connection}</b> · revision{' '}
        <b data-testid="board-rev">{board?.rev ?? 0}</b> · watching{' '}
        <b data-testid="watchers">{board?.watchers ?? 0}</b> · last event{' '}
        <b data-testid="last-event">{lastEvent || 'none'}</b> · queued{' '}
        <b data-testid="queued-count">{queue.length}</b>
      </p>

      <div className="flex flex-wrap items-center gap-3">
        <button data-testid="offline-toggle" onClick={() => (online ? goOffline() : void reconnect())}>
          {online ? 'Go offline' : 'Reconnect and flush'}
        </button>
        {drag && (
          <span className="font-mono text-xs text-muted" data-testid="drop-target">
            {drag.column} {drag.index}
          </span>
        )}
        {lastDrop && (
          <span className="font-mono text-xs text-muted" data-testid="last-drop">
            asked for {lastDrop}
          </span>
        )}
      </div>

      {changedMidDrag && (
        <p className="mt-3 text-sm" data-testid="board-changed">
          {changedMidDrag}. The release will still use the position chosen before it.
        </p>
      )}
      {refusal && (
        <p className="mt-3 text-sm" data-testid="refusal">
          {refusal}
        </p>
      )}
      {flushNote && (
        <p className="mt-3 text-sm" data-testid="flush-note">
          {flushNote}
        </p>
      )}
      {refused.length > 0 && (
        <ul className="mt-2 list-none p-0 text-sm">
          {refused.map((move, at) => (
            // Keyed by position as well as card: one card can be refused twice
            // in a single flush -- two queued moves of the same finished card
            // both bounce off the one-way column -- and duplicate keys make
            // React reuse the wrong row.
            <li key={`${move.card}-${at}`} data-testid="refused-move" data-card={move.card}>
              {move.card} stayed where it was: {move.reason}
            </li>
          ))}
        </ul>
      )}
      {queue.length > 0 && (
        <ol className="mt-3 list-none p-0 font-mono text-xs text-muted">
          {queue.map((move, at) => (
            <li key={`${move.card}-${at}`} data-testid="queued-move" data-card={move.card}>
              {at + 1}. {move.card} to {move.column} {move.index}
            </li>
          ))}
        </ol>
      )}

      <div className="mt-6 grid gap-4 md:grid-cols-3">
        {board?.columns.map((column) => (
          <section
            key={column.id}
            data-testid="column"
            data-column={column.id}
            // A release over the empty part of a column has to finish the drag
            // too, or a card dropped into a column that has room but no card to
            // aim at is held for ever.
            onPointerUp={release}
            className="rounded-lg border border-line p-3"
          >
            <h2 className="m-0 text-xs font-semibold uppercase tracking-wide text-muted">
              {column.title} · <b data-testid="column-count">{column.cards.length}</b>
              {column.limit > 0 && (
                <>
                  {' '}
                  of at most <b data-testid="column-limit">{column.limit}</b>
                </>
              )}
            </h2>

            <ul className="m-0 mt-3 flex min-h-40 list-none flex-col gap-2 p-0">
              {column.cards.map((card, index) => (
                <li
                  key={card.id}
                  data-testid="card"
                  data-card-id={card.id}
                  data-column={column.id}
                  data-position={index}
                  onPointerDown={(event) => hold(event, card, index)}
                  onPointerMove={track}
                  onPointerUp={release}
                  onPointerCancel={release}
                  className={`cursor-grab touch-none select-none rounded-md border border-line bg-sunken px-3 py-2 text-sm ${
                    drag?.card === card.id ? 'opacity-50' : ''
                  }`}
                >
                  {card.title}
                </li>
              ))}
            </ul>
          </section>
        ))}
      </div>

      <p className="mt-4 text-sm text-muted">
        Doing holds two cards and done is one way. Both rules live on the server, so the board
        you see while offline knows neither of them.
      </p>
    </ChallengePage>
  )
}
