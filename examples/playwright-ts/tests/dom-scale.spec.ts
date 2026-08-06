import { expect, test } from './fixtures'

const PAGE = '/app/dom-scale'

test('the page says nothing different when it gets heavy', async ({ page }) => {
  await page.goto(`${PAGE}?nodes=15000`)
  await expect(page.getByTestId('node-count')).toHaveText('0')

  await page.getByTestId('build-nodes').click()
  await expect(page.getByTestId('node-count')).toHaveText('15000')

  // Every content assertion on the page still passes. Nothing reports a cost.
  await expect(page.getByTestId('thread-state')).toHaveText('free')
})

test('the cost is in the volume, which is exact, not in a stopwatch', async ({ page }) => {
  await page.goto(`${PAGE}?nodes=25000`)
  await page.getByTestId('build-nodes').click()
  await expect(page.getByTestId('node-count')).toHaveText('25000')

  // Twenty-five thousand elements a document-wide query has to walk, and
  // twenty-five thousand strings it has to marshal back. That is the cost,
  // and it is a number rather than a feeling.
  await expect(page.locator('.scale-cell')).toHaveCount(25_000)
  expect((await page.getByTestId('node-host').locator('.scale-cell').allTextContents()).length)
    .toBe(25_000)

  // Deliberately no wall-clock assertion. Timing one locator against another
  // inside a suite measures the machine it happens to be running on, and a
  // test that fails on a busy CI runner and passes on a laptop teaches the
  // wrong lesson. Time this by hand when you want the number.
  await expect(page.locator('[data-index="24999"]')).toHaveCount(1)
})

test('a blocked thread is not a slow response, and needs a different fix', async ({ page }) => {
  await page.goto(`${PAGE}?blockMs=1500`)

  const started = Date.now()
  await page.getByTestId('block-thread').click()

  // Nothing in the page can run while the page is not running, so a poll that
  // executes in the page cannot observe the block while it is happening.
  await expect(page.getByTestId('thread-state')).toHaveText('free', { timeout: 10_000 })
  expect(Date.now() - started).toBeGreaterThan(1200)
})

test('listeners attach without changing anything the page says', async ({ page }) => {
  await page.goto(`${PAGE}?nodes=2000`)
  await page.getByTestId('build-nodes').click()

  await page.getByTestId('attach-listeners').click()
  await expect(page.getByTestId('listener-count')).toHaveText('500')
})

test('layout thrash is a state, not an appearance', async ({ page }) => {
  await page.goto(`${PAGE}?nodes=1500`)
  await page.getByTestId('build-nodes').click()

  await page.getByTestId('toggle-thrash').click()
  await expect(page.getByTestId('toggle-thrash')).toHaveAttribute('data-thrashing', 'true')

  await page.getByTestId('toggle-thrash').click()
  await expect(page.getByTestId('toggle-thrash')).toHaveAttribute('data-thrashing', 'false')
})

test('the leak is only observable because this page chose to report it', async ({ page }) => {
  await page.goto(PAGE)

  for (let i = 0; i < 3; i++) await page.getByTestId('leak').click()
  await expect(page.getByTestId('retained-count')).toHaveText('3')

  // In a real application nothing would say this, which is the lesson: the
  // counter is the affordance, not the leak.
  await expect(page.getByTestId('thread-state')).toHaveText('free')
})
