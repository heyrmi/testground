import { test as base, type BrowserContext } from '@playwright/test'

/**
 * Every test gets its own isolated copy of the playground.
 *
 * The server keys state on the X-Playground-Session header, so pinning a
 * unique id per test is what lets this suite run fully parallel while tests
 * mutate server state. Without it, two workers toggling the same task would
 * fight over one list.
 */
export const test = base.extend<{ sessionId: string; context: BrowserContext }>({
  sessionId: async ({}, use, testInfo) => {
    const safe = testInfo.testId.replace(/[^A-Za-z0-9_-]/g, '')
    await use(`pw-${testInfo.workerIndex}-${safe}`)
  },

  context: async ({ browser, sessionId }, use) => {
    const context = await browser.newContext({
      extraHTTPHeaders: { 'X-Playground-Session': sessionId },
    })
    await use(context)
    await context.close()
  },
})

export { expect } from '@playwright/test'
