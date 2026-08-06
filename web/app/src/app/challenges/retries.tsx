import { useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'
import { clampInt } from '../search'

interface Attempt {
  number: number
  status: number
}

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/retries',
  validateSearch: (search: Record<string, unknown>) => ({
    failFirst: clampInt(search.failFirst, 3, 0, 20),
  }),
  component: Retries,
})

const MAX_ATTEMPTS = 6

function Retries() {
  const { failFirst } = route.useSearch()
  const [attempts, setAttempts] = useState<Attempt[]>([])
  const [outcome, setOutcome] = useState('not started')
  const [payload, setPayload] = useState<string | null>(null)

  async function ask() {
    const res = await fetch(`/api/app/retries/data?failFirst=${failFirst}`)
    return { res, body: (await res.json()) as { message?: string } }
  }

  async function run(retrying: boolean) {
    setAttempts([])
    setPayload(null)
    setOutcome('in flight')

    for (let number = 1; number <= (retrying ? MAX_ATTEMPTS : 1); number++) {
      const { res, body } = await ask()
      setAttempts((current) => [...current, { number, status: res.status }])

      if (res.ok) {
        setPayload(body.message ?? '')
        setOutcome('succeeded')
        return
      }
      // Growing backoff, kept small so a suite is not paying for the lesson.
      if (retrying && number < MAX_ATTEMPTS) {
        await new Promise((resolve) => setTimeout(resolve, 50 * number))
      }
    }
    setOutcome('failed')
  }

  async function reset() {
    await fetch('/api/app/retries/reset', { method: 'POST' })
    setAttempts([])
    setPayload(null)
    setOutcome('not started')
  }

  return (
    <ChallengePage id="retries">
      <p className="stage__label">
        The endpoint refuses its first <b data-testid="fail-first">{failFirst}</b> calls in this
        session, then answers
      </p>

      <div className="flex flex-wrap gap-2">
        <button className="primary" data-testid="fetch-retrying" onClick={() => run(true)}>
          Ask, and keep asking
        </button>
        <button data-testid="fetch-once" onClick={() => run(false)}>
          Ask once
        </button>
        <button data-testid="reset-endpoint" onClick={reset}>
          Reset the endpoint
        </button>
      </div>

      <dl className="mt-5 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        <dt className="text-muted">Attempts</dt>
        <dd data-testid="attempt-count">{attempts.length}</dd>
        <dt className="text-muted">Outcome</dt>
        <dd data-testid="outcome">{outcome}</dd>
      </dl>

      {attempts.length > 0 && (
        <ul className="mt-4 list-none p-0 font-mono text-xs">
          {attempts.map((attempt) => (
            <li key={attempt.number} data-testid="attempt-row" data-status={attempt.status}>
              attempt {attempt.number} → {attempt.status}
            </li>
          ))}
        </ul>
      )}

      {payload && (
        <p className="mt-4" data-testid="payload">
          {payload}
        </p>
      )}
    </ChallengePage>
  )
}
