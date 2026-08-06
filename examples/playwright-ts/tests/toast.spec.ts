import { expect, test } from './fixtures'

test('the toast renders outside the app root', async ({ page }) => {
  await page.goto('/app/toast?dismissMs=20000')
  await page.getByTestId('show-toast').click()

  await expect(page.getByTestId('toast')).toBeVisible()

  // Scoping to the React root is the mistake: the portal mounts on body.
  await expect(page.locator('#root').getByTestId('toast')).toHaveCount(0)
  await expect(page.locator('body > [data-testid="toast-region"]')).toHaveCount(1)
})

test('the toast leaves the DOM on its own', async ({ page }) => {
  await page.goto('/app/toast?dismissMs=1000')
  await page.getByTestId('show-toast').click()

  await expect(page.getByTestId('toast')).toBeVisible()
  await expect(page.getByTestId('toast')).toHaveCount(0)
  await expect(page.getByTestId('toast-last')).toHaveText('1')
})

test('counters outlive the toast, so assert against those', async ({ page }) => {
  await page.goto('/app/toast?dismissMs=500')

  await page.getByTestId('show-toast').click()
  await page.getByTestId('show-toast').click()
  await expect(page.getByTestId('toast-live')).toHaveText('0')

  // The toasts are long gone, but the fact that they happened is not.
  await expect(page.getByTestId('toast-count')).toHaveText('2')
  await expect(page.getByTestId('toast-last')).toHaveText('2')
})

test('two toasts make one test id match two nodes', async ({ page }) => {
  await page.goto('/app/toast?dismissMs=20000')
  await page.getByTestId('show-toast').click()
  await page.getByTestId('show-toast').click()

  await expect(page.getByTestId('toast')).toHaveCount(2)

  // A bare locator is ambiguous here and strict mode says so. Narrow by the
  // attribute that distinguishes them rather than reaching for .first().
  await expect(
    page.getByTestId('toast').filter({ hasText: 'Saved change #2' }),
  ).toHaveCount(1)
  await expect(page.locator('[data-testid="toast"][data-sequence="1"]')).toBeVisible()
})
