import {
  Link,
  Outlet,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router'
import { Colophon, Masthead, useManifest } from './chrome'

const rootRoute = createRootRoute({
  component: () => (
    <>
      <Masthead />
      <main className="shell">
        <Outlet />
        <Colophon />
      </main>
    </>
  ),
  notFoundComponent: () => (
    <section className="intro">
      <h1 data-testid="not-found-heading">404 — no such challenge</h1>
      <p>
        <Link to="/">Back to the Modern SPA challenges</Link>
      </p>
    </section>
  ),
})

function ZoneIndex() {
  const challenges = useManifest().challenges.filter((c) => c.zone === 'app')

  return (
    <>
      <section className="intro">
        <h1>Modern SPA</h1>
        <p>
          React 19 with client-side routing. These pages break the assumptions that
          hold on a server-rendered site: elements arrive late, detach mid-interaction,
          never enter the DOM at all, or show a value the server has not agreed to yet.
        </p>
      </section>

      {challenges.length === 0 ? (
        <p className="empty" data-testid="no-challenges">
          This build ships no challenges in this zone.
        </p>
      ) : (
        <div className="cards">
          {challenges.map((challenge) => (
            <Link
              key={challenge.id}
              to={challenge.url.replace(/^\/app/, '') || '/'}
              className="card"
              data-testid={`challenge-card-${challenge.id}`}
            >
              <div className="badges">
                <span className={`badge badge--${challenge.tier.toLowerCase()}`}>
                  {challenge.tier}
                </span>
                <span className="tag">{challenge.category}</span>
              </div>
              <div className="card__title">{challenge.title}</div>
              <p className="card__summary">{challenge.summary}</p>
              <div className="card__url">{challenge.url}</div>
            </Link>
          ))}
        </div>
      )}
    </>
  )
}

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: ZoneIndex,
})

export const routeTree = rootRoute.addChildren([indexRoute])

export const router = createRouter({
  routeTree,
  // The Go server mounts this bundle under /app and serves the shell for every
  // path below it, so client routing owns everything after the prefix.
  basepath: '/app',
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
