import { expect, test } from './fixtures'

test('the message is not in the DOM until the delay elapses', async ({ page }) => {
  await page.goto('/app/delayed-element?delayMs=1500')

  await expect(page.getByTestId('delayed-message')).toHaveCount(0)
  await expect(page.getByTestId('delay-pending')).toBeVisible()
})

test('waiting for the element is all it takes', async ({ page }) => {
  await page.goto('/app/delayed-element?delayMs=1500')

  // No sleep, no polling loop. The assertion retries until the element is
  // there or the timeout expires.
  await expect(page.getByTestId('delayed-message')).toHaveText('The element you were waiting for.')
})

test('a sleep shorter than the delay is exactly the mistake this page teaches', async ({
  page,
}) => {
  await page.goto('/app/delayed-element?delayMs=1500')

  await page.waitForTimeout(400)
  await expect(
    page.getByTestId('delayed-message'),
    'a guessed sleep is a guess about someone else machine',
  ).toHaveCount(0)

  await expect(page.getByTestId('delayed-message')).toBeVisible()
})

test('restarting removes the element and brings it back', async ({ page }) => {
  await page.goto('/app/delayed-element?delayMs=800')
  await expect(page.getByTestId('delayed-message')).toBeVisible()

  await page.getByTestId('restart').click()
  await expect(page.getByTestId('delayed-message')).toHaveCount(0)
  await expect(page.getByTestId('delayed-message')).toBeVisible()
})

test('the delay is under the caller control', async ({ page }) => {
  await page.goto('/app/delayed-element?delayMs=0')

  await expect(page.getByTestId('delay-ms')).toHaveText('0')
  await expect(page.getByTestId('delayed-message')).toBeVisible()
})
