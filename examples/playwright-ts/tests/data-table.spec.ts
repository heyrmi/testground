import { expect, test } from './fixtures'

const PAGE = '/app/data-table'

test('sorting is a round trip, so the old rows are still on screen', async ({ page }) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('row')).toHaveCount(10)

  const before = await page.getByTestId('row').first().getAttribute('data-id')
  await page.getByTestId('sort-name').click()

  // The table publishes the sort its rows were fetched with, so waiting for
  // that is waiting for the thing that actually changed.
  await expect(page.getByTestId('current-sort')).toHaveText('name asc')

  const after = await page.getByTestId('row').first().getAttribute('data-id')
  expect(after).not.toBe(before)
})

test('clicking the same header again reverses it', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('sort-amount').click()
  await expect(page.getByTestId('current-sort')).toHaveText('amount asc')

  await page.getByTestId('sort-amount').click()
  await expect(page.getByTestId('current-sort')).toHaveText('amount desc')
})

test('amounts sort numerically, not as text', async ({ page }) => {
  await page.goto(`${PAGE}?size=5`)
  await page.getByTestId('sort-amount').click()
  await expect(page.getByTestId('current-sort')).toHaveText('amount asc')

  const amounts = await page.locator('[data-testid="row"] td:nth-child(4)').allTextContents()
  const numbers = amounts.map(Number)
  expect(numbers).toEqual([...numbers].sort((a, b) => a - b))
})

test('select-all has a third state that is a property, not an attribute', async ({ page }) => {
  await page.goto(PAGE)
  const all = page.getByTestId('select-all')

  await page.getByTestId('select-row').first().check()
  await expect(page.getByTestId('selected-count')).toHaveText('1')

  // Neither checked nor unchecked. It appears nowhere in the markup.
  await expect(all).not.toBeChecked()
  expect(await all.evaluate((el: HTMLInputElement) => el.indeterminate)).toBe(true)

  await all.check()
  expect(await all.evaluate((el: HTMLInputElement) => el.indeterminate)).toBe(false)
  await expect(page.getByTestId('selected-count')).toHaveText('10')
})

test('a selection survives paging, because it is not stored in the rows', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('select-all').check()
  await expect(page.getByTestId('selected-count')).toHaveText('10')

  await page.getByTestId('page-next').click()
  await expect(page.getByTestId('page-label')).toHaveText('page 2 of 12')

  await expect(page.getByTestId('selected-count')).toHaveText('10')
  await expect(page.getByTestId('select-all')).not.toBeChecked()
})

test('an edited cell is not saved until focus leaves it', async ({ page }) => {
  await page.goto(PAGE)

  const cell = page.getByTestId('cell-note').first()
  await cell.click()
  await page.getByTestId('cell-note-input').first().fill('written but not committed')

  // Still in the input. Nothing has been saved.
  await expect(page.getByTestId('cell-note')).toHaveCount(9)

  await page.getByTestId('cell-note-input').first().blur()
  await expect(page.getByTestId('cell-note').first()).toHaveAttribute(
    'data-committed',
    'written but not committed',
  )
})

test('empty is an outcome, not a slow success', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('filter').fill('nobodyisnamedthis')

  await expect(page.getByTestId('table-empty')).toBeVisible()
  await expect(page.getByTestId('total-rows')).toHaveText('0')
  await expect(page.getByTestId('table')).toHaveCount(0)
})

test('error is an outcome too, and says so', async ({ page }) => {
  await page.goto(`${PAGE}?state=error`)

  await expect(page.getByTestId('table-error')).toBeVisible()
  await expect(page.getByTestId('table-empty')).toHaveCount(0)
  await expect(page.getByTestId('table-loading')).toHaveCount(0)
})

test('the loading state is distinguishable from both of them', async ({ page }) => {
  await page.goto(`${PAGE}?state=slow`)

  await expect(page.getByTestId('table-loading')).toBeVisible()
  await expect(page.getByTestId('table')).toBeVisible({ timeout: 5000 })
  await expect(page.getByTestId('table-loading')).toHaveCount(0)
})

test('filtering resets to the first page', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('page-next').click()
  await expect(page.getByTestId('page-label')).toHaveText('page 2 of 12')

  await page.getByTestId('filter').fill('a')
  await expect(page.getByTestId('page-label')).toContainText('page 1 of')
})
