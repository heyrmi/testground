import { useRef, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/fake-controls',
  component: FakeControls,
})

const STARS = 5
const SLIDER_WIDTH = 320

function FakeControls() {
  const [enabled, setEnabled] = useState(false)
  const [rating, setRating] = useState(0)
  const [hovered, setHovered] = useState(0)
  const [amount, setAmount] = useState(20)
  const track = useRef<HTMLDivElement>(null)

  // The slider has no input behind it, so there is nothing to type into and
  // nothing to press an arrow key against. Only a drag moves it.
  function moveTo(clientX: number) {
    const box = track.current?.getBoundingClientRect()
    if (!box) return
    const ratio = Math.min(1, Math.max(0, (clientX - box.left) / box.width))
    setAmount(Math.round(ratio * 100))
  }

  return (
    <ChallengePage id="fake-controls">
      <div className="flex flex-col gap-8">
        <section>
          <h2 className="stage__heading">A switch that is not a checkbox</h2>
          <div
            role="switch"
            tabIndex={0}
            aria-checked={enabled}
            data-testid="toggle"
            data-state={enabled ? 'on' : 'off'}
            onClick={() => setEnabled((on) => !on)}
            onKeyDown={(event) => {
              if (event.key === ' ' || event.key === 'Enter') {
                event.preventDefault()
                setEnabled((on) => !on)
              }
            }}
            className={`inline-flex h-7 w-12 cursor-pointer items-center rounded-full border border-line p-0.5 transition-colors ${
              enabled ? 'bg-accent' : 'bg-sunken'
            }`}
          >
            <span
              className={`h-5 w-5 rounded-full bg-surface transition-transform ${
                enabled ? 'translate-x-5' : ''
              }`}
            />
          </div>
          <p className="mt-2 text-sm text-muted">
            State: <b data-testid="toggle-state">{enabled ? 'on' : 'off'}</b>. There is no input
            element here, so nothing to check and nothing to read a checked property from.
          </p>
        </section>

        <section>
          <h2 className="stage__heading">A rating that changes under the pointer</h2>
          <div
            data-testid="rating"
            role="radiogroup"
            aria-label="Rating"
            onMouseLeave={() => setHovered(0)}
            className="flex gap-1 text-2xl"
          >
            {Array.from({ length: STARS }, (_, i) => i + 1).map((star) => (
              <span
                key={star}
                role="radio"
                aria-checked={rating === star}
                aria-label={`${star} stars`}
                data-testid={`star-${star}`}
                onMouseEnter={() => setHovered(star)}
                onClick={() => setRating(star)}
                className="cursor-pointer select-none"
              >
                {star <= (hovered || rating) ? '★' : '☆'}
              </span>
            ))}
          </div>
          <p className="mt-2 text-sm text-muted">
            Chosen: <b data-testid="rating-value">{rating}</b>, showing{' '}
            <b data-testid="rating-shown">{hovered || rating}</b>. Reading the stars while the
            pointer is over them measures the hover, not the choice.
          </p>
        </section>

        <section>
          <h2 className="stage__heading">A slider that only a drag can move</h2>
          <div
            ref={track}
            data-testid="slider-track"
            className="relative h-2 rounded-full bg-sunken"
            style={{ width: SLIDER_WIDTH }}
            onPointerDown={(event) => {
              event.currentTarget.setPointerCapture(event.pointerId)
              moveTo(event.clientX)
            }}
            onPointerMove={(event) => {
              if (event.buttons === 1) moveTo(event.clientX)
            }}
          >
            <div
              data-testid="slider-thumb"
              className="absolute top-1/2 h-5 w-5 -translate-x-1/2 -translate-y-1/2 rounded-full border border-line bg-accent"
              style={{ left: `${amount}%` }}
            />
          </div>
          <p className="mt-3 text-sm text-muted">
            Value: <b data-testid="slider-value">{amount}</b>. No input, no arrow keys, no value to
            set — the position is the state.
          </p>
        </section>
      </div>
    </ChallengePage>
  )
}
