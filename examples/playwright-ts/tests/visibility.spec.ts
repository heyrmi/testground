import { expect, test } from './fixtures'

const PAGE = '/legacy/visibility'

test('the control is plainly clickable', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('btn-normal').click()
  await expect(page.getByTestId('clicked')).toHaveText('normal')
})

test('an invisible button can still be perfectly clickable', async ({ page }) => {
  await page.goto(PAGE)

  // opacity:0 removes it from sight and from nothing else. The click lands,
  // the handler runs, and no user could have done it.
  await expect(page.getByTestId('btn-opacity-zero')).toBeVisible()
  await page.getByTestId('btn-opacity-zero').click()
  await expect(page.getByTestId('clicked')).toHaveText('opacity-zero')
})

test('display:none and visibility:hidden are both hidden, and differ in layout', async ({
  page,
}) => {
  await page.goto(PAGE)

  await expect(page.getByTestId('btn-display-none')).toBeHidden()
  await expect(page.getByTestId('btn-visibility-hidden')).toBeHidden()

  // One occupies space, the other does not, which is the difference that
  // matters when something else is positioned relative to it.
  expect(await page.getByTestId('btn-display-none').boundingBox()).toBeNull()
  expect(await page.getByTestId('btn-visibility-hidden').boundingBox()).not.toBeNull()
})

test('an off-screen button is laid out, sized, and unreachable by a person', async ({ page }) => {
  await page.goto(PAGE)
  const box = await page.getByTestId('btn-offscreen').boundingBox()

  expect(box, 'it has a real size, which is why size checks pass').not.toBeNull()
  expect(box!.x, 'and it is nowhere near the viewport').toBeLessThan(0)
})

test('a covered button passes every visibility check and the click hits the overlay', async ({
  page,
}) => {
  await page.goto(PAGE)
  const covered = page.getByTestId('btn-covered')

  await expect(covered).toBeVisible()
  await expect(covered).toBeEnabled()

  // The failure names an element the test never mentioned. Believe it: this
  // is a real bug, and users are hitting the same overlay.
  const error = await covered.click({ timeout: 2000 }).catch((e: Error) => e.message)
  expect(error).toContain('intercepts pointer events')
  await expect(page.getByTestId('clicked')).toHaveText('none')
})

test('a transitioning button is in place before it is in place', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('reveal').click()

  // Playwright waits for the element to stop moving before clicking, which is
  // exactly the right behaviour and not every tool does it.
  await page.getByTestId('btn-fading').click()
  await expect(page.getByTestId('clicked')).toHaveText('fading')
})
