import { useEffect, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'
import { clampInt } from '../search'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/delayed-element',
  validateSearch: (search: Record<string, unknown>) => ({
    delayMs: clampInt(search.delayMs, 3000, 0, 60_000),
  }),
  component: DelayedElement,
})

function DelayedElement() {
  const { delayMs } = route.useSearch()
  const [attempt, setAttempt] = useState(0)
  const [arrived, setArrived] = useState(false)

  useEffect(() => {
    setArrived(false)
    const timer = setTimeout(() => setArrived(true), delayMs)
    return () => clearTimeout(timer)
  }, [delayMs, attempt])

  return (
    <ChallengePage id="delayed-element">
      <p className="stage__label">
        Waiting <b data-testid="delay-ms">{delayMs}</b> ms for the element to arrive
      </p>

      <div className="min-h-24 flex flex-col justify-center">
        {arrived ? (
          <p className="text-xl font-medium" data-testid="delayed-message">
            The element you were waiting for.
          </p>
        ) : (
          <p className="text-muted" data-testid="delay-pending">
            Nothing here yet. The message element is not in the DOM.
          </p>
        )}
      </div>

      <button data-testid="restart" onClick={() => setAttempt((n) => n + 1)}>
        Run the wait again
      </button>
    </ChallengePage>
  )
}
