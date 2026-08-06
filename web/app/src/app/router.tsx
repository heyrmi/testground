import { createRouter } from '@tanstack/react-router'
import { indexRoute, rootRoute } from './root'
import { route as delayedElementRoute } from './challenges/delayed-element'

const routeTree = rootRoute.addChildren([indexRoute, delayedElementRoute])

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
