import { expect, test } from './fixtures'

const PAGE = '/classic/form-login'

async function signIn(page: import('@playwright/test').Page, user = 'admin', pass = 'admin123') {
  await page.goto(PAGE)
  await page.getByTestId('field-username').fill(user)
  await page.getByTestId('field-password').fill(pass)
  await page.getByTestId('submit').click()
}

test('a correct password signs you in', async ({ page }) => {
  await signIn(page)

  await expect(page.getByTestId('welcome')).toContainText('Ada Admin')
  await expect(page.getByTestId('current-role')).toHaveText('admin')
})

test('the three refusals have three different statuses', async ({ page }) => {
  await page.goto(PAGE)
  const csrf = await page.getByTestId('csrf-token').inputValue()

  // Wrong password: the form is fine, the credentials are not.
  const wrong = await page.request.post(PAGE, {
    form: { csrf, username: 'admin', password: 'nope' },
  })
  expect(wrong.status()).toBe(200)

  // Missing token: refused before the password is even considered.
  const noToken = await page.request.post(PAGE, {
    form: { username: 'admin', password: 'admin123' },
  })
  expect(noToken.status()).toBe(403)
})

test('the CSRF token belongs to this session and no other', async ({ page, playwright, baseURL }) => {
  await page.goto(PAGE)

  const other = await playwright.request.newContext({
    baseURL,
    extraHTTPHeaders: { 'X-Playground-Session': 'csrf-thief' },
  })
  const stolen = await (await other.get(PAGE)).text()
  const foreign = stolen.match(/name="csrf" value="([a-f0-9]+)"/)![1]!
  await other.dispose()

  const response = await page.request.post(PAGE, {
    form: { csrf: foreign, username: 'admin', password: 'admin123' },
  })
  expect(response.status(), 'a token from another worker is not a token').toBe(403)
})

test('five wrong passwords throttle the sixth, however right it is', async ({ page }) => {
  await page.goto(PAGE)
  const csrf = await page.getByTestId('csrf-token').inputValue()

  for (let i = 0; i < 5; i++) {
    await page.request.post(PAGE, { form: { csrf, username: 'admin', password: 'nope' } })
  }

  const correct = await page.request.post(PAGE, {
    form: { csrf, username: 'admin', password: 'admin123' },
  })
  expect(correct.status(), 'the throttle is about the attempts, not the credentials').toBe(429)
})

test('the throttle lifts by moving the clock, not by waiting', async ({ page }) => {
  await page.goto(PAGE)
  const csrf = await page.getByTestId('csrf-token').inputValue()
  for (let i = 0; i < 5; i++) {
    await page.request.post(PAGE, { form: { csrf, username: 'admin', password: 'nope' } })
  }

  await page.request.post('/api/control/clock', { data: { action: 'advance', ms: 40_000 } })

  const after = await page.request.post(PAGE, {
    form: { csrf, username: 'admin', password: 'admin123' },
    maxRedirects: 0,
  })
  expect(after.status()).toBe(303)
})

test('a suite that fails logins on purpose has to clean up after itself', async ({ page }) => {
  await page.goto(PAGE)
  const csrf = await page.getByTestId('csrf-token').inputValue()
  for (let i = 0; i < 5; i++) {
    await page.request.post(PAGE, { form: { csrf, username: 'admin', password: 'nope' } })
  }

  // Without this, every later test in this worker meets a throttle it did not
  // cause and cannot explain.
  await page.request.post('/api/control/auth/reset')

  await signIn(page)
  await expect(page.getByTestId('welcome')).toBeVisible()
})

test('remember me is recorded on the login', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('field-username').fill('user')
  await page.getByTestId('field-password').fill('user123')
  await page.getByTestId('field-remember').check()
  await page.getByTestId('submit').click()

  await expect(page.getByTestId('remembered')).toBeVisible()
})

test('logging out ends the login on the server', async ({ page }) => {
  await signIn(page)
  await page.getByTestId('logout').click()

  await expect(page.getByTestId('login-form')).toBeVisible()

  // The server is the authority, not the browser.
  const state = await (await page.request.get('/api/control/auth')).json()
  expect(state.login).toBeNull()
})
