import { expect, test } from './fixtures'

const PAGE = '/legacy/ajax-search'

test('typing eight characters does not send eight requests', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('search-input').pressSequentially('kaminski', { delay: 30 })
  await expect(page.getByTestId('search-spinner')).toHaveCount(0)

  // The debounce collapses the burst into one request.
  await expect(page.getByTestId('request-count')).toHaveText('1')
})

test('asserting before the debounce measures a prefix, or nothing', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('search-input').fill('zof')

  // Immediately after typing, no request has been sent at all.
  await expect(page.getByTestId('request-count')).toHaveText('0')

  // Waiting for the state rather than for a duration is what makes this work.
  await expect(page.getByTestId('search-count')).toHaveText('16')
})

test('the total is not the number of rows on screen', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('search-input').fill('an')

  await expect(page.getByTestId('search-count')).toHaveText('88')
  await expect(page.getByTestId('search-shown')).toHaveText('25')
  await expect(page.getByTestId('search-row')).toHaveCount(25)
})

test('rows are detached by the next search, not updated', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('search-input').fill('zof')
  await expect(page.getByTestId('search-row').first()).toBeVisible()

  const held = await page.getByTestId('search-row').first().elementHandle()

  await page.getByTestId('search-input').fill('ova')
  await expect(page.getByTestId('search-empty')).toBeVisible()

  // The container was replaced wholesale, so the row is gone rather than blank.
  expect(await held!.evaluate((el) => el.isConnected)).toBe(false)
})

test('a query matching nothing says so', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('search-input').fill('ova')

  await expect(page.getByTestId('search-empty')).toContainText('No names match')
  await expect(page.getByTestId('search-count')).toHaveText('0')
})
