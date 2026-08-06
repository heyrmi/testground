import { expect, test } from './fixtures'

const PAGE = '/live/server-sent-events'

test('a stream that finishes says so', async ({ page }) => {
  await page.goto(`${PAGE}?count=4&ms=40`)
  await page.getByTestId('events-start').click()

  await expect(page.getByTestId('events-state')).toHaveText('done')
  await expect(page.getByTestId('events-count')).toHaveText('4')
})

test('a stalled stream is neither failed nor finished', async ({ page }) => {
  await page.goto(`${PAGE}?before=3&ms=40`)
  await page.getByTestId('stall-start').click()

  await expect(page.getByTestId('stall-count')).toHaveText('3')

  // No error, no close, no done event. Waiting for a fourth update times out
  // with a message about the update rather than about the stream.
  await expect(page.getByTestId('stall-count')).not.toHaveText('4', { timeout: 1500 })
  await expect(page.getByTestId('stall-state')).toHaveText('streaming')
})

test('waiting for the network to quieten agrees the stalled page is done', async ({ page }) => {
  await page.goto(`${PAGE}?before=2&ms=30`)
  await page.getByTestId('stall-start').click()
  await expect(page.getByTestId('stall-count')).toHaveText('2')

  // Asserting what you actually expect -- that it stops at two -- is the
  // difference between a test that means something and a timeout.
  await page.waitForTimeout(600)
  await expect(page.getByTestId('stall-count')).toHaveText('2')
  await expect(page.getByTestId('stall-state')).not.toHaveText('done')
})

test('every intermediate state of the token stream reads correctly', async ({ page }) => {
  await page.goto(`${PAGE}?ms=20`)
  await page.getByTestId('stream-start').click()

  await expect(page.getByTestId('stream-tokens')).not.toHaveText('0')
  const partial = await page.getByTestId('stream-text').textContent()

  await expect(page.getByTestId('stream-state')).toHaveText('done', { timeout: 10_000 })
  const complete = await page.getByTestId('stream-text').textContent()

  // The partial was a real sentence. It was just not the whole one.
  expect(partial!.length).toBeGreaterThan(0)
  expect(complete!.length).toBeGreaterThan(partial!.length)
  expect(complete).toContain('after the last piece lands')
})

test('waiting for the done state, not for a substring that was true earlier', async ({ page }) => {
  await page.goto(`${PAGE}?ms=15`)
  await page.getByTestId('stream-start').click()

  await expect(page.getByTestId('stream-state')).toHaveText('done', { timeout: 10_000 })
  await expect(page.getByTestId('stream-text')).toContainText('A stream is not a page')
  await expect(page.getByTestId('stream-text')).toContainText('after the last piece lands')
})
