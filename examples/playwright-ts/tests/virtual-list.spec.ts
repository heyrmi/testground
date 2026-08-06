import { expect, test } from './fixtures'

const ROW_HEIGHT = 40

test('ten thousand rows exist but only a window of them is rendered', async ({ page }) => {
  await page.goto('/app/virtual-list')

  await expect(page.getByTestId('row-total')).toHaveText('10000')

  const rendered = await page.getByTestId('row').count()
  expect(rendered).toBeGreaterThan(0)
  expect(rendered, 'the list is windowed, not merely long').toBeLessThan(50)
})

test('a row far down the list is simply not there yet', async ({ page }) => {
  await page.goto('/app/virtual-list')

  await expect(page.locator('[data-testid="row"][data-index="9999"]')).toHaveCount(0)
})

test('scroll the container, not the window, and compute the offset', async ({ page }) => {
  await page.goto('/app/virtual-list')
  await expect(page.getByTestId('row').first()).toBeVisible()

  // Rows are a fixed height and positioned by index, so the last row is one
  // jump away rather than a scroll loop.
  await page.getByTestId('viewport').evaluate((el, offset) => {
    el.scrollTop = offset
  }, 9999 * ROW_HEIGHT)

  const row = page.locator('[data-testid="row"][data-index="9999"]')
  await expect(row).toBeVisible()
  await expect(row.getByTestId('row-index')).toHaveText('09999')
})

test('scrolling the window achieves nothing', async ({ page }) => {
  await page.goto('/app/virtual-list')
  await expect(page.getByTestId('row').first()).toBeVisible()

  await page.mouse.wheel(0, 20_000)

  await expect(page.locator('[data-testid="row"][data-index="9999"]')).toHaveCount(0)
})

test('the whole data set is available without a browser', async ({ page }) => {
  const body = await (await page.request.get('/api/app/virtual-list/rows')).json()

  expect(body.count).toBe(10_000)
  expect(body.rows).toHaveLength(10_000)
  expect(body.rows[9999]).toMatchObject({ index: 9999 })
})

test('the row count is under the caller control', async ({ page }) => {
  await page.goto('/app/virtual-list?count=25')

  await expect(page.getByTestId('row-total')).toHaveText('25')

  // Twenty-five rows still overflow a viewport that holds about ten, so the
  // last one is windowed out until the container is scrolled.
  await expect(page.locator('[data-testid="row"][data-index="24"]')).toHaveCount(0)
  await page.getByTestId('viewport').evaluate((el, offset) => {
    el.scrollTop = offset
  }, 24 * ROW_HEIGHT)
  await expect(page.locator('[data-testid="row"][data-index="24"]')).toBeVisible()
})
