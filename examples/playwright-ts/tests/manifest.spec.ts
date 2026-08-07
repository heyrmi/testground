import { expect, test } from './fixtures'
import type { Page } from '@playwright/test'

interface Selector {
  testId: string
  transient?: boolean
  /** Iframe test ids to descend before looking, outermost first. */
  frame?: string[]
  /**
   * A repeated set this selector represents, where <n> stands for a run of
   * digits and <s> for a run of word characters. A page with six identical
   * boxes declares "otp-<n>" once rather than listing every index.
   */
  family?: string
}

/**
 * The same expansion the server does, so both sides agree on what a family
 * covers. Everything outside a placeholder is matched literally: a pattern
 * must not become a wildcard through a character that happens to mean
 * something to a regular expression.
 */
function familyPattern(family: string): RegExp {
  const expanded = family
    .split(/(<[ns]>)/)
    .map((part) =>
      part === '<n>'
        ? '[0-9]+'
        : part === '<s>'
          ? '[A-Za-z0-9_-]+'
          : part.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'),
    )
    .join('')
  return new RegExp(`^${expanded}$`)
}

/** Whether a rendered test id is a selector, or a member of the family it represents. */
function covers(selector: Selector, testId: string): boolean {
  if (selector.testId === testId) return true
  return selector.family !== undefined && familyPattern(selector.family).test(testId)
}

interface Challenge {
  id: string
  title: string
  url: string
  summary: string
  hint: string
  selectors: Selector[]
  hostileLocators?: boolean
}

/**
 * Test ids the shared chrome puts on every challenge page in a zone. They
 * describe the wrapper rather than the challenge, so no manifest entry claims
 * them, and the reverse check would otherwise report the same handful of ids
 * against every page in the suite.
 */
const chromeTestIds = new Set([
  'meta-version',
  'meta-seed',
  'meta-session',
  'back-to-index',
  'back-to-zone',
  'challenge-tier',
  'challenge-zone',
  'challenge-title',
  'challenge-panel',
  'challenge-hint',
  'stage',
])

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

test('every test id the page renders is declared by the challenge', async ({ page }) => {
  const manifest = await (await page.request.get('/api/challenges')).json()

  for (const challenge of manifest.challenges as Challenge[]) {
    // A hostile-locators page withholds test ids because finding the elements
    // without them is the exercise, and the registry lets it declare none at
    // all. There is nothing here to hold its markup to.
    if (challenge.hostileLocators) continue

    await test.step(challenge.id, async () => {
      await page.goto(challenge.url)

      // The SPA zone renders nothing at all until the manifest it fetches
      // arrives, so reading the DOM any earlier finds an empty document and
      // passes without having checked anything. Soft, so a page that never
      // renders is reported rather than abandoning the remaining challenges.
      await expect
        .soft(page.getByTestId('challenge-panel'), `${challenge.id} never rendered its panel`)
        .toBeAttached()

      // Only the top document: a locator does not cross a frame boundary and
      // querySelectorAll does not enter a shadow root, so an id in either is
      // out of reach here rather than evidence of a page that under-declares.
      const rendered = await page.evaluate(() =>
        [...document.querySelectorAll('[data-testid]')].map((el) =>
          el.getAttribute('data-testid'),
        ),
      )

      // Compared against every declaration, transient ones included. A
      // transient selector that is present on load is the previous test's to
      // report, and saying it twice in two different voices helps nobody.
      const undeclared = [...new Set(rendered)]
        .filter(
          (id): id is string =>
            id !== null &&
            !chromeTestIds.has(id) &&
            !challenge.selectors.some((s) => covers(s, id)),
        )
        .sort()

      // The direction the presence check above cannot see: an element the page
      // grew without anyone declaring it is a contract that exists in the
      // markup and nowhere a reader would look for it.
      expect
        .soft(undeclared, `${challenge.id} renders test ids its manifest entry does not declare`)
        .toEqual([])
    })
  }
})

test('an unknown challenge id is a clean 404', async ({ page }) => {
  const response = await page.request.get('/api/challenges/no-such-challenge')

  expect(response.status()).toBe(404)
  expect(await response.json()).toMatchObject({ status: 404 })
})
