import { useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/hostile-locators',
  component: HostileLocators,
})

// A CSS-in-JS style hash. Real ones are content-derived and change whenever
// the styles do, which is to say on a schedule nobody testing controls.
function hashed(build: number, salt: string): string {
  let hash = build * 2654435761
  for (const character of salt) hash = (hash * 31 + character.charCodeAt(0)) >>> 0
  return `css-${hash.toString(36).slice(0, 7)}`
}

const ZERO_WIDTH = '​'

function HostileLocators() {
  const [build, setBuild] = useState(1)
  const [chosen, setChosen] = useState('nothing yet')

  const cls = (salt: string) => hashed(build, salt)

  return (
    <ChallengePage id="hostile-locators">
      <div className="flex flex-wrap items-center gap-3">
        <button className="primary" data-testid="rebuild" onClick={() => setBuild((n) => n + 1)}>
          Ship a new build
        </button>
        <span className="text-sm text-muted">
          build <b data-testid="build-number">{build}</b>, class names{' '}
          <code data-testid="sample-class">{cls('save')}</code>
        </span>
      </div>

      <dl className="mt-4 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        <dt className="text-muted">Last chosen</dt>
        <dd data-testid="chosen">{chosen}</dd>
      </dl>

      <section className="mt-6">
        <h2 className="stage__heading">Class names that change every build</h2>
        <div className="flex gap-2">
          <button className={cls('save')} onClick={() => setChosen('save')}>
            Save
          </button>
          <button className={cls('discard')} onClick={() => setChosen('discard')}>
            Discard
          </button>
        </div>
        <p className="mt-2 text-sm text-muted">
          A selector written against one of these is correct until the next deploy, and the
          failure arrives with no code change anyone will connect to it.
        </p>
      </section>

      <section className="mt-6">
        <h2 className="stage__heading">Two elements, one id</h2>
        {/* Invalid HTML that browsers accept silently. getElementById returns
            the first; a CSS id selector matches both. */}
        <div>
          <button id="duplicate" onClick={() => setChosen('first-duplicate')}>
            First with id=duplicate
          </button>{' '}
          <button id="duplicate" onClick={() => setChosen('second-duplicate')}>
            Second with the same id
          </button>
        </div>
        <p className="mt-2 text-sm text-muted">
          Both carry <code>id="duplicate"</code>. Whichever one your tool picks, it picked.
        </p>
      </section>

      <section className="mt-6">
        <h2 className="stage__heading">Twelve levels of div, no semantics anywhere</h2>
        <DivSoup depth={12} onReach={() => setChosen('div-soup')} />
        <p className="mt-2 text-sm text-muted">
          No role, no label, no test id, no heading. There is nothing here to locate by except
          structure and the word on the element.
        </p>
      </section>

      <section className="mt-6">
        <h2 className="stage__heading">Text that is not what it looks like</h2>
        <ul className="m-0 list-none p-0 text-sm">
          <li>
            Split across spans:{' '}
            <span data-testid="split-text">
              <span>Order</span> <span>number</span> <span>4417</span>
            </span>
          </li>
          <li>
            With zero-width characters:{' '}
            <span data-testid="zero-width">{`Total${ZERO_WIDTH}: ${ZERO_WIDTH}42`}</span>
          </li>
          <li>
            Truncated by CSS:{' '}
            <span
              data-testid="truncated"
              className="inline-block max-w-40 truncate align-bottom"
              title="This sentence is much longer than the box that is drawing it."
            >
              This sentence is much longer than the box that is drawing it.
            </span>
          </li>
        </ul>
        <p className="mt-2 text-sm text-muted">
          The first reads as one sentence and is three nodes. The second has invisible
          characters between the words. The third shows an ellipsis while the DOM holds the
          whole string, so what a user can read and what a test can read are different.
        </p>
      </section>

      <section className="mt-6">
        <h2 className="stage__heading">Identical twins, told apart only by position</h2>
        <div className="flex gap-2">
          {['left', 'right'].map((side) => (
            <button key={side} className={cls('twin')} onClick={() => setChosen(`twin-${side}`)}>
              Continue
            </button>
          ))}
        </div>
        <p className="mt-2 text-sm text-muted">
          Same text, same class, same everything. Only their order distinguishes them.
        </p>
      </section>
    </ChallengePage>
  )
}

function DivSoup({ depth, onReach }: { depth: number; onReach: () => void }) {
  if (depth === 0) {
    return (
      <div onClick={onReach} style={{ cursor: 'pointer', display: 'inline-block' }}>
        Approve
      </div>
    )
  }
  return (
    <div>
      <DivSoup depth={depth - 1} onReach={onReach} />
    </div>
  )
}
