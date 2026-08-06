import { expect, test } from './fixtures'

test('the slow page is a server that has not answered yet', async ({ page }) => {
  const started = Date.now()
  await page.goto('/classic/slow-pages/slow?ms=1500')

  expect(Date.now() - started).toBeGreaterThanOrEqual(1400)
  await expect(page.getByTestId('slow-body')).toBeVisible()
  await expect(page.getByTestId('slow-ms')).toHaveText('1500')
})

test('the delay is under the caller control', async ({ page }) => {
  await page.goto('/classic/slow-pages/slow?ms=0')
  await expect(page.getByTestId('slow-ms')).toHaveText('0')
})

test('the hanging page is usable long before it finishes loading', async ({ page }) => {
  await page.goto('/classic/slow-pages/hanging', { waitUntil: 'domcontentloaded' })

  // The document is complete. Only a subresource is still outstanding.
  await expect(page.getByTestId('hanging-body')).toBeVisible()
})

test('waiting for the load event on the hanging page never returns', async ({ page }) => {
  // The same page, the same content, one word different in the wait. This is
  // the difference between a fast test and a guaranteed timeout.
  await expect(
    page.goto('/classic/slow-pages/hanging', { waitUntil: 'load', timeout: 3000 }),
  ).rejects.toThrow(/Timeout/)
})
