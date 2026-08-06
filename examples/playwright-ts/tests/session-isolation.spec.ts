import { expect, test } from './fixtures'

const TASKS = '/api/app/optimistic-revert/tasks'

test('two sessions do not see each other mutations', async ({ playwright, baseURL }) => {
  const alice = await playwright.request.newContext({
    baseURL,
    extraHTTPHeaders: { 'X-Playground-Session': 'isolation-alice' },
  })
  const bob = await playwright.request.newContext({
    baseURL,
    extraHTTPHeaders: { 'X-Playground-Session': 'isolation-bob' },
  })

  await alice.post(`${TASKS}/1/toggle?latencyMs=0`)

  const aliceTasks = (await (await alice.get(TASKS)).json()).tasks
  const bobTasks = (await (await bob.get(TASKS)).json()).tasks

  expect(aliceTasks[0].done).toBe(true)
  expect(bobTasks[0].done, 'bob must not observe alice write').toBe(false)

  await alice.dispose()
  await bob.dispose()
})

test('the same seed produces identical content in different sessions', async ({
  playwright,
  baseURL,
}) => {
  const read = async (session: string) => {
    const context = await playwright.request.newContext({
      baseURL,
      extraHTTPHeaders: { 'X-Playground-Session': session },
    })
    const body = await (await context.get('/api/app/virtual-list/rows?count=50')).text()
    await context.dispose()
    return body
  }

  expect(await read('determinism-one')).toBe(await read('determinism-two'))
})

test('a client that asks for no session is given one', async ({ playwright, baseURL }) => {
  const context = await playwright.request.newContext({ baseURL })
  const response = await context.get('/api/challenges')

  const issued = response.headers()['x-playground-session']
  expect(issued, 'the server should name the session it served').toBeTruthy()
  expect((await response.json()).session).toBe(issued)

  await context.dispose()
})

test('a malformed session id is rejected rather than silently replaced', async ({
  playwright,
  baseURL,
}) => {
  const context = await playwright.request.newContext({
    baseURL,
    extraHTTPHeaders: { 'X-Playground-Session': 'not a valid id' },
  })

  expect((await context.get('/api/challenges')).status()).toBe(400)
  await context.dispose()
})
