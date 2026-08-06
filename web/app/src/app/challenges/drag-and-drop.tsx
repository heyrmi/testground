import { useRef, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/drag-and-drop',
  component: DragAndDrop,
})

const PARCELS = ['letter', 'parcel', 'crate']
const ORDER = ['one', 'two', 'three', 'four']

function DragAndDrop() {
  const [delivered, setDelivered] = useState<string[]>([])
  const [order, setOrder] = useState(ORDER)
  const [held, setHeld] = useState<string | null>(null)
  const [handleAt, setHandleAt] = useState(0)
  const rail = useRef<HTMLDivElement>(null)

  // A pointer drag with no HTML5 drag-and-drop anywhere near it. It needs a
  // press, at least one move, and a release -- a jump straight to the target
  // moves nothing, because there is no drop target to jump to.
  function onPointerMove(event: React.PointerEvent) {
    if (event.buttons !== 1) return
    const box = rail.current?.getBoundingClientRect()
    if (!box) return
    const ratio = Math.min(1, Math.max(0, (event.clientX - box.left) / box.width))
    setHandleAt(Math.round(ratio * 100))
  }

  return (
    <ChallengePage id="drag-and-drop">
      <section>
        <h2 className="stage__heading">HTML5 drag and drop</h2>
        <div className="flex flex-wrap gap-6">
          <div className="flex flex-col gap-2" data-testid="parcel-source">
            {PARCELS.filter((parcel) => !delivered.includes(parcel)).map((parcel) => (
              <div
                key={parcel}
                draggable
                data-testid="parcel"
                data-name={parcel}
                onDragStart={(event) => event.dataTransfer.setData('text/plain', parcel)}
                className="cursor-grab rounded-lg border border-line bg-sunken px-3 py-2 text-sm"
              >
                {parcel}
              </div>
            ))}
          </div>

          <div
            data-testid="dropzone"
            onDragOver={(event) => event.preventDefault()}
            onDrop={(event) => {
              event.preventDefault()
              const parcel = event.dataTransfer.getData('text/plain')
              if (parcel) setDelivered((current) => [...current, parcel])
            }}
            className="min-h-24 min-w-56 rounded-lg border border-dashed border-line p-3 text-sm text-muted"
          >
            Drop here — <b data-testid="delivered-count">{delivered.length}</b> delivered
            <div className="mt-2 flex flex-wrap gap-2">
              {delivered.map((parcel) => (
                <span key={parcel} data-testid="delivered" data-name={parcel} className="tag">
                  {parcel}
                </span>
              ))}
            </div>
          </div>
        </div>
        <p className="mt-2 text-sm text-muted">
          These use the native drag events. Synthesising them is not the same as moving a mouse,
          which is why so many drivers cannot do it at all.
        </p>
      </section>

      <section className="mt-8">
        <h2 className="stage__heading">A list reordered by dragging</h2>
        <ul className="m-0 flex list-none flex-col gap-1 p-0" data-testid="sortable">
          {order.map((item) => (
            <li
              key={item}
              draggable
              data-testid="sortable-item"
              data-name={item}
              onDragStart={() => setHeld(item)}
              onDragOver={(event) => event.preventDefault()}
              onDrop={() => {
                if (!held || held === item) return
                setOrder((current) => {
                  const next = current.filter((entry) => entry !== held)
                  next.splice(next.indexOf(item), 0, held)
                  return next
                })
                setHeld(null)
              }}
              className="cursor-grab rounded-md border border-line bg-sunken px-3 py-1.5 text-sm"
            >
              {item}
            </li>
          ))}
        </ul>
        <p className="mt-2 text-sm text-muted">
          Order: <b data-testid="sortable-order">{order.join(', ')}</b>
        </p>
      </section>

      <section className="mt-8">
        <h2 className="stage__heading">A handle that only a pointer sequence moves</h2>
        <div
          ref={rail}
          data-testid="rail"
          className="relative h-2 w-80 rounded-full bg-sunken"
          onPointerDown={(event) => {
            event.currentTarget.setPointerCapture(event.pointerId)
            onPointerMove(event)
          }}
          onPointerMove={onPointerMove}
        >
          <div
            data-testid="handle"
            className="absolute top-1/2 h-5 w-5 -translate-x-1/2 -translate-y-1/2 rounded-full border border-line bg-accent"
            style={{ left: `${handleAt}%` }}
          />
        </div>
        <p className="mt-3 text-sm text-muted">
          Position: <b data-testid="handle-position">{handleAt}</b>. No drag events here at all,
          only pointer ones.
        </p>
      </section>
    </ChallengePage>
  )
}
