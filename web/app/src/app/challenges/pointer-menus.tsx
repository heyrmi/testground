import { useRef, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/pointer-menus',
  component: PointerMenus,
})

const LONG_PRESS_MS = 500

function PointerMenus() {
  const [chosen, setChosen] = useState('nothing yet')
  const [menuOpen, setMenuOpen] = useState(false)
  const [submenuOpen, setSubmenuOpen] = useState(false)
  const [contextAt, setContextAt] = useState<{ x: number; y: number } | null>(null)
  const [doubleClicks, setDoubleClicks] = useState(0)
  const [singleClicks, setSingleClicks] = useState(0)
  const [longPresses, setLongPresses] = useState(0)
  const pressTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  function startPress() {
    pressTimer.current = setTimeout(() => {
      setLongPresses((n) => n + 1)
      pressTimer.current = null
    }, LONG_PRESS_MS)
  }

  function endPress() {
    if (pressTimer.current) {
      clearTimeout(pressTimer.current)
      pressTimer.current = null
    }
  }

  return (
    <ChallengePage id="pointer-menus">
      <dl className="mb-6 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        <dt className="text-muted">Last choice</dt>
        <dd data-testid="menu-choice">{chosen}</dd>
        <dt className="text-muted">Single / double</dt>
        <dd>
          <span data-testid="single-clicks">{singleClicks}</span> /{' '}
          <span data-testid="double-clicks">{doubleClicks}</span>
        </dd>
        <dt className="text-muted">Long presses</dt>
        <dd data-testid="long-presses">{longPresses}</dd>
      </dl>

      <section>
        <h2 className="stage__heading">A menu that exists only while hovered</h2>
        <div
          className="relative inline-block"
          data-testid="hover-root"
          onMouseEnter={() => setMenuOpen(true)}
          onMouseLeave={() => {
            setMenuOpen(false)
            setSubmenuOpen(false)
          }}
        >
          <button data-testid="hover-trigger">Hover me</button>

          {/* In the DOM only while the pointer is inside the wrapper, so the
              journey from the trigger into the menu must not leave the group. */}
          {menuOpen && (
          <div className="absolute left-0 top-full z-10 pt-1">
            <div className="rounded-lg border border-line bg-surface p-1 shadow-lg" data-testid="hover-menu">
              <button className="block w-full !border-0 text-left" data-testid="menu-open" onClick={() => setChosen('open')}>
                Open
              </button>
              <div
                className="relative"
                onMouseEnter={() => setSubmenuOpen(true)}
                onMouseLeave={() => setSubmenuOpen(false)}
              >
                <button className="block w-full !border-0 text-left" data-testid="menu-more">
                  More ▸
                </button>
                {submenuOpen && (
                  <div
                    className="absolute left-full top-0 rounded-lg border border-line bg-surface p-1 shadow-lg"
                    data-testid="submenu"
                  >
                    <button
                      className="block w-full !border-0 text-left"
                      data-testid="menu-archive"
                      onClick={() => setChosen('archive')}
                    >
                      Archive
                    </button>
                  </div>
                )}
              </div>
            </div>
          </div>
          )}
        </div>
        <p className="mt-2 text-sm text-muted">
          The menu is in the DOM only while the pointer is inside the group. Moving to it in a
          straight line works; moving away at all closes it.
        </p>
      </section>

      <section className="mt-8">
        <h2 className="stage__heading">Right-click, double-click, long press</h2>
        <div
          data-testid="gesture-target"
          className="flex h-28 w-full max-w-md items-center justify-center rounded-lg border border-dashed border-line text-sm text-muted select-none"
          onContextMenu={(event) => {
            event.preventDefault()
            const box = event.currentTarget.getBoundingClientRect()
            setContextAt({ x: event.clientX - box.left, y: event.clientY - box.top })
          }}
          onClick={() => setSingleClicks((n) => n + 1)}
          onDoubleClick={() => setDoubleClicks((n) => n + 1)}
          onPointerDown={startPress}
          onPointerUp={endPress}
          onPointerLeave={endPress}
        >
          right-click, double-click, or hold for {LONG_PRESS_MS} ms
        </div>

        {contextAt && (
          <div
            data-testid="context-menu"
            className="mt-2 inline-block rounded-lg border border-line bg-surface p-1 shadow-lg"
          >
            <button
              className="block w-full !border-0 text-left"
              data-testid="context-rename"
              onClick={() => {
                setChosen('rename')
                setContextAt(null)
              }}
            >
              Rename
            </button>
          </div>
        )}
        <p className="mt-2 text-sm text-muted">
          A double-click fires two single clicks as well, so the counters never agree — and the
          browser's own context menu is suppressed in favour of that one.
        </p>
      </section>
    </ChallengePage>
  )
}
