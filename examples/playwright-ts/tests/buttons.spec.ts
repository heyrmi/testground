import { expect, test } from './fixtures'

const PAGE = '/classic/buttons'

test('two submits share a name and are told apart by their value', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('submit-save').click()
  await expect(page.getByTestId('result-action')).toHaveText('save')

  await page.getByTestId('submit-publish').click()
  await expect(page.getByTestId('result-action')).toHaveText('publish')
})

test('the anchor is a link, not a button, and posts nothing', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('submit-save').click()
  await expect(page.getByTestId('submission-count')).toHaveText('1')

  // Locating by role is what separates these two: one is a link.
  await expect(page.getByRole('link', { name: 'Looks like a button' })).toBeVisible()
  await page.getByTestId('link-button').click()

  await expect(page).toHaveURL(new RegExp(`${PAGE}$`))
  await expect(page.getByTestId('submission-count')).toHaveText('1')
})

test('the inert button does nothing, which looks exactly like a failed click', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('inert').click()

  await expect(page.getByTestId('no-submission')).toBeVisible()
  await expect(page).toHaveURL(new RegExp(`${PAGE}$`))
})

test('the disabled button is declared disabled rather than clicked hopefully', async ({ page }) => {
  await page.goto(PAGE)

  await expect(page.getByTestId('disabled')).toBeDisabled()
  await expect(page.getByTestId('no-submission')).toBeVisible()
})

test('reset clears the field without posting', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('field-draft').fill('some work in progress')
  await page.getByTestId('reset').click()

  await expect(page.getByTestId('field-draft')).toHaveValue('')
  await expect(page.getByTestId('no-submission')).toBeVisible()
})

test('a click landing on a child element still activates the button', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('submit-icon').locator('span').last().click()

  await expect(page.getByTestId('result-action')).toHaveText('icon')
})
