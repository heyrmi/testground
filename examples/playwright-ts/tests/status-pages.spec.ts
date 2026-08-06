import { expect, test } from './fixtures'

test('a rendered error page is still an error', async ({ page }) => {
  const response = await page.goto('/classic/status-pages/500')

  // Every content assertion here passes. The response was a 500.
  await expect(page.getByTestId('status-code')).toHaveText('500')
  await expect(page.getByTestId('status-reason')).toBeVisible()
  expect(response?.status(), 'the DOM cannot tell you this').toBe(500)
})

test('each code is served with its own page', async ({ page }) => {
  for (const code of [400, 401, 403, 404, 418, 429, 500, 502, 503, 504]) {
    const response = await page.goto(`/classic/status-pages/${code}`)
    expect(response?.status(), `${code} page`).toBe(code)
    await expect(page.getByTestId('status-code')).toHaveText(String(code))
  }
})

test('the throttling codes say how long to wait', async ({ page }) => {
  const throttled = await page.goto('/classic/status-pages/429')
  expect(throttled?.status()).toBe(429)
  expect(throttled?.headers()['retry-after']).toBe('5')

  const unavailable = await page.goto('/classic/status-pages/503')
  expect(unavailable?.headers()['retry-after']).toBe('10')
  await expect(page.getByTestId('status-retry-after')).toHaveText('10')
})

test('a code this page does not serve is a clean 404', async ({ page }) => {
  const response = await page.goto('/classic/status-pages/599')
  expect(response?.status()).toBe(404)
})
