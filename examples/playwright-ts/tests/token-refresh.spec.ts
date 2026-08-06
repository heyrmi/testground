import { expect, test } from './fixtures'

const PAGE = '/app/token-refresh'

test('a fresh token works', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('sign-in').click()
  await page.getByTestId('call-api').click()

  await expect(page.getByTestId('last-status')).toHaveText('200')
  await expect(page.getByTestId('identity')).toContainText('user (user)')
})

test('the token expires by moving the clock, not by waiting sixty seconds', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('sign-in').click()
  await page.getByTestId('auto-refresh').uncheck()

  await page.request.post('/api/control/clock', { data: { action: 'advance', ms: 61_000 } })
  await page.getByTestId('call-api').click()

  await expect(page.getByTestId('last-status')).toHaveText('401')
  await expect(page.getByTestId('token-state')).toHaveText('expired')
})

test('expired and invalid are different 401s, and only one is fixed by refreshing', async ({
  page,
}) => {
  await page.goto(PAGE)
  await page.getByTestId('sign-in').click()

  const expired = await page.request.get('/api/app/auth/me', {
    headers: { Authorization: 'Bearer not.a.real.token' },
  })
  expect(expired.status()).toBe(401)
  expect((await expired.json()).reason).toBe('invalid')

  await page.request.post('/api/control/clock', { data: { action: 'advance', ms: 61_000 } })
  await page.getByTestId('auto-refresh').uncheck()
  await page.getByTestId('call-api').click()
  await expect(page.getByTestId('last-reason')).toHaveText('expired')
})

test('refreshing recovers, and the page proves it did', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('sign-in').click()
  await page.request.post('/api/control/clock', { data: { action: 'advance', ms: 61_000 } })

  await page.getByTestId('call-api').click()

  // The call succeeded, and the refresh counter is what distinguishes that
  // from a call that simply never needed refreshing.
  await expect(page.getByTestId('last-status')).toHaveText('200')
  await expect(page.getByTestId('refresh-count')).toHaveText('1')
  await expect(page.getByTestId('identity')).toBeVisible()
})

test('the signing key is published, so an expired token can be built on purpose', async ({
  page,
}) => {
  await page.goto(PAGE)
  const state = await (await page.request.get('/api/control/auth')).json()

  // Not a secret, deliberately: a test that cannot sign cannot construct the
  // token it needs to exercise this path without waiting for one.
  expect(state.signingKey).toMatch(/^[0-9a-f]{64}$/)
})

test('a refresh token is not an access token', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('sign-in').click()

  const tokens = await (
    await page.request.post('/api/app/auth/login', {
      data: { username: 'user', password: 'user123' },
    })
  ).json()

  const wrongKind = await page.request.get('/api/app/auth/me', {
    headers: { Authorization: `Bearer ${tokens.refresh}` },
  })
  expect(wrongKind.status()).toBe(401)
  expect((await wrongKind.json()).reason).toBe('wrong-kind')
})
