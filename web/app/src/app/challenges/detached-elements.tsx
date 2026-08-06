import { useEffect, useRef, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'
import { clampInt } from '../search'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/detached-elements',
  validateSearch: (search: Record<string, unknown>) => ({
    churnMs: clampInt(search.churnMs, 400, 50, 10_000),
  }),
  component: DetachedElements,
})

const ROWS = ['alpha', 'bravo', 'charlie', 'delta', 'echo']

function DetachedElements() {
  const { churnMs } = route.useSearch()
  const [churning, setChurning] = useState(false)
  const [generation, setGeneration] = useState(0)
  const [clicked, setClicked] = useState<string[]>([])
  const [vanishing, setVanishing] = useState(false)
  const [vanishClicks, setVanishClicks] = useState(0)

  useEffect(() => {
    if (!churning) return
    const timer = setInterval(() => setGeneration((n) => n + 1), churnMs)
    return () => clearInterval(timer)
  }, [churning, churnMs])

  // The vanishing button removes itself shortly after it appears, so a click
  // computed from an earlier snapshot of the page lands on nothing.
  useEffect(() => {
    if (!vanishing) return
    const timer = setTimeout(() => setVanishing(false), 600)
    return () => clearTimeout(timer)
  }, [vanishing])

  const record = (name: string) => setClicked((current) => [...current, name])

  return (
    <ChallengePage id="detached-elements">
      <h2 className="stage__heading">A list that rebuilds itself</h2>

      <div className="flex flex-wrap items-center gap-3">
        <button
          className="primary"
          data-testid="toggle-churn"
          data-churning={churning}
          onClick={() => setChurning((on) => !on)}
        >
          {churning ? 'Stop rebuilding' : `Rebuild every ${churnMs} ms`}
        </button>
        <span className="text-sm text-muted">
          generation <b data-testid="generation">{generation}</b>
        </span>
      </div>

      <ul className="mt-4 list-none p-0">
        {ROWS.map((name, index) => (
          // The key changes every generation, so React discards the element
          // and builds a new one rather than updating the old one.
          <li
            key={`${name}-${generation}`}
            id={`row-${generation}-${index}`}
            data-testid="unstable-row"
            data-name={name}
            className="flex items-center gap-3 border-b border-line py-2 last:border-b-0"
          >
            <span className="w-24 font-mono text-xs text-muted" data-testid="row-dom-id">
              row-{generation}-{index}
            </span>
            <span className="flex-1">{name}</span>
            <button data-testid="row-action" onClick={() => record(name)}>
              Choose
            </button>
          </li>
        ))}
      </ul>

      <dl className="mt-4 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        <dt className="text-muted">Chosen</dt>
        <dd data-testid="chosen">{clicked.join(', ') || 'nothing yet'}</dd>
      </dl>

      <h2 className="stage__heading">A button that leaves</h2>

      <div className="flex flex-wrap items-center gap-3">
        <button data-testid="summon" onClick={() => setVanishing(true)}>
          Summon it
        </button>
        {vanishing && (
          <button
            className="primary"
            data-testid="vanishing"
            onClick={() => setVanishClicks((n) => n + 1)}
          >
            Click me before I go
          </button>
        )}
        <span className="text-sm text-muted">
          caught <b data-testid="vanish-clicks">{vanishClicks}</b>
        </span>
      </div>

      <UnmountingForm />
    </ChallengePage>
  )
}

// A form that unmounts while it is being filled in, which is what a route
// change or a websocket push does to a page nobody was expecting it on.
function UnmountingForm() {
  const [alive, setAlive] = useState(true)
  const [typed, setTyped] = useState('')
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => () => clearTimeout(timer.current ?? undefined), [])

  return (
    <>
      <h2 className="stage__heading">A form that leaves mid-sentence</h2>
      <div className="flex flex-wrap items-center gap-3">
        <button
          data-testid="arm-unmount"
          onClick={() => {
            setAlive(true)
            setTyped('')
            timer.current = setTimeout(() => setAlive(false), 800)
          }}
        >
          Arm it (unmounts in 800 ms)
        </button>
        {alive ? (
          <input
            data-testid="doomed-field"
            className="rounded-md border border-line bg-sunken px-2 py-1"
            placeholder="start typing"
            value={typed}
            onChange={(event) => setTyped(event.target.value)}
          />
        ) : (
          <span className="text-sm text-muted" data-testid="form-gone">
            the field unmounted, taking what was typed with it
          </span>
        )}
      </div>
    </>
  )
}
