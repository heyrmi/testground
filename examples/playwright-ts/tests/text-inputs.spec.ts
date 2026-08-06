import { expect, test } from './fixtures'

const PAGE = '/classic/text-inputs'

const filled = {
  text: 'hello',
  password: 'hunter22',
  email: 'name@example.test',
  number: '25',
  tel: '+44 20 7946 0000',
  url: 'https://example.test',
  search: 'needle',
  comment: 'a comment',
}

async function fillEverything(page: import('@playwright/test').Page) {
  for (const [field, value] of Object.entries(filled)) {
    await page.getByTestId(`field-${field}`).fill(value)
  }
}

test('the form posts and the server echoes what it received', async ({ page }) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('no-submission')).toBeVisible()

  await fillEverything(page)
  await page.getByTestId('submit').click()

  await expect(page.getByTestId('result')).toBeVisible()
  await expect(page.getByTestId('result-text')).toHaveText(filled.text)
  await expect(page.getByTestId('result-email')).toHaveText(filled.email)
  await expect(page.getByTestId('result-comment')).toHaveText(filled.comment)
  await expect(page.getByTestId('submission-count')).toHaveText('1')
})

test('the password is never reflected back into the page', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('field-password').fill(filled.password)
  await page.getByTestId('submit').click()

  await expect(page.getByTestId('result-password')).toHaveText('8 characters, not echoed')
  await expect(page.locator('body')).not.toContainText(filled.password)
})

test('the post answers 303 and the browser follows it', async ({ page }) => {
  await page.goto(PAGE)

  const post = page.waitForResponse(
    (res) => res.request().method() === 'POST' && res.url().endsWith(PAGE),
  )
  await page.getByTestId('submit').click()

  // Waiting on the POST is waiting on the wrong request: its body is a
  // redirect, not the page. The page arrives on the GET that follows.
  expect((await post).status()).toBe(303)
  await expect(page).toHaveURL(new RegExp(`${PAGE}$`))
  await expect(page.getByTestId('result')).toBeVisible()
})

test('element handles held across the submit go stale; locators do not', async ({ page }) => {
  await page.goto(PAGE)

  const handle = await page.$('[data-testid="field-text"]')
  const locator = page.getByTestId('field-text')

  await page.getByTestId('submit').click()
  await expect(page.getByTestId('result')).toBeVisible()

  // The handle points at an element from the previous document.
  await expect(handle!.fill('again')).rejects.toThrow()

  // The locator is a query, re-resolved on use, so it simply works.
  await locator.fill('again')
  await expect(locator).toHaveValue('again')
})

test('Enter in a text field submits without touching the button', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('field-text').fill('submitted with the keyboard')
  await page.getByTestId('field-text').press('Enter')

  await expect(page.getByTestId('result-text')).toHaveText('submitted with the keyboard')
})

test('submissions accumulate per session and can be discarded', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('submit').click()
  await page.getByTestId('submit').click()
  await expect(page.getByTestId('submission-count')).toHaveText('2')

  await page.getByTestId('clear').click()
  await expect(page.getByTestId('no-submission')).toBeVisible()
  await expect(page.getByTestId('result')).toHaveCount(0)
})

test('the number field carries its own constraints', async ({ page }) => {
  await page.goto(PAGE)
  const number = page.getByTestId('field-number')

  await expect(number).toHaveAttribute('min', '0')
  await expect(number).toHaveAttribute('max', '100')
  await expect(number).toHaveAttribute('step', '5')
})
