import { expect, test } from './fixtures'

const PAGE = '/app/visual-regression'

test('a capture is the same on every run', async ({ page }) => {
  await page.goto(PAGE)
  const block = page.getByTestId('reference')
  await expect(page.getByTestId('freeze-state')).toHaveText('frozen')

  const first = await block.screenshot({ mask: [page.getByTestId('volatile')] })
  await page.reload()
  const second = await block.screenshot({ mask: [page.getByTestId('volatile')] })

  expect(second).toEqual(first)
})

// The check that matters more than any number of green runs: prove the
// comparison is capable of failing before trusting it when it does not.
test('and it is not the same when one pixel changes', async ({ page }) => {
  await page.goto(PAGE)
  const baseline = await page
    .getByTestId('reference')
    .screenshot({ mask: [page.getByTestId('volatile')] })

  await page.goto(`${PAGE}?diff=1`)
  await expect(page.getByTestId('diff-state')).toHaveText('on')
  const changed = await page
    .getByTestId('reference')
    .screenshot({ mask: [page.getByTestId('volatile')] })

  expect(changed, 'a comparison that passes both ways is comparing nothing').not.toEqual(baseline)
})

test('the volatile region is marked rather than hidden', async ({ page }) => {
  await page.goto(PAGE)
  const volatile = page.getByTestId('volatile')

  await expect(volatile).toHaveAttribute('data-vr-mask', 'true')
  await expect(volatile).toBeVisible()

  // It really does change, which is why an unmasked capture is unstable.
  const first = await volatile.textContent()
  await expect(volatile).not.toHaveText(first!)
})

test('an unmasked capture of a changing region is unstable', async ({ page }) => {
  await page.goto(PAGE)
  const block = page.getByTestId('reference')

  const first = await block.screenshot()
  await page.waitForTimeout(250)
  const second = await block.screenshot()

  // Nothing regressed. The clock moved, which is exactly the noise that
  // drives people to raise the tolerance until nothing fails.
  expect(second).not.toEqual(first)
})

test('the animation is frozen unless asked otherwise', async ({ page }) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('freeze-state')).toHaveText('frozen')

  await page.goto(`${PAGE}?freeze=0`)
  await expect(page.getByTestId('freeze-state')).toHaveText('running')

  const animated = await page
    .getByTestId('spinner')
    .evaluate((el) => getComputedStyle(el).animationName)
  expect(animated).toBe('vr-spin')
})
