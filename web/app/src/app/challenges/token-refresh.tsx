import { useRef, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/token-refresh',
  component: TokenRefresh,
})

interface Tokens {
  access: string
  refresh: string
}

function TokenRefresh() {
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [tokenState, setTokenState] = useState('none')
  const [status, setStatus] = useState('—')
  const [reason, setReason] = useState('—')
  const [refreshes, setRefreshes] = useState(0)
  const [identity, setIdentity] = useState<string | null>(null)
  const tokens = useRef<Tokens | null>(null)

  async function signIn() {
    const res = await fetch('/api/app/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: 'user', password: 'user123' }),
    })
    const body = (await res.json()) as Tokens
    tokens.current = body
    setTokenState('valid')
    setStatus(String(res.status))
    setReason('—')
    setIdentity(null)
    setRefreshes(0)
  }

  async function callProtected(): Promise<Response> {
    return fetch('/api/app/auth/me', {
      headers: { Authorization: `Bearer ${tokens.current?.access ?? ''}` },
    })
  }

  async function refresh(): Promise<boolean> {
    const res = await fetch('/api/app/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh: tokens.current?.refresh ?? '' }),
    })
    if (!res.ok) return false

    const body = (await res.json()) as { access: string }
    tokens.current = { access: body.access, refresh: tokens.current!.refresh }
    setRefreshes((n) => n + 1)
    return true
  }

  async function callApi() {
    let res = await callProtected()
    let body = (await res.json()) as { reason?: string; username?: string; role?: string }

    // The refresh path only helps one kind of 401, which is why the endpoint
    // says which kind it gave.
    if (res.status === 401 && body.reason === 'expired' && autoRefresh) {
      if (await refresh()) {
        res = await callProtected()
        body = (await res.json()) as { reason?: string; username?: string; role?: string }
      }
    }

    setStatus(String(res.status))
    setReason(body.reason ?? '—')
    setTokenState(res.ok ? 'valid' : body.reason === 'expired' ? 'expired' : 'rejected')
    setIdentity(res.ok ? `${body.username} (${body.role})` : null)
  }

  return (
    <ChallengePage id="token-refresh">
      <div className="flex flex-wrap items-center gap-3">
        <button className="primary" data-testid="sign-in" onClick={signIn}>
          Sign in
        </button>
        <button data-testid="call-api" onClick={callApi} disabled={tokenState === 'none'}>
          Call the protected endpoint
        </button>
        <label className="flex items-center gap-1 text-sm">
          <input
            type="checkbox"
            data-testid="auto-refresh"
            checked={autoRefresh}
            onChange={(event) => setAutoRefresh(event.target.checked)}
          />
          refresh on 401
        </label>
      </div>

      <dl className="mt-5 grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        <dt className="text-muted">Token</dt>
        <dd data-testid="token-state">{tokenState}</dd>
        <dt className="text-muted">Last status</dt>
        <dd data-testid="last-status">{status}</dd>
        <dt className="text-muted">Reason</dt>
        <dd data-testid="last-reason">{reason}</dd>
        <dt className="text-muted">Refreshes</dt>
        <dd data-testid="refresh-count">{refreshes}</dd>
      </dl>

      {identity && (
        <p className="mt-4" data-testid="identity">
          The server says you are {identity}.
        </p>
      )}

      <p className="mt-5 text-sm text-muted">
        The access token lives sixty seconds on this session's clock. Advance the clock through
        the control plane rather than waiting — the expiry then happens now, every time, in the
        same place.
      </p>
    </ChallengePage>
  )
}
