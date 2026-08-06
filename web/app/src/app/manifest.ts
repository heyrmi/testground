export type Tier = 'T1' | 'T2' | 'T3' | 'T4'

export interface Selector {
  testId: string
  role?: string
  note: string
}

export interface Challenge {
  id: string
  title: string
  url: string
  zone: string
  tier: Tier
  category: string
  summary: string
  whyHard: string
  hint: string
  tags: string[]
  concepts: string[]
  selectors: Selector[]
  stability: 'stable' | 'experimental'
}

export interface Manifest {
  version: string
  session: string
  seed: number
  count: number
  challenges: Challenge[]
}

// The manifest is the single source of truth for every page's description and
// hint, so the SPA renders the same text the server-rendered zones do. One
// request per page load is enough; it cannot change while the tab is open.
let pending: Promise<Manifest> | undefined

export function loadManifest(): Promise<Manifest> {
  pending ??= fetch('/api/challenges', { headers: { Accept: 'application/json' } }).then((res) => {
    if (!res.ok) throw new Error(`manifest request failed with ${res.status}`)
    return res.json() as Promise<Manifest>
  })
  return pending
}
