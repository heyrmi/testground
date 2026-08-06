import { expect, test } from './fixtures'

const PAGE = '/live/reconnects'

test('a dropped socket is announced by the state, not by the content', async ({ page }) => {
  await page.goto(`${PAGE}?dropAfterMs=400`)
  await page.getByTestId('flaky-connect').click()

  await expect(page.getByTestId('flaky-drops')).toHaveText('1', { timeout: 6000 })

  // The client keeps going, so the generation moves on. Nothing in the
  // messages says any of this happened.
  await expect(page.getByTestId('flaky-generation')).not.toHaveText('1')
})

test('reconnecting recovers, and the generation says which connection you are reading', async ({
  page,
}) => {
  await page.goto(`${PAGE}?dropAfterMs=400`)
  await page.getByTestId('flaky-connect').click()

  await expect(page.getByTestId('flaky-drops')).toHaveText('2', { timeout: 10_000 })
  await expect(page.getByTestId('flaky-generation')).toHaveText('3')

  // Messages accumulate across connections, so the count alone cannot tell
  // you the socket ever died.
  const total = Number(await page.getByTestId('flaky-count').textContent())
  expect(total).toBeGreaterThan(0)
})

test('stopping deliberately is distinguishable from being dropped', async ({ page }) => {
  await page.goto(`${PAGE}?dropAfterMs=5000`)
  await page.getByTestId('flaky-connect').click()
  await expect(page.getByTestId('flaky-state')).toHaveText('open')

  await page.getByTestId('flaky-stop').click()
  await expect(page.getByTestId('flaky-state')).toHaveText('closed')
})

test('messages arrive in an order that is not their numbering', async ({ page }) => {
  await page.goto(`${PAGE}?count=6`)
  await page.getByTestId('shuffled-connect').click()
  await expect(page.getByTestId('shuffled-done')).toBeVisible()

  // Fixed rather than random, so this is an exact assertion rather than a
  // hopeful one.
  await expect(page.getByTestId('arrival-order')).toHaveText('2, 1, 4, 3, 6, 5')
  await expect(page.getByTestId('sorted-order')).toHaveText('1, 2, 3, 4, 5, 6')
})

test('appending in arrival order renders them wrong', async ({ page }) => {
  await page.goto(`${PAGE}?count=4`)
  await page.getByTestId('shuffled-connect').click()
  await expect(page.getByTestId('shuffled-done')).toBeVisible()

  const arrival = await page.getByTestId('arrival-order').textContent()
  const sorted = await page.getByTestId('sorted-order').textContent()
  expect(arrival, 'sorting by sequence is the whole fix').not.toBe(sorted)
})
