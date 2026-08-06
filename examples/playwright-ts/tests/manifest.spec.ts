import { expect, test } from './fixtures'
import type { Page } from '@playwright/test'

interface Selector {
  testId: string
  transient?: boolean
  /** Iframe test ids to descend before looking, outermost first. */
  frame?: string[]
}

interface Challenge {
  id: string
  title: string
  url: string
  summary: string
  hint: string
  selectors: Selector[]
}

/**
 * A locator does not cross a frame boundary on its own, so a declared selector
 * that lives inside one has to be reached by descending into it. The manifest
 * says which frames, and in what order.
 */
function locate(page: Page, selector: Selector) {
  if (!selector.frame?.length) return page.getByTestId(selector.testId)

  let frame = page.frameLocator(`[data-testid="${selector.frame[0]}"]`)
  for (const nested of selector.frame.slice(1)) {
    frame = frame.frameLocator(`[data-testid="${nested}"]`)
  }
  return frame.getByTestId(selector.testId)
}

/**
 * The manifest is the contract. These tests read it rather than hard-coding a
 * list, so they keep covering the promise as challenges are added.
 */
test('the manifest describes the caller own session', async ({ page, sessionId }) => {
  const manifest = await (await page.request.get('/api/challenges')).json()

  expect(manifest.session).toBe(sessionId)
  expect(manifest.seed).toBe(42)
  expect(manifest.count).toBe(manifest.challenges.length)
  expect(manifest.count).toBeGreaterThan(0)
})

test('every challenge page is reachable and describes itself', async ({ page }) => {
  const manifest = await (await page.request.get('/api/challenges')).json()

  for (const challenge of manifest.challenges as Challenge[]) {
    await test.step(challenge.id, async () => {
      const response = await page.goto(challenge.url)
      expect(response?.status(), `${challenge.url} should serve a page`).toBe(200)

      await expect(page.getByTestId('challenge-title')).toHaveText(challenge.title)
      await expect(page.getByTestId('challenge-panel')).toContainText(challenge.summary)

      // The hint is a disclosure: closed by default so it cannot spoil the
      // exercise, but always present.
      const hint = page.getByTestId('challenge-hint')
      await expect(hint).toBeVisible()
      await expect(hint.locator('p')).toBeHidden()
      await hint.locator('summary').click()
      await expect(hint.locator('p')).toHaveText(challenge.hint)
    })
  }
})

test('every declared selector is actually in the page it describes', async ({ page }) => {
  const manifest = await (await page.request.get('/api/challenges')).json()

  for (const challenge of manifest.challenges as Challenge[]) {
    await test.step(challenge.id, async () => {
      await page.goto(challenge.url)

      // Checked against the live DOM, not against the selector table the page
      // prints. Reading the table would only prove the manifest agrees with
      // itself, which is exactly the drift this is supposed to catch.
      // Soft, so one run reports every mismatch rather than stopping at the
      // first and hiding the rest behind another cycle.
      for (const selector of challenge.selectors.filter((s) => !s.transient)) {
        await expect
          .soft(
            locate(page, selector).first(),
            `${challenge.id} declares ${selector.testId} but the page has no such element`,
          )
          .toBeAttached()
      }
    })
  }
})

test('selectors that only exist mid-interaction say so', async ({ page }) => {
  const manifest = await (await page.request.get('/api/challenges')).json()

  for (const challenge of manifest.challenges as Challenge[]) {
    await test.step(challenge.id, async () => {
      await page.goto(challenge.url)

      // The transient flag is an exemption from the check above, so it has to
      // be earned: an element present on load must not claim it.
      for (const selector of challenge.selectors.filter((s) => s.transient)) {
        await expect
          .soft(
            locate(page, selector),
            `${challenge.id} marks ${selector.testId} transient, but it is present on load`,
          )
          .toHaveCount(0)
      }
    })
  }
})

test('an unknown challenge id is a clean 404', async ({ page }) => {
  const response = await page.request.get('/api/challenges/no-such-challenge')

  expect(response.status()).toBe(404)
  expect(await response.json()).toMatchObject({ status: 404 })
})
