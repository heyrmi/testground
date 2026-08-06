import { expect, test } from './fixtures'

const PAGE = '/legacy/dialog-element'

test('a modal makes the background inert while leaving it visible and enabled', async ({
  page,
}) => {
  await page.goto(PAGE)
  await page.getByTestId('open-modal').click()

  const background = page.getByTestId('background-button')
  await expect(background).toBeVisible()
  await expect(background).toBeEnabled()

  // Every precondition passes and the click is still refused. That is the
  // feature, not a flake.
  const error = await background.click({ timeout: 2000 }).catch((e: Error) => e.message)
  expect(error).toBeTruthy()
  await expect(page.getByTestId('background-clicks')).toHaveText('0')
})

test('a non-modal dialog leaves the page working', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('open-modeless').click()

  await expect(page.getByTestId('modeless-dialog')).toBeVisible()
  await page.getByTestId('background-button').click()
  await expect(page.getByTestId('background-clicks')).toHaveText('1')
})

test('the return value is the only record of how a modal closed', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('open-modal').click()
  await page.getByTestId('confirm-dialog').click()
  await expect(page.getByTestId('modal-dialog')).toBeHidden()
  await expect(page.getByTestId('dialog-return')).toHaveText('confirmed')

  await page.getByTestId('open-modal').click()
  await page.getByTestId('cancel-dialog').click()
  await expect(page.getByTestId('dialog-return')).toHaveText('cancelled')
})

test('escape closes a modal and not a non-modal one', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('open-modal').click()
  await page.keyboard.press('Escape')
  await expect(page.getByTestId('modal-dialog')).toBeHidden()

  await page.getByTestId('open-modeless').click()
  await page.keyboard.press('Escape')
  await expect(page.getByTestId('modeless-dialog')).toBeVisible()
})
