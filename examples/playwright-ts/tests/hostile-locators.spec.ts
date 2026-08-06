import { expect, test } from './fixtures'

const PAGE = '/app/hostile-locators'

test('a class-name selector is correct until the next deploy', async ({ page }) => {
  await page.goto(PAGE)

  const before = (await page.getByTestId('sample-class').textContent())!
  await expect(page.locator(`.${before}`)).toBeVisible()

  await page.getByTestId('rebuild').click()

  // No code changed. The selector is simply gone, and the failure arrives
  // with nothing anyone will connect to it.
  await expect(page.locator(`.${before}`)).toHaveCount(0)
  expect(await page.getByTestId('sample-class').textContent()).not.toBe(before)
})

test('two elements share one id, and the lookups disagree', async ({ page }) => {
  await page.goto(PAGE)

  const counts = await page.evaluate(() => ({
    byId: document.getElementById('duplicate') ? 1 : 0,
    bySelector: document.querySelectorAll('#duplicate').length,
  }))

  // Invalid HTML the browser accepts in silence.
  expect(counts.byId).toBe(1)
  expect(counts.bySelector).toBe(2)
})

test('text split across nodes defeats an exact match', async ({ page }) => {
  await page.goto(PAGE)
  const split = page.getByTestId('split-text')

  // The user sees one sentence. So does a contains match.
  await expect(split).toContainText('Order number 4417')

  // An exact match on any single node does not, because no node holds it.
  const nodeTexts = await split.locator('span').allTextContents()
  expect(nodeTexts).toEqual(['Order', 'number', '4417'])
})

test('invisible characters sit between the words', async ({ page }) => {
  await page.goto(PAGE)
  const raw = (await page.getByTestId('zero-width').textContent())!

  expect(raw).toContain('​')
  expect(raw, 'the obvious assertion fails against text that looks right').not.toBe('Total: 42')

  // Normalising before comparing is what makes this tractable.
  expect(raw.replace(/​/g, '')).toBe('Total: 42')
})

test('what a user can read and what a test can read have diverged', async ({ page }) => {
  await page.goto(PAGE)
  const truncated = page.getByTestId('truncated')

  const { dom, rendered } = await truncated.evaluate((el) => ({
    dom: el.textContent ?? '',
    rendered: el.scrollWidth > el.clientWidth,
  }))

  expect(rendered, 'CSS is drawing an ellipsis').toBe(true)
  expect(dom).toContain('longer than the box that is drawing it')
})

test('identical twins can only be told apart by position, which is the finding', async ({
  page,
}) => {
  await page.goto(PAGE)
  const twins = page.getByRole('button', { name: 'Continue' })

  await expect(twins).toHaveCount(2)

  // Nothing distinguishes them but order. Using that works, and is a note to
  // go and fix the markup rather than a technique to be pleased with.
  await twins.nth(1).click()
  await expect(page.getByTestId('chosen')).toHaveText('twin-right')
})

test('the div soup has nothing to locate by but its text', async ({ page }) => {
  await page.goto(PAGE)

  const depth = await page.evaluate(() => {
    // The innermost one: every ancestor has the same textContent, so matching
    // on text alone finds the outermost wrapper rather than the target.
    const leaf = [...document.querySelectorAll('div')].find(
      (el) => el.childElementCount === 0 && el.textContent === 'Approve',
    )

    let levels = 0
    for (let node = leaf?.parentElement; node?.tagName === 'DIV'; node = node.parentElement) {
      levels += 1
    }
    return levels
  })
  expect(depth, 'twelve wrappers, none of them meaning anything').toBeGreaterThanOrEqual(11)

  await page.getByText('Approve', { exact: true }).click()
  await expect(page.getByTestId('chosen')).toHaveText('div-soup')
})
