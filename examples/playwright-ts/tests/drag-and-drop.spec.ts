import { expect, test } from './fixtures'

const PAGE = '/app/drag-and-drop'

test('native drag and drop moves a parcel', async ({ page }) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('delivered-count')).toHaveText('0')

  await page
    .locator('[data-testid="parcel"][data-name="crate"]')
    .dragTo(page.getByTestId('dropzone'))

  await expect(page.getByTestId('delivered-count')).toHaveText('1')
  await expect(page.locator('[data-testid="delivered"][data-name="crate"]')).toBeVisible()
})

test('moving the mouse across the screen is not a drag', async ({ page }) => {
  await page.goto(PAGE)

  const parcel = page.locator('[data-testid="parcel"][data-name="letter"]')
  const zone = page.getByTestId('dropzone')
  const from = (await parcel.boundingBox())!
  const to = (await zone.boundingBox())!

  await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2)
  await page.mouse.down()
  await page.mouse.move(to.x + to.width / 2, to.y + to.height / 2, { steps: 12 })
  await page.mouse.up()

  // Native drag is a separate event family the browser raises on the operating
  // system's behalf. Mouse movement does not produce it.
  await expect(page.getByTestId('delivered-count')).toHaveText('0')
})

test('the list reorders by dropping one item onto another', async ({ page }) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('sortable-order')).toHaveText('one, two, three, four')

  await page
    .locator('[data-testid="sortable-item"][data-name="four"]')
    .dragTo(page.locator('[data-testid="sortable-item"][data-name="one"]'))

  await expect(page.getByTestId('sortable-order')).toHaveText('four, one, two, three')
})

test('the pointer handle needs a press, a move and a release', async ({ page }) => {
  await page.goto(PAGE)
  const rail = page.getByTestId('rail')
  await rail.scrollIntoViewIfNeeded()

  const box = (await rail.boundingBox())!
  await page.mouse.move(box.x + 4, box.y + box.height / 2)
  await page.mouse.down()
  await page.mouse.move(box.x + box.width * 0.6, box.y + box.height / 2, { steps: 8 })
  await page.mouse.up()

  const position = Number(await page.getByTestId('handle-position').textContent())
  expect(position).toBeGreaterThan(50)
  expect(position).toBeLessThan(70)
})

test('the asymmetry: one technique covers both, the other covers one', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('rail').scrollIntoViewIfNeeded()

  // dragTo drives real mouse input and turns on the browser's drag
  // interception, so it satisfies the pointer handle as well as the native
  // parcels -- moving the handle even though nothing here listens for a drag.
  await page.getByTestId('handle').dragTo(page.getByTestId('rail'))
  expect(Number(await page.getByTestId('handle-position').textContent())).toBeGreaterThan(0)

  // Raw mouse events go the other way: they move the handle and leave the
  // native parcels exactly where they were, as the test above shows.
  await expect(page.getByTestId('delivered-count')).toHaveText('0')
})
