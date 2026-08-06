import { expect, test } from './fixtures'

const PAGE = '/app/fake-controls'

test('the switch has no checkbox to check', async ({ page }) => {
  await page.goto(PAGE)
  const toggle = page.getByTestId('toggle')

  // There is no input element, so there is no checked property to read. The
  // state lives on the attributes instead.
  await expect(page.locator('input[type=checkbox]')).toHaveCount(0)
  await expect(toggle).toHaveAttribute('aria-checked', 'false')

  await toggle.click()
  await expect(toggle).toHaveAttribute('aria-checked', 'true')
  await expect(toggle).toHaveAttribute('data-state', 'on')
  await expect(page.getByTestId('toggle-state')).toHaveText('on')
})

test('the switch answers the keyboard too', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('toggle').focus()
  await page.keyboard.press(' ')
  await expect(page.getByTestId('toggle-state')).toHaveText('on')
})

test('reading the stars under the pointer measures the pointer', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('star-4').hover()
  await expect(page.getByTestId('rating-shown')).toHaveText('4')

  // Nothing has been chosen. The stars are drawing the hover.
  await expect(page.getByTestId('rating-value')).toHaveText('0')
})

test('moving the pointer away is what makes the rating readable', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('star-3').click()
  await page.getByTestId('slider-value').hover()

  await expect(page.getByTestId('rating-value')).toHaveText('3')
  await expect(page.getByTestId('rating-shown')).toHaveText('3')
})

test('the slider has no value to set, only a position to drag to', async ({ page }) => {
  await page.goto(PAGE)
  const track = page.getByTestId('slider-track')

  await expect(page.locator('input[type=range]')).toHaveCount(0)
  await expect(page.getByTestId('slider-value')).toHaveText('20')

  // Raw pointer coordinates do not scroll anything into view the way click()
  // and hover() do, so an element below the fold gets a drag that lands on
  // nothing at all and reports no error.
  await track.scrollIntoViewIfNeeded()

  const box = (await track.boundingBox())!
  await page.mouse.move(box.x + box.width * 0.2, box.y + box.height / 2)
  await page.mouse.down()
  await page.mouse.move(box.x + box.width * 0.75, box.y + box.height / 2, { steps: 10 })
  await page.mouse.up()

  const value = Number(await page.getByTestId('slider-value').textContent())
  expect(value).toBeGreaterThan(70)
  expect(value).toBeLessThan(80)
})
