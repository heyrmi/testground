import { useRef, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/request-races',
  component: RequestRaces,
})

// The slow search is asked for first and answers second, which is the whole
// point: the older question wins the race to be rendered.
const SLOW = { q: 'slow', ms: 700 }
const FAST = { q: 'fast', ms: 100 }

const WATERFALL = ['first', 'second', 'third']
const STEP_MS = 250

function RequestRaces() {
  const [naive, setNaive] = useState('nothing yet')
  const [guarded, setGuarded] = useState('nothing yet')
  const [steps, setSteps] = useState<string[]>([])
  const [total, setTotal] = useState(0)
  const [running, setRunning] = useState(false)

  // Only the most recent request is allowed to write to the guarded panel.
  const latest = useRef(0)

  async function search({ q, ms }: { q: string; ms: number }) {
    const res = await fetch(`/api/app/races/search?q=${q}&ms=${ms}`)
    return (await res.json()) as { query: string }
  }

  function runRace() {
    setNaive('nothing yet')
    setGuarded('nothing yet')
    // Identical handlers but for one line: each records the ticket it was
    // issued and only writes to the guarded panel if it is still the newest.
    const slowTicket = ++latest.current
    void search(SLOW).then((body) => {
      setNaive(body.query)
      if (slowTicket === latest.current) setGuarded(body.query)
    })

    const fastTicket = ++latest.current
    void search(FAST).then((body) => {
      setNaive(body.query)
      if (fastTicket === latest.current) setGuarded(body.query)
    })
  }

  async function runWaterfall() {
    setSteps([])
    setRunning(true)
    const started = performance.now()

    // Each request only starts once the one before it has answered, so the
    // chain costs the sum of its parts rather than the slowest of them.
    for (const step of WATERFALL) {
      const res = await fetch(`/api/app/races/step?step=${step}&ms=${STEP_MS}`)
      const body = (await res.json()) as { step: string }
      setSteps((current) => [...current, body.step])
    }

    setTotal(Math.round(performance.now() - started))
    setRunning(false)
  }

  return (
    <ChallengePage id="request-races">
      <h2 className="stage__heading">Two searches, the older one answering last</h2>

      <button className="primary" data-testid="run-race" onClick={runRace}>
        Search “{SLOW.q}”, then “{FAST.q}”
      </button>

      <dl className="mt-4 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        <dt className="text-muted">Naive handler</dt>
        <dd data-testid="naive-result">{naive}</dd>
        <dt className="text-muted">Guarded handler</dt>
        <dd data-testid="guarded-result">{guarded}</dd>
      </dl>

      <p className="mt-3 text-sm text-muted">
        Both panels receive the same two responses in the same order. One of them checks
        whether the answer it is holding is still the answer to the current question.
      </p>

      <h2 className="stage__heading">Three requests, each waiting for the last</h2>

      <button data-testid="run-waterfall" onClick={runWaterfall} disabled={running}>
        Run the waterfall
      </button>

      <ol className="mt-4 list-none p-0 font-mono text-xs">
        {steps.map((step) => (
          <li key={step} data-testid="waterfall-step">
            {step}
          </li>
        ))}
      </ol>

      <dl className="mt-3 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        <dt className="text-muted">Total</dt>
        <dd>
          <span data-testid="waterfall-total">{total}</span> ms
        </dd>
      </dl>

      {steps.length === WATERFALL.length && !running && (
        <p className="mt-2 text-sm" data-testid="waterfall-done">
          Every step finished. Waiting for the first response would have read this page two
          requests too early.
        </p>
      )}
    </ChallengePage>
  )
}
