import { expect, test } from './fixtures'

const PAGE = '/wc/closed-shadow'

test('a closed root is not reachable by anything', async ({ page }) => {
  await page.goto(PAGE)

  const closed = await page.evaluate(
    () => (document.querySelector('[data-testid="closed-host"]') as HTMLElement).shadowRoot,
  )
  expect(closed, 'there is no root to traverse into').toBeNull()

  // Not the DOM API, and not the framework's piercing either. There is no
  // technique here, only an absence.
  await expect(page.getByTestId('closed-input')).toHaveCount(0)
  await expect(page.getByTestId('closed-submit')).toHaveCount(0)
})

test('an open root beside it is reachable, so the page is not simply broken', async ({ page }) => {
  await page.goto(PAGE)

  await expect(page.getByTestId('part-button')).toBeVisible()
})

test('the property is the supported way in', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('closed-read').click()
  await expect(page.getByTestId('closed-value')).toHaveText('(empty)')

  await page.getByTestId('closed-write').click()
  await expect(page.getByTestId('closed-value')).toHaveText('written through the property')
})

test('a test can drive the component through its property directly', async ({ page }) => {
  await page.goto(PAGE)

  // No traversal, because there is nothing to traverse. This is the whole
  // supported surface, and a component with a closed root and no such surface
  // is one the test has correctly found a defect in.
  await page.getByTestId('closed-host').evaluate((el: HTMLElement & { value: string }) => {
    el.value = 'set from the test'
  })

  await page.getByTestId('closed-read').click()
  await expect(page.getByTestId('closed-value')).toHaveText('set from the test')
})

test('a composed event still crosses the closed boundary', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('closed-host').evaluate((el: HTMLElement & { value: string }) => {
    el.value = 'escaping'
    el.dispatchEvent(
      new CustomEvent('pg-closed-submit', {
        detail: { value: el.value },
        bubbles: true,
        composed: true,
      }),
    )
  })

  await expect(page.getByTestId('closed-escaped')).toHaveText('escaping')
})

test('the late element is present all along and does nothing', async ({ page }) => {
  await page.goto(PAGE)
  const host = page.getByTestId('late-host')

  // Waiting for the element finds it immediately, which is the trap: it is
  // there, it has no shadow root, and it looks like a component that failed.
  await expect(host).toBeAttached()
  await expect(host).not.toHaveAttribute('data-upgraded', 'true')

  // Waiting for the marker is what actually waits for the component.
  await expect(host).toHaveAttribute('data-upgraded', 'true', { timeout: 5000 })
  await expect(page.getByTestId('late-content')).toBeVisible()
})

test('::part styling reaches in where selectors cannot', async ({ page }) => {
  await page.goto(PAGE)

  const weight = await page
    .getByTestId('part-button')
    .evaluate((el) => getComputedStyle(el).fontWeight)

  // The page stylesheet set this through ::part, without any access to the
  // element's internals.
  expect(weight).toBe('600')
})
