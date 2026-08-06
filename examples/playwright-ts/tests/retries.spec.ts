import { expect, test } from './fixtures'

const PAGE = '/app/retries'

test('retrying eventually succeeds where asking once does not', async ({ page }) => {
  await page.goto(`${PAGE}?failFirst=2`)

  await page.getByTestId('fetch-once').click()
  await expect(page.getByTestId('outcome')).toHaveText('failed')
  await expect(page.getByTestId('attempt-count')).toHaveText('1')

  await page.getByTestId('reset-endpoint').click()
  await page.getByTestId('fetch-retrying').click()

  // Same server, same endpoint, opposite result. Which button was pressed is
  // the whole difference.
  await expect(page.getByTestId('outcome')).toHaveText('succeeded')
  await expect(page.getByTestId('attempt-count')).toHaveText('3')
})

test('the outcome alone cannot tell three attempts from one', async ({ page }) => {
  await page.goto(`${PAGE}?failFirst=0`)
  await page.getByTestId('fetch-retrying').click()

  await expect(page.getByTestId('outcome')).toHaveText('succeeded')

  // Identical outcome to the test above, and a completely different story.
  await expect(page.getByTestId('attempt-count')).toHaveText('1')
})

test('every attempt is recorded with the status it got', async ({ page }) => {
  await page.goto(`${PAGE}?failFirst=2`)
  await page.getByTestId('fetch-retrying').click()
  await expect(page.getByTestId('outcome')).toHaveText('succeeded')

  await expect(page.getByTestId('attempt-row')).toHaveCount(3)
  await expect(page.locator('[data-testid="attempt-row"][data-status="503"]')).toHaveCount(2)
  await expect(page.locator('[data-testid="attempt-row"][data-status="200"]')).toHaveCount(1)
})

test('the refusal counter is per session', async ({ page, playwright, baseURL }) => {
  await page.goto(`${PAGE}?failFirst=1`)
  await page.getByTestId('fetch-retrying').click()
  await expect(page.getByTestId('outcome')).toHaveText('succeeded')

  // A different worker meets a fresh endpoint rather than one this test spent.
  const other = await playwright.request.newContext({
    baseURL,
    extraHTTPHeaders: { 'X-Playground-Session': 'retries-other-worker' },
  })
  const first = await other.get('/api/app/retries/data?failFirst=1')
  expect(first.status()).toBe(503)
  await other.dispose()
})
