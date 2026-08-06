import { createRoot } from 'react-dom/client'
import { RouterProvider } from '@tanstack/react-router'
import { ManifestProvider } from './chrome'
import { loadManifest } from './manifest'
import { router } from './router'
import './styles.css'

// The manifest resolves before the first render so every challenge page mounts
// once, at its final size. Suspending inside the page would shift the layout
// under the timing challenges and make them measure the chrome instead.
const manifest = await loadManifest()

const container = document.getElementById('root')
if (!container) throw new Error('missing #root')

// Deliberately no StrictMode. Its double-invoked effects would fire every
// challenge's timers twice, which is exactly the kind of phantom behaviour
// these pages exist to teach people to recognise.
createRoot(container).render(
  <ManifestProvider value={manifest}>
    <RouterProvider router={router} />
  </ManifestProvider>,
)
