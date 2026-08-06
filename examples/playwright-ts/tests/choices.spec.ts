import { expect, test } from './fixtures'

const PAGE = '/classic/choices'

test('one checkbox starts checked, and the group posts every checked value', async ({ page }) => {
  await page.goto(PAGE)

  await expect(page.getByTestId('topping-cheese')).toBeChecked()
  await expect(page.getByTestId('topping-olives')).not.toBeChecked()

  await page.getByTestId('topping-olives').check()
  await page.getByTestId('submit').click()

  // Three inputs share one name, so the request carries a repeated field.
  await expect(page.getByTestId('result-toppings')).toHaveText('cheese, olives')
})

test('an unchecked box posts nothing at all, rather than a false value', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('submit').click()

  await expect(page.getByTestId('result-newsletter')).toBeEmpty()
})

test('selecting one radio clears the rest of its group', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('delivery-standard').check()
  await page.getByTestId('delivery-express').check()

  await expect(page.getByTestId('delivery-standard')).not.toBeChecked()
  await page.getByTestId('submit').click()
  await expect(page.getByTestId('result-delivery')).toHaveText('express')
})

test('a multi-select takes several values through the select API', async ({ page }) => {
  await page.goto(PAGE)

  // Clicking twice would not do this; the second click replaces the first.
  await page.getByTestId('field-languages').selectOption(['ja', 'sw'])
  await page.getByTestId('submit').click()

  await expect(page.getByTestId('result-languages')).toHaveText('ja, sw')
})

test('a disabled option is declared disabled rather than discovered by clicking', async ({
  page,
}) => {
  await page.goto(PAGE)
  const iceland = page.getByTestId('field-country').locator('option[value="is"]')

  await expect(iceland).toBeDisabled()
  await expect(page.getByTestId('field-country').locator('optgroup')).toHaveCount(2)
})

test('options inside an optgroup are still selectable by value', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('field-country').selectOption('jp')
  await page.getByTestId('submit').click()

  await expect(page.getByTestId('result-country')).toHaveText('jp')
})
