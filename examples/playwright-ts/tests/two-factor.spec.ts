import { expect, test } from './fixtures'

const PAGE = '/classic/two-factor'

async function passwordStep(page: import('@playwright/test').Page) {
  await page.goto(PAGE)
  await page.getByTestId('field-username').fill('twofactor')
  await page.getByTestId('field-password').fill('twofactor123')
  await page.getByTestId('submit').click()
}

test('the password alone leaves the login half done', async ({ page }) => {
  await passwordStep(page)

  await expect(page.getByTestId('pending-notice')).toBeVisible()
  await expect(page.getByTestId('welcome')).toHaveCount(0)
  await expect(page.getByTestId('code-form')).toBeVisible()
})

test('the code comes from the back channel, because it cannot be a fixture', async ({ page }) => {
  await passwordStep(page)

  // A recorded code is wrong within thirty seconds of being recorded, so the
  // control plane publishes the one valid on this session's clock right now.
  const { totpCode } = await (await page.request.get('/api/control/auth')).json()
  await page.getByTestId('field-code').fill(totpCode)
  await page.getByTestId('submit-code').click()

  await expect(page.getByTestId('welcome')).toContainText('Tam Two-Factor')
})

test('a test can compute the code itself from the published secret', async ({ page }) => {
  await passwordStep(page)
  const { totpSecret } = await (await page.request.get('/api/control/auth')).json()

  // Proving your own implementation rather than trusting the server's answer.
  const code = await page.evaluate(async (secret: string) => {
    const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567'
    let bits = ''
    for (const ch of secret) bits += alphabet.indexOf(ch).toString(2).padStart(5, '0')
    const bytes = new Uint8Array(Math.floor(bits.length / 8))
    for (let i = 0; i < bytes.length; i++) bytes[i] = parseInt(bits.slice(i * 8, i * 8 + 8), 2)

    const counter = new DataView(new ArrayBuffer(8))
    counter.setUint32(4, Math.floor(Date.now() / 1000 / 30))

    const key = await crypto.subtle.importKey(
      'raw', bytes, { name: 'HMAC', hash: 'SHA-1' }, false, ['sign'],
    )
    const mac = new Uint8Array(await crypto.subtle.sign('HMAC', key, counter.buffer))
    const offset = mac[mac.length - 1]! & 0x0f
    const binary =
      ((mac[offset]! & 0x7f) << 24) | (mac[offset + 1]! << 16) |
      (mac[offset + 2]! << 8) | mac[offset + 3]!
    return String(binary % 1_000_000).padStart(6, '0')
  }, totpSecret)

  await page.getByTestId('field-code').fill(code)
  await page.getByTestId('submit-code').click()
  await expect(page.getByTestId('welcome')).toBeVisible()
})

test('a stale code is refused', async ({ page }) => {
  await passwordStep(page)
  const { totpCode } = await (await page.request.get('/api/control/auth')).json()

  // Move the clock well past the code's window. It follows the session clock,
  // so this is deterministic rather than a race against thirty seconds.
  await page.request.post('/api/control/clock', { data: { action: 'advance', ms: 120_000 } })

  await page.getByTestId('field-code').fill(totpCode)
  await page.getByTestId('submit-code').click()

  await expect(page.getByTestId('login-error')).toContainText('not valid at this moment')
  await expect(page.getByTestId('welcome')).toHaveCount(0)
})

test('the code follows the clock, so a moved clock has its own valid code', async ({ page }) => {
  await passwordStep(page)
  await page.request.post('/api/control/clock', { data: { action: 'advance', ms: 120_000 } })

  const { totpCode } = await (await page.request.get('/api/control/auth')).json()
  await page.getByTestId('field-code').fill(totpCode)
  await page.getByTestId('submit-code').click()

  await expect(page.getByTestId('welcome')).toBeVisible()
})

test('a sign-in link is retrievable without an inbox', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('send-magic-link').click()

  const { magicLinks } = await (await page.request.get('/api/control/auth')).json()
  const tokens = Object.keys(magicLinks)
  expect(tokens).toHaveLength(1)
  expect(magicLinks[tokens[0]!]).toBe('user')

  await page.goto(`${PAGE}/magic/${tokens[0]}`)
  await expect(page.getByTestId('welcome')).toContainText('Uma User')
})

test('a link works exactly once', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('send-magic-link').click()

  const { magicLinks } = await (await page.request.get('/api/control/auth')).json()
  const token = Object.keys(magicLinks)[0]!

  await page.goto(`${PAGE}/magic/${token}`)
  await expect(page.getByTestId('welcome')).toBeVisible()

  await page.request.post('/api/control/auth/reset')
  const second = await page.request.get(`${PAGE}/magic/${token}`)
  expect(second.status()).toBe(404)
})
