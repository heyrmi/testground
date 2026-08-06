import { expect, test } from './fixtures'

const PAGE = '/app/detached-elements'

test('a handle taken before a rebuild is detached after it', async ({ page }) => {
  await page.goto(`${PAGE}?churnMs=200`)

  const handle = await page.locator('[data-testid="unstable-row"][data-name="charlie"]').elementHandle()
  await page.getByTestId('toggle-churn').click()
  await expect(page.getByTestId('generation')).not.toHaveText('0')

  // The row is on screen. This is not that row.
  expect(await handle!.evaluate((el) => el.isConnected)).toBe(false)
  await expect(page.locator('[data-testid="unstable-row"][data-name="charlie"]')).toBeVisible()
})

test('a locator survives what a handle does not', async ({ page }) => {
  await page.goto(`${PAGE}?churnMs=200`)
  await page.getByTestId('toggle-churn').click()
  await expect(page.getByTestId('generation')).not.toHaveText('0')

  // Re-resolved on use, so the rebuild is simply not its problem.
  await page
    .locator('[data-testid="unstable-row"][data-name="delta"]')
    .getByTestId('row-action')
    .click()

  await expect(page.getByTestId('chosen')).toContainText('delta')
})

test('the DOM ids are correct until the next tick', async ({ page }) => {
  await page.goto(`${PAGE}?churnMs=200`)

  const before = await page.getByTestId('row-dom-id').first().textContent()
  await page.getByTestId('toggle-churn').click()
  await expect(page.getByTestId('generation')).not.toHaveText('0')

  const after = await page.getByTestId('row-dom-id').first().textContent()
  expect(after, 'a selector built from the old id is now wrong').not.toBe(before)
  await expect(page.locator(`#${before}`)).toHaveCount(0)
})

test('the vanishing button has to be caught', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('summon').click()
  await page.getByTestId('vanishing').click()
  await expect(page.getByTestId('vanish-clicks')).toHaveText('1')

  await expect(page.getByTestId('vanishing')).toHaveCount(0)
})

test('a field can unmount while it is being filled in', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('arm-unmount').click()
  await page.getByTestId('doomed-field').fill('half a sen')

  await expect(page.getByTestId('form-gone')).toBeVisible()
  await expect(page.getByTestId('doomed-field')).toHaveCount(0)
})
