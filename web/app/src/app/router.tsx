import { createRouter } from '@tanstack/react-router'
import { indexRoute, rootRoute } from './root'
import { route as delayedElementRoute } from './challenges/delayed-element'
import { route as toastRoute } from './challenges/toast'
import { route as optimisticRevertRoute } from './challenges/optimistic-revert'
import { route as dataTableRoute } from './challenges/data-table'
import { route as domScaleRoute } from './challenges/dom-scale'
import { route as hostileRoute } from './challenges/hostile-locators'
import { route as dragAndDropRoute } from './challenges/drag-and-drop'
import { route as pointerMenusRoute } from './challenges/pointer-menus'
import { route as detachedRoute } from './challenges/detached-elements'
import { route as fakeControlsRoute } from './challenges/fake-controls'
import { route as modalPortalRoute } from './challenges/modal-portal'
import { route as otpRoute } from './challenges/otp-input'
import { route as racesRoute } from './challenges/request-races'
import { route as tokenRefreshRoute } from './challenges/token-refresh'
import { route as retriesRoute } from './challenges/retries'
import { route as virtualListRoute } from './challenges/virtual-list'

const routeTree = rootRoute.addChildren([
  indexRoute,
  delayedElementRoute,
  toastRoute,
  virtualListRoute,
  optimisticRevertRoute,
  retriesRoute,
  racesRoute,
  otpRoute,
  fakeControlsRoute,
  detachedRoute,
  modalPortalRoute,
  dataTableRoute,
  dragAndDropRoute,
  pointerMenusRoute,
  tokenRefreshRoute,
  hostileRoute,
  domScaleRoute,
])

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
