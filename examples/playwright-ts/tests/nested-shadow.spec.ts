import { expect, test } from './fixtures'

test('Playwright pierces open shadow roots for you', async ({ page }) => {
  await page.goto('/wc/nested-shadow')

  // Three roots deep and located as if it were in the light DOM. Tools that
  // do not do this need the explicit walk in the next test.
  await expect(page.getByTestId('inner-input')).toBeVisible()
  await expect(page.getByTestId('inner-submit')).toBeVisible()
})

test('the DOM API itself sees nothing', async ({ page }) => {
  await page.goto('/wc/nested-shadow')

  const found = await page.evaluate(
    () => document.querySelectorAll('[data-testid="inner-input"]').length,
  )
  expect(found, 'querySelector stops at the first shadow boundary').toBe(0)
})

test('walking the roots explicitly works everywhere', async ({ page }) => {
  await page.goto('/wc/nested-shadow')

  const depth = await page.evaluate(() => {
    const outer = document.querySelector('[data-testid="shadow-host"]')!
    const middle = outer.shadowRoot!.querySelector('pg-shadow-middle')!
    const inner = middle.shadowRoot!.querySelector('pg-shadow-inner')!
    return {
      allOpen: [outer, middle, inner].every((host) => host.shadowRoot !== null),
      reachedInput: inner.shadowRoot!.querySelector('[data-testid="inner-input"]') !== null,
    }
  })

  expect(depth.allOpen).toBe(true)
  expect(depth.reachedInput).toBe(true)
})

test('light DOM projected through a slot stays in the light DOM', async ({ page }) => {
  await page.goto('/wc/nested-shadow')

  const label = page.getByTestId('slotted-label')
  await expect(label).toBeVisible()

  const isLightDom = await label.evaluate((el) => el.getRootNode() === document)
  expect(isLightDom, 'a slotted node is rendered inside a root it does not belong to').toBe(true)
})

test('a composed event escapes every boundary', async ({ page }) => {
  await page.goto('/wc/nested-shadow')

  await page.getByTestId('inner-input').fill('crossed all three')
  await page.getByTestId('inner-submit').click()

  // Asserting on the light-DOM echo needs no traversal at all, which is the
  // easier contract to write tests against when a component offers it.
  await expect(page.getByTestId('shadow-echo')).toHaveText('crossed all three')
  await expect(page.getByTestId('shadow-submit-count')).toHaveText('1')
  await expect(page.getByTestId('inner-echo')).toHaveText('crossed all three')
})
