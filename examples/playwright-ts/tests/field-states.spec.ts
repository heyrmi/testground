import { expect, test } from './fixtures'

const PAGE = '/classic/field-states'

test('the three uneditable-looking fields behave completely differently', async ({ page }) => {
  await page.goto(PAGE)

  await expect(page.getByTestId('field-readonly')).toHaveAttribute('readonly', '')
  await expect(page.getByTestId('field-disabled')).toBeDisabled()
  await expect(page.getByTestId('field-aria-disabled')).toHaveAttribute('aria-disabled', 'true')
})

test('the tooling and the browser disagree about aria-disabled', async ({ page }) => {
  await page.goto(PAGE)
  const ariaDisabled = page.getByTestId('field-aria-disabled')

  // Playwright reads aria-disabled as disabled and will not type into it.
  await expect(ariaDisabled).toBeDisabled()
  await expect(ariaDisabled.fill('typed', { timeout: 1500 })).rejects.toThrow()

  // The browser disagrees. aria-disabled is an announcement and nothing more,
  // so the field is fully editable by anyone using a keyboard.
  const editable = await ariaDisabled.evaluate((el: HTMLInputElement) => {
    el.focus()
    el.value = 'typed by a person'
    return document.activeElement === el && !el.disabled && !el.readOnly
  })
  expect(editable, 'the browser lets a real user edit this field').toBe(true)

  // Which makes it a control tests cannot reach and users can. The failure to
  // interact is the finding; the markup is the thing to fix.
  await page.getByTestId('submit').click()
  await expect(page.getByTestId('result-aria-disabled')).toHaveText('typed by a person')
})

test('only the disabled field is absent from the request', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('submit').click()

  const arrived = await page.getByTestId('result-arrived').textContent()

  expect(arrived).toContain('readonly')
  expect(arrived).toContain('ariaDisabled')
  expect(arrived, 'a disabled control is not part of the form').not.toContain('locked')

  await expect(page.getByTestId('result-readonly')).toHaveText('posted anyway')
  await expect(page.getByTestId('result-locked')).toBeEmpty()
})

test('three of the four fields can be found by their accessible name', async ({ page }) => {
  await page.goto(PAGE)

  await expect(page.getByLabel('Labelled with for')).toBeVisible()
  await expect(page.getByLabel('Labelled by wrapping')).toBeVisible()
  await expect(page.getByLabel('Labelled with aria-label')).toBeVisible()
})

test('the placeholder-only field has no accessible name', async ({ page }) => {
  await page.goto(PAGE)

  // A placeholder is not a label. Finding this field needs a different
  // locator, and needing one is the defect rather than the workaround.
  await expect(page.getByLabel('Placeholder pretending to be a label')).toHaveCount(0)

  const named = await page
    .getByTestId('field-unlabelled')
    .evaluate((el: HTMLInputElement) => Boolean(el.getAttribute('aria-label')) || el.labels!.length > 0)
  expect(named, 'no label element and no aria-label leaves it nameless').toBe(false)
})
