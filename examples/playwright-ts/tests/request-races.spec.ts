import { expect, test } from './fixtures'

const PAGE = '/app/request-races'

test('the older answer wins, and the page looks finished either way', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('run-race').click()

  // The fast search answers first and both panels agree.
  await expect(page.getByTestId('naive-result')).toHaveText('fast')
  await expect(page.getByTestId('guarded-result')).toHaveText('fast')

  // Then the older, slower answer arrives and overwrites the newer one.
  await expect(page.getByTestId('naive-result')).toHaveText('slow')
  await expect(page.getByTestId('guarded-result')).toHaveText('fast')
})

test('waiting for the network to go quiet agrees with the bug', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('run-race').click()

  // No spinner, no error, a plausible result -- and it answers a question the
  // user has already moved on from.
  await page.waitForLoadState('networkidle')
  await expect(page.getByTestId('naive-result')).toHaveText('slow')

  // Asserting that the result matches the last thing requested is what
  // separates the two, and it is barely more work to write.
  await expect(page.getByTestId('guarded-result')).toHaveText('fast')
})

test('a waterfall costs the sum of its steps, not the slowest', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('run-waterfall').click()

  await expect(page.getByTestId('waterfall-done')).toBeVisible()
  await expect(page.getByTestId('waterfall-step')).toHaveCount(3)

  const total = Number(await page.getByTestId('waterfall-total').textContent())
  expect(total, 'three sequential 250 ms steps').toBeGreaterThan(700)
})

test('the steps finish in order, because each waits for the last', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('run-waterfall').click()
  await expect(page.getByTestId('waterfall-done')).toBeVisible()

  await expect(page.getByTestId('waterfall-step')).toHaveText(['first', 'second', 'third'])
})

test('waiting for the first response reads the page two requests too early', async ({ page }) => {
  await page.goto(PAGE)

  const firstResponse = page.waitForResponse((res) => res.url().includes('/races/step'))
  await page.getByTestId('run-waterfall').click()
  await firstResponse

  await expect(page.getByTestId('waterfall-done')).toHaveCount(0)
  await expect(page.getByTestId('waterfall-step')).not.toHaveCount(3)
})
