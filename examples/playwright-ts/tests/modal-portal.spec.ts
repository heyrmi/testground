import { expect, test } from './fixtures'

const PAGE = '/app/modal-portal'

test('the dialog is not inside the application root', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('open-modal').click()

  await expect(page.getByTestId('modal')).toBeVisible()

  // Scoping to the component tree finds nothing, which is the whole trap.
  await expect(page.locator('#root').getByTestId('modal')).toHaveCount(0)
  await expect(page.locator('body > [data-testid="modal-overlay"]')).toHaveCount(1)
})

test('the background is enabled, visible, and unclickable', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('open-modal').click()

  const background = page.getByTestId('background-button')
  await expect(background).toBeEnabled()

  const error = await background.click({ timeout: 2000 }).catch((e: Error) => e.message)
  expect(error, 'the error names the overlay, which the test never mentioned').toContain(
    'intercepts pointer events',
  )
  await expect(page.getByTestId('background-clicks')).toHaveText('0')
})

test('the page behind cannot scroll, and says so', async ({ page }) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('scroll-state')).toHaveText('free')

  await page.getByTestId('open-modal').click()
  await expect(page.getByTestId('scroll-state')).toHaveText('locked')

  // Published on the body, so it can be asserted rather than inferred from a
  // scroll that silently did nothing.
  await expect(page.locator('body')).toHaveAttribute('data-scroll-locked', 'true')

  await page.getByTestId('modal-cancel').click()
  await expect(page.locator('body')).not.toHaveAttribute('data-scroll-locked', 'true')
})

test('tab cannot leave the dialog', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('open-modal').click()
  await expect(page.getByTestId('modal-confirm')).toBeFocused()

  await page.keyboard.press('Tab')
  await expect(page.getByTestId('modal-cancel')).toBeFocused()

  // A fixed number of tabs would walk out of any normal page. Not this one.
  await page.keyboard.press('Tab')
  await expect(page.getByTestId('modal-confirm')).toBeFocused()
})

test('the dialog reports how it closed, because afterwards there is nothing to ask', async ({
  page,
}) => {
  await page.goto(PAGE)

  for (const [action, expected] of [
    ['modal-confirm', 'confirmed'],
    ['modal-cancel', 'cancelled'],
  ] as const) {
    await page.getByTestId('open-modal').click()
    await page.getByTestId(action).click()
    await expect(page.getByTestId('modal-outcome')).toHaveText(expected)
  }

  await page.getByTestId('open-modal').click()
  await page.keyboard.press('Escape')
  await expect(page.getByTestId('modal-outcome')).toHaveText('escape')
})

test('clicking the overlay itself closes the dialog', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('open-modal').click()

  await page.getByTestId('modal-overlay').click({ position: { x: 8, y: 8 } })
  await expect(page.getByTestId('modal-outcome')).toHaveText('overlay')
})
