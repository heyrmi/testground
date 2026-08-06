import { useEffect, useRef, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { useVirtualizer } from '@tanstack/react-virtual'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'
import { clampInt } from '../search'

// Fixed rather than measured, so the offset of row N is exactly N * ROW_HEIGHT
// and a test can jump to a row instead of scrolling towards it.
const ROW_HEIGHT = 40

interface Row {
  index: number
  name: string
  email: string
  status: string
  amount: string
}

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/virtual-list',
  validateSearch: (search: Record<string, unknown>) => ({
    count: clampInt(search.count, 10_000, 0, 100_000),
  }),
  component: VirtualList,
})

function VirtualList() {
  const { count } = route.useSearch()
  const [rows, setRows] = useState<Row[]>([])
  const [loaded, setLoaded] = useState(false)
  const viewport = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let current = true
    setLoaded(false)
    fetch(`/api/app/virtual-list/rows?count=${count}`)
      .then((res) => res.json() as Promise<{ rows: Row[] }>)
      .then((body) => {
        if (!current) return
        setRows(body.rows)
        setLoaded(true)
      })
    return () => {
      current = false
    }
  }, [count])

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => viewport.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 4,
  })
  const windowed = virtualizer.getVirtualItems()

  return (
    <ChallengePage id="virtual-list">
      <dl className="mb-4 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        <dt className="text-muted">Rows in the data set</dt>
        <dd data-testid="row-total">{rows.length}</dd>
        <dt className="text-muted">Rows in the DOM</dt>
        <dd data-testid="row-rendered">{windowed.length}</dd>
        <dt className="text-muted">Row height</dt>
        <dd data-testid="row-height">{ROW_HEIGHT}</dd>
      </dl>

      {!loaded && <p data-testid="rows-loading">Fetching rows…</p>}

      <div
        ref={viewport}
        data-testid="viewport"
        className="h-96 overflow-auto rounded-lg border border-line bg-sunken"
      >
        <div className="relative w-full" style={{ height: virtualizer.getTotalSize() }}>
          {windowed.map((item) => {
            const row = rows[item.index]
            if (!row) return null
            return (
              <div
                key={item.key}
                data-testid="row"
                data-index={row.index}
                role="row"
                className="absolute left-0 top-0 flex w-full items-center gap-4 border-b border-line px-4 font-mono text-xs"
                style={{ height: item.size, transform: `translateY(${item.start}px)` }}
              >
                <span className="w-14 text-muted" data-testid="row-index">
                  {String(row.index).padStart(5, '0')}
                </span>
                <span className="w-44 truncate" data-testid="row-name">
                  {row.name}
                </span>
                <span className="flex-1 truncate text-muted" data-testid="row-email">
                  {row.email}
                </span>
                <span className="w-24" data-testid="row-status">
                  {row.status}
                </span>
                <span className="w-20 text-right" data-testid="row-amount">
                  {row.amount}
                </span>
              </div>
            )
          })}
        </div>
      </div>
    </ChallengePage>
  )
}
