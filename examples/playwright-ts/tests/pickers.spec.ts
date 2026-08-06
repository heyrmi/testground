import { expect, test } from './fixtures'

const PAGE = '/classic/pickers'

test('the slider moves with the keyboard, which fires the events a person would', async ({
  page,
}) => {
  await page.goto(PAGE)
  const slider = page.getByTestId('field-volume')

  await expect(slider).toHaveValue('30')
  await slider.focus()
  await slider.press('ArrowRight')
  await slider.press('ArrowRight')

  // step=10, so two presses move it twenty.
  await expect(slider).toHaveValue('50')

  await page.getByTestId('submit').click()
  await expect(page.getByTestId('result-volume')).toHaveText('50')
})

test('the colour input can only be set, never clicked through', async ({ page }) => {
  await page.goto(PAGE)

  // Clicking opens an operating-system dialog no driver can reach. Setting the
  // value is not a clever workaround, it is the only route there is.
  await page.getByTestId('field-colour').fill('#2f7d4f')
  await page.getByTestId('submit').click()

  await expect(page.getByTestId('result-colour')).toHaveText('#2f7d4f')
})

test('date inputs post the format the specification fixes, not the one displayed', async ({
  page,
}) => {
  await page.goto(PAGE)

  await page.getByTestId('field-date').fill('2026-03-14')
  await page.getByTestId('field-time').fill('09:30')
  await page.getByTestId('field-month').fill('2026-03')
  await page.getByTestId('field-week').fill('2026-W11')
  await page.getByTestId('submit').click()

  await expect(page.getByTestId('result-date')).toHaveText('2026-03-14')
  await expect(page.getByTestId('result-time')).toHaveText('09:30')
  await expect(page.getByTestId('result-month')).toHaveText('2026-03')
  await expect(page.getByTestId('result-week')).toHaveText('2026-W11')
})

test('a date outside the allowed range fails validation before posting', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('field-deadline').fill('2020-01-01')
  await page.getByTestId('submit').click()

  // The browser blocks the submit, so nothing reaches the server.
  await expect(page.getByTestId('no-submission')).toBeVisible()

  const valid = await page
    .getByTestId('field-deadline')
    .evaluate((el: HTMLInputElement) => el.validity.rangeUnderflow)
  expect(valid).toBe(true)
})

test('a date inside the range posts normally', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('field-deadline').fill('2026-06-15')
  await page.getByTestId('submit').click()

  await expect(page.getByTestId('result-deadline')).toHaveText('2026-06-15')
})
