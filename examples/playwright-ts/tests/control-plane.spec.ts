import { expect, test } from './fixtures'
import type { APIRequestContext } from '@playwright/test'

const API = '/api/control'

test('nothing misbehaves until it is asked to', async ({ page }) => {
  const state = await (await page.request.get(`${API}/state`)).json()

  expect(state.control.latency).toEqual([])
  expect(state.control.failures).toEqual([])
  expect(state.control.flakes).toEqual([])
  expect(state.clock.frozen).toBe(false)
  expect(state.seed).toBe(42)
})

test('an injected failure is marked as injected', async ({ page }) => {
  await page.request.post(`${API}/failure`, {
    data: { route: '/api/challenges', status: 503, message: 'asked for' },
  })

  const response = await page.request.get('/api/challenges')
  expect(response.status()).toBe(503)

  // A 503 you asked for and a 503 you did not look identical otherwise.
  expect(response.headers()['x-playground-injected']).toBe('503')
  expect((await response.json()).error).toBe('asked for')
})

test('failures replay for the same seed', async ({ playwright, baseURL }) => {
  const pattern = async (session: string) => {
    const worker = await playwright.request.newContext({
      baseURL,
      extraHTTPHeaders: { 'X-Playground-Session': session },
    })
    await worker.post(`${API}/failure`, {
      data: { route: '/api/challenges', status: 500, rate: 0.5 },
    })

    const statuses: number[] = []
    for (let i = 0; i < 16; i++) {
      statuses.push((await worker.get('/api/challenges')).status())
    }
    await worker.dispose()
    return statuses.join(',')
  }

  // Two sessions on the same seed meet exactly the same run of failures,
  // which is what makes an injected-chaos failure debuggable.
  expect(await pattern('replay-a')).toBe(await pattern('replay-b'))
})

test('a different seed produces a different run of failures', async ({ playwright, baseURL }) => {
  const pattern = async (session: string, seed: number) => {
    const worker = await playwright.request.newContext({
      baseURL,
      extraHTTPHeaders: { 'X-Playground-Session': session },
    })
    await worker.post(`${API}/seed`, { data: { seed } })
    await worker.post(`${API}/failure`, {
      data: { route: '/api/challenges', status: 500, rate: 0.5 },
    })

    const statuses: number[] = []
    for (let i = 0; i < 24; i++) {
      statuses.push((await worker.get('/api/challenges')).status())
    }
    await worker.dispose()
    return statuses.join(',')
  }

  expect(await pattern('seed-one', 1)).not.toBe(await pattern('seed-two', 2))
})

test('fail the first three calls, then recover', async ({ page }) => {
  await page.request.post(`${API}/failure`, {
    data: { route: '/api/challenges', status: 503, times: 3 },
  })

  const statuses: number[] = []
  for (let i = 0; i < 5; i++) {
    statuses.push((await page.request.get('/api/challenges')).status())
  }
  expect(statuses).toEqual([503, 503, 503, 200, 200])
})

test('latency is applied and is removable', async ({ page }) => {
  await page.request.post(`${API}/latency`, { data: { route: '/api/challenges', ms: 600 } })

  const slow = Date.now()
  await page.request.get('/api/challenges')
  expect(Date.now() - slow).toBeGreaterThan(500)

  await page.request.post(`${API}/latency`, { data: { route: '/api/challenges', ms: 0 } })

  const quick = Date.now()
  await page.request.get('/api/challenges')
  expect(Date.now() - quick).toBeLessThan(400)
})

test('the clock is yours to move', async ({ page }) => {
  await page.request.post(`${API}/clock`, { data: { action: 'freeze' } })
  const frozen = await (await page.request.get(`${API}/state`)).json()
  expect(frozen.clock.frozen).toBe(true)

  await page.request.post(`${API}/clock`, { data: { action: 'advance', ms: 86_400_000 } })
  const advanced = await (await page.request.get(`${API}/state`)).json()
  expect(new Date(advanced.clock.now).getTime() - new Date(frozen.clock.now).getTime()).toBe(
    86_400_000,
  )

  await page.request.post(`${API}/clock`, { data: { action: 'reset' } })
  expect((await (await page.request.get(`${API}/state`)).json()).clock.frozen).toBe(false)
})

test('a catch-all failure cannot lock you out of the control plane', async ({ page }) => {
  await page.request.post(`${API}/failure`, { data: { route: '*', status: 500 } })

  expect((await page.request.get('/api/challenges')).status()).toBe(500)

  // The escape hatch has to work even when everything else is refusing.
  expect((await page.request.get(`${API}/state`)).status()).toBe(200)
  expect((await page.request.post(`${API}/reset`)).status()).toBe(200)
  expect((await page.request.get('/api/challenges')).status()).toBe(200)
})

test('reset clears the rules and keeps the seed', async ({ page }) => {
  await page.request.post(`${API}/seed`, { data: { seed: 99 } })
  await page.request.post(`${API}/latency`, { data: { route: '/api/*', ms: 100 } })
  await page.request.post(`${API}/feature`, { data: { flag: 'beta', enabled: true } })

  const state = await (await page.request.post(`${API}/reset`)).json()

  expect(state.control.latency).toEqual([])
  expect(state.control.features).toEqual({})
  expect(state.seed, 'a suite picks a seed once and resets between tests').toBe(99)
})

test('bad input is refused with a reason', async ({ page }) => {
  const cases: [string, unknown, string][] = [
    ['failure', { route: '/x', status: 500, rate: 5 }, 'rate must be between 0 and 1'],
    ['flake', { challenge: 'toast', probability: -1 }, 'probability must be between 0 and 1'],
    ['clock', { action: 'sideways' }, 'action must be'],
    ['latency', { ms: 100 }, 'route is required'],
    ['seed', {}, 'seed is required'],
  ]

  for (const [route, data, expected] of cases) {
    const response = await page.request.post(`${API}/${route}`, { data })
    expect(response.status(), route).toBe(400)
    expect((await response.json()).error, route).toContain(expected)
  }
})

// The property everything else rests on: many workers driving their own copy
// of the playground into different states at the same time, none of them
// seeing another's.
test('eight parallel workers do not disturb each other', async ({ playwright, baseURL }) => {
  const open = (name: string) =>
    playwright.request.newContext({
      baseURL,
      extraHTTPHeaders: { 'X-Playground-Session': name },
    })

  const workers = await Promise.all(
    Array.from({ length: 8 }, (_, i) => open(`parallel-worker-${i}`)),
  )

  // Each worker sets a seed and a failure rule nobody else asked for, then
  // hammers the endpoint its own rule covers.
  await Promise.all(
    workers.map(async (worker, i) => {
      await worker.post(`${API}/seed`, { data: { seed: 100 + i } })
      await worker.post(`${API}/failure`, {
        data: { route: '/api/challenges', status: 400 + i, times: 1 },
      })
      await worker.post(`${API}/feature`, { data: { flag: `flag-${i}`, enabled: true } })
    }),
  )

  const check = async (worker: APIRequestContext, i: number) => {
    // The one failure this worker asked for, with its own status.
    expect((await worker.get('/api/challenges')).status()).toBe(400 + i)
    expect((await worker.get('/api/challenges')).status()).toBe(200)

    const state = await (await worker.get(`${API}/state`)).json()
    expect(state.session).toBe(`parallel-worker-${i}`)
    expect(state.seed).toBe(100 + i)
    expect(Object.keys(state.control.features)).toEqual([`flag-${i}`])
  }

  await Promise.all(workers.map(check))
  await Promise.all(workers.map((worker) => worker.dispose()))
})
