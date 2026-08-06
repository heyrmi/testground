import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'
import { clampInt } from '../search'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/toast',
  validateSearch: (search: Record<string, unknown>) => ({
    dismissMs: clampInt(search.dismissMs, 3000, 100, 60_000),
  }),
  component: DisappearingToast,
})

function DisappearingToast() {
  const { dismissMs } = route.useSearch()
  const [live, setLive] = useState<number[]>([])
  const [shown, setShown] = useState(0)
  const [lastDismissed, setLastDismissed] = useState(0)

  const nextSequence = useRef(0)
  const timers = useRef(new Set<ReturnType<typeof setTimeout>>())

  useEffect(() => {
    const pending = timers.current
    return () => {
      pending.forEach(clearTimeout)
      pending.clear()
    }
  }, [])

  function show() {
    const sequence = (nextSequence.current += 1)
    setShown(sequence)
    setLive((current) => [...current, sequence])

    const timer = setTimeout(() => {
      timers.current.delete(timer)
      setLive((current) => current.filter((seq) => seq !== sequence))
      setLastDismissed(sequence)
    }, dismissMs)
    timers.current.add(timer)
  }

  return (
    <ChallengePage id="toast">
      <p className="stage__label">
        Each toast removes itself after <b data-testid="dismiss-ms">{dismissMs}</b> ms
      </p>

      <button className="primary" data-testid="show-toast" onClick={show}>
        Show a toast
      </button>

      <dl className="mt-6 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        <dt className="text-muted">Toasts shown</dt>
        <dd data-testid="toast-count">{shown}</dd>
        <dt className="text-muted">Last dismissed</dt>
        <dd data-testid="toast-last">{lastDismissed}</dd>
        <dt className="text-muted">Visible now</dt>
        <dd data-testid="toast-live">{live.length}</dd>
      </dl>

      {createPortal(
        <div className="fixed bottom-6 right-6 flex flex-col gap-2 z-50" data-testid="toast-region">
          {live.map((sequence) => (
            <output
              key={sequence}
              data-testid="toast"
              data-sequence={sequence}
              id={`toast-${sequence}`}
              className="rounded-lg border border-line bg-surface px-4 py-3 text-sm shadow-lg"
            >
              Saved change #{sequence}
            </output>
          ))}
        </div>,
        document.body,
      )}
    </ChallengePage>
  )
}
