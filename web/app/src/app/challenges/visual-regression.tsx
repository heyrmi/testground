import { useEffect, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/visual-regression',
  validateSearch: (search: Record<string, unknown>) => ({
    // The one deliberate difference, so a comparison can be proved to detect
    // something rather than merely proved not to complain.
    diff: truthy(search.diff, false),
    freeze: truthy(search.freeze, true),
  }),
  component: VisualRegression,
})

// The router parses search values rather than handing back raw strings, so
// ?diff=1 arrives as the number 1 and a string comparison silently never
// matches. Coercing here keeps the URL forgiving either way.
function truthy(value: unknown, fallback: boolean): boolean {
  if (value === undefined || value === '') return fallback
  return value === true || value === 1 || value === '1' || value === 'true'
}

function VisualRegression() {
  const { diff, freeze } = route.useSearch()
  const [now, setNow] = useState('')

  // Dynamic content is what makes a screenshot differ from itself. It is here
  // on purpose, and marked so a comparison can be told to ignore it.
  useEffect(() => {
    const timer = setInterval(() => setNow(new Date().toISOString()), 100)
    return () => clearInterval(timer)
  }, [])

  return (
    <ChallengePage id="visual-regression">
      <div className="flex flex-wrap gap-3 text-sm">
        <span>
          diff <b data-testid="diff-state">{diff ? 'on' : 'off'}</b>
        </span>
        <span>
          animation <b data-testid="freeze-state">{freeze ? 'frozen' : 'running'}</b>
        </span>
      </div>

      <section
        data-testid="reference"
        className="mt-4 rounded-lg border border-line bg-sunken p-6"
        style={{
          // System fonts only, so nothing swaps in late and shifts the layout
          // after the first paint.
          fontFamily: 'ui-monospace, monospace',
          // A fixed size, so the capture does not depend on the window.
          width: 420,
        }}
      >
        <div className="flex items-center gap-4">
          <div
            data-testid="swatch"
            style={{
              // The only intentional difference between the two states, and it
              // is one pixel wide.
              width: diff ? 65 : 64,
              height: 64,
              background: '#b4541e',
              borderRadius: 8,
            }}
          />
          <div>
            <p className="m-0 text-sm font-semibold">Reference block</p>
            <p className="m-0 text-xs text-muted">Fixed size, fixed font, fixed colours.</p>
          </div>
        </div>

        <div className="mt-4 grid grid-cols-4 gap-1">
          {Array.from({ length: 8 }, (_, i) => (
            <div key={i} style={{ height: 24, background: i % 2 ? '#2f7d4f' : '#2b6cb0' }} />
          ))}
        </div>

        {/* Masked rather than removed: a comparison that cannot ignore regions
            has to be told about this, and pretending it is not there would
            teach the wrong habit. */}
        <p className="mt-4 font-mono text-xs" data-vr-mask="true" data-testid="volatile">
          {now || 'starting'}
        </p>

        <div
          data-testid="spinner"
          className="mt-4 h-6 w-6 rounded-full border-2 border-line border-t-accent"
          style={
            freeze
              ? { transform: 'rotate(45deg)' }
              : { animation: 'vr-spin 1s linear infinite' }
          }
        />
        <style>{`@keyframes vr-spin { to { transform: rotate(360deg) } }`}</style>
      </section>

      <p className="mt-4 text-sm text-muted">
        Add <code>?diff=1</code> for the one-pixel change, and <code>?freeze=0</code> to let the
        spinner run. A comparison that passes with the diff on is not comparing anything.
      </p>
    </ChallengePage>
  )
}
