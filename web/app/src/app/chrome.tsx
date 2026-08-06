import { createContext, useContext, type ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import type { Challenge, Manifest } from './manifest'

const ManifestContext = createContext<Manifest | null>(null)

export function ManifestProvider({ value, children }: { value: Manifest; children: ReactNode }) {
  return <ManifestContext value={value}>{children}</ManifestContext>
}

export function useManifest(): Manifest {
  const manifest = useContext(ManifestContext)
  if (!manifest) throw new Error('useManifest called outside ManifestProvider')
  return manifest
}

export function useChallenge(id: string): Challenge | undefined {
  return useManifest().challenges.find((c) => c.id === id)
}

export function Masthead() {
  const { version, seed, session } = useManifest()
  return (
    <header className="masthead">
      <div className="masthead__inner">
        <a className="masthead__name" href="/">
          test<span>ground</span>
        </a>
        <div className="masthead__meta">
          <span>
            version <b data-testid="meta-version">{version}</b>
          </span>
          <span>
            seed <b data-testid="meta-seed">{seed}</b>
          </span>
          <span>
            session <b data-testid="meta-session">{session}</b>
          </span>
        </div>
      </div>
    </header>
  )
}

export function Colophon() {
  return (
    <footer className="colophon">
      <span>Deterministic by default. Same seed, same page.</span>
      <a href="/api/challenges">Challenge manifest</a>
      <a href="/api/health">Health</a>
      <a href="https://github.com/heyrmi/testground">Source</a>
    </footer>
  )
}

export function TierBadge({ tier }: { tier: string }) {
  return (
    <span className={`badge badge--${tier.toLowerCase()}`} data-testid="challenge-tier">
      {tier}
    </span>
  )
}

/**
 * ChallengePage renders the same chrome the server-rendered zones produce from
 * templates/partials/challenge.html, down to the test ids, so a test that
 * reads a description or opens a hint works identically in every zone.
 */
export function ChallengePage({ id, children }: { id: string; children: ReactNode }) {
  const challenge = useChallenge(id)

  if (!challenge) {
    return (
      <p className="empty" data-testid="unknown-challenge">
        This build does not serve a challenge called <code>{id}</code>.
      </p>
    )
  }

  return (
    <>
      <nav className="crumb">
        <Link to="/" data-testid="back-to-zone">
          ← Modern SPA challenges
        </Link>
      </nav>

      <header className="challenge__head">
        <div className="badges">
          <TierBadge tier={challenge.tier} />
          <span className="badge" data-testid="challenge-zone">
            {challenge.zone}
          </span>
          <span className="tag">{challenge.category}</span>
        </div>
        <h1 data-testid="challenge-title">{challenge.title}</h1>
      </header>

      <section className="panel" data-testid="challenge-panel">
        <h2>What this page does</h2>
        <p>{challenge.summary}</p>

        <h2>Why it breaks naive automation</h2>
        <p>{challenge.whyHard}</p>

        {challenge.selectors.length > 0 && (
          <>
            <h2>Elements worth locating</h2>
            <table className="selectors">
              <thead>
                <tr>
                  <th>data-testid</th>
                  <th>Role</th>
                  <th>What it is</th>
                </tr>
              </thead>
              <tbody>
                {challenge.selectors.map((selector) => (
                  <tr key={selector.testId}>
                    <td>{selector.testId}</td>
                    <td>{selector.role ?? '—'}</td>
                    <td>{selector.note}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </>
        )}

        <details className="hint" data-testid="challenge-hint">
          <summary>Show hint</summary>
          <p>{challenge.hint}</p>
        </details>
      </section>

      <section className="stage" data-testid="stage">
        {children}
      </section>
    </>
  )
}
