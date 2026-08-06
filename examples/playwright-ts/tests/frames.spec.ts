import { expect, test } from './fixtures'

const PAGE = '/classic/frames'

test('a page-level locator does not look inside a frame', async ({ page }) => {
  await page.goto(PAGE)

  // The element exists. It is just not in this document.
  await expect(page.getByTestId('embedded-target')).toHaveCount(0)
  await expect(page.getByTestId('deepest-target')).toHaveCount(0)
  await expect(page.getByTestId('cross-origin-target')).toHaveCount(0)

  await expect(
    page.frameLocator('[data-testid="frame-same-origin"]').getByTestId('embedded-target'),
  ).toBeVisible()
})

test('the nested chain is descended one frame at a time', async ({ page }) => {
  await page.goto(PAGE)

  const deepest = page
    .frameLocator('[data-testid="frame-nested"]')
    .frameLocator('[data-testid="frame-level-2"]')
    .frameLocator('[data-testid="frame-level-3"]')
    .getByTestId('deepest-target')

  await expect(deepest).toBeVisible()
  await expect(deepest).toContainText('Three frames down')
})

test('page script cannot read across the origin boundary', async ({ page }) => {
  await page.goto(PAGE)

  const reach = await page.evaluate(() => {
    const same = document.querySelector<HTMLIFrameElement>('[data-testid="frame-same-origin"]')!
    const cross = document.querySelector<HTMLIFrameElement>('[data-testid="frame-cross-origin"]')!

    const read = (frame: HTMLIFrameElement) => {
      try {
        return frame.contentWindow!.document.body !== null ? 'readable' : 'empty'
      } catch (error) {
        return `threw: ${(error as Error).name}`
      }
    }
    return { same: read(same), cross: read(cross), crossDocument: cross.contentDocument }
  })

  expect(reach.same).toBe('readable')
  expect(reach.cross, 'the same-origin policy is not a wait problem').toContain('threw')
  expect(reach.crossDocument).toBeNull()
})

test('the framework enters the cross-origin frame that page script cannot', async ({ page }) => {
  await page.goto(PAGE)

  // Playwright drives the browser rather than running inside the page, so the
  // boundary that stops document.querySelector does not stop it.
  await expect(
    page.frameLocator('[data-testid="frame-cross-origin"]').getByTestId('cross-origin-target'),
  ).toBeVisible()
})

test('both origins resolve to the same session, because cookies ignore the port', async ({
  page,
  sessionId,
}) => {
  await page.goto(PAGE)

  await expect(page.getByTestId('parent-session')).toHaveText(sessionId)
  await expect(
    page.frameLocator('[data-testid="frame-cross-origin"]').getByTestId('cross-origin-session'),
  ).toHaveText(sessionId)
})

test('a srcdoc frame has content but no URL', async ({ page }) => {
  await page.goto(PAGE)

  const frame = page.locator('[data-testid="frame-srcdoc"]')
  await expect(frame).not.toHaveAttribute('src', /./)
  await expect(
    page.frameLocator('[data-testid="frame-srcdoc"]').getByTestId('srcdoc-target'),
  ).toContainText('arrived as an attribute')
})

test('the second origin reports the session directly', async ({ page, sessionId, baseURL }) => {
  const crossOrigin = baseURL!.replace(/:\d+$/, ':7374')
  const body = await (await page.request.get(`${crossOrigin}/whoami`)).json()

  expect(body).toMatchObject({ session: sessionId, origin: 'cross' })
})
