import { useEffect, useRef, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'
import { clampInt } from '../search'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/dom-scale',
  validateSearch: (search: Record<string, unknown>) => ({
    nodes: clampInt(search.nodes, 20_000, 0, 60_000),
    blockMs: clampInt(search.blockMs, 3000, 0, 20_000),
  }),
  component: DomScale,
})

function DomScale() {
  const { nodes, blockMs } = route.useSearch()
  const [built, setBuilt] = useState(0)
  const [listeners, setListeners] = useState(0)
  const [blocked, setBlocked] = useState(false)
  const [thrashing, setThrashing] = useState(false)
  const [retained, setRetained] = useState(0)
  const host = useRef<HTMLDivElement>(null)
  const leak = useRef<unknown[]>([])
  const thrashTimer = useRef<number | null>(null)

  // Built outside React on purpose: fifty thousand components would measure
  // the framework, and the point is to measure the page.
  function build() {
    const container = host.current
    if (!container) return

    container.replaceChildren()
    const fragment = document.createDocumentFragment()
    for (let i = 0; i < nodes; i++) {
      const cell = document.createElement('span')
      cell.className = 'scale-cell'
      cell.dataset.index = String(i)
      cell.textContent = String(i % 10)
      fragment.appendChild(cell)
    }
    container.appendChild(fragment)
    setBuilt(nodes)
  }

  function attachListeners() {
    const cells = host.current?.querySelectorAll<HTMLElement>('.scale-cell') ?? []
    let attached = 0
    for (const cell of cells) {
      if (attached >= 500) break
      cell.addEventListener('mousemove', () => {})
      attached += 1
    }
    setListeners(attached)
  }

  // Synchronous, so nothing else can run. A framework's own waiting stops too,
  // which is what makes this different from a slow network.
  function blockMainThread() {
    setBlocked(true)
    requestAnimationFrame(() => {
      const until = performance.now() + blockMs
      while (performance.now() < until) {
        /* deliberately busy */
      }
      setBlocked(false)
    })
  }

  // Reading a layout property after every write forces a synchronous reflow
  // per iteration, which is the classic accidental way to make a page crawl.
  useEffect(() => {
    if (!thrashing) return

    const tick = () => {
      const cells = host.current?.querySelectorAll<HTMLElement>('.scale-cell') ?? []
      let index = 0
      for (const cell of cells) {
        if (index++ > 400) break
        cell.style.paddingLeft = `${index % 3}px`
        void cell.offsetWidth
      }
      thrashTimer.current = requestAnimationFrame(tick)
    }
    thrashTimer.current = requestAnimationFrame(tick)

    return () => {
      if (thrashTimer.current !== null) cancelAnimationFrame(thrashTimer.current)
    }
  }, [thrashing])

  return (
    <ChallengePage id="dom-scale">
      <div className="flex flex-wrap gap-2">
        <button className="primary" data-testid="build-nodes" onClick={build}>
          Build {nodes.toLocaleString('en')} nodes
        </button>
        <button data-testid="attach-listeners" onClick={attachListeners}>
          Attach 500 listeners
        </button>
        <button data-testid="block-thread" onClick={blockMainThread}>
          Block the thread for {blockMs} ms
        </button>
        <button
          data-testid="toggle-thrash"
          data-thrashing={thrashing}
          onClick={() => setThrashing((on) => !on)}
        >
          {thrashing ? 'Stop thrashing layout' : 'Thrash layout'}
        </button>
        <button
          data-testid="leak"
          onClick={() => {
            // Never released, and nothing on screen says so.
            leak.current.push(new Array(50_000).fill('retained'))
            setRetained(leak.current.length)
          }}
        >
          Leak a little
        </button>
      </div>

      <dl className="mt-5 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        <dt className="text-muted">Nodes built</dt>
        <dd data-testid="node-count">{built}</dd>
        <dt className="text-muted">Listeners attached</dt>
        <dd data-testid="listener-count">{listeners}</dd>
        <dt className="text-muted">Main thread</dt>
        <dd data-testid="thread-state">{blocked ? 'blocked' : 'free'}</dd>
        <dt className="text-muted">Retained blocks</dt>
        <dd data-testid="retained-count">{retained}</dd>
      </dl>

      <p className="mt-4 text-sm text-muted">
        Add <code>?nodes=</code> and <code>?blockMs=</code> to change the scale. Everything here
        is a number a test can read, because none of it changes what the page says.
      </p>

      <div
        ref={host}
        data-testid="node-host"
        className="mt-4 max-h-64 overflow-auto break-all rounded-lg border border-line bg-sunken p-2 font-mono text-[10px] leading-3"
      />
    </ChallengePage>
  )
}
