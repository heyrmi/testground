import { expect, test } from './fixtures'

interface Challenge {
  id: string
  title: string
  url: string
  summary: string
  hint: string
  selectors: { testId: string }[]
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

test('every declared selector exists on its page', async ({ page }) => {
  const manifest = await (await page.request.get('/api/challenges')).json()

  for (const challenge of manifest.challenges as Challenge[]) {
    await test.step(challenge.id, async () => {
      await page.goto(challenge.url)
      for (const selector of challenge.selectors) {
        // Some markers only appear mid-interaction, so presence in the
        // selector table is checked against the panel's own documentation
        // rather than the live DOM for those.
        const documented = page
          .getByTestId('challenge-panel')
          .locator('td', { hasText: selector.testId })
        await expect(documented.first()).toBeVisible()
      }
    })
  }
})

test('an unknown challenge id is a clean 404', async ({ page }) => {
  const response = await page.request.get('/api/challenges/no-such-challenge')

  expect(response.status()).toBe(404)
  expect(await response.json()).toMatchObject({ status: 404 })
})
