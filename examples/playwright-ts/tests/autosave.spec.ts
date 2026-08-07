import { expect, test } from './fixtures'
import type { Page } from '@playwright/test'

const PAGE = '/app/autosave'

const mergeRow = (page: Page, field: string) =>
  page.locator(`[data-testid="merge-row"][data-field="${field}"]`)

/**
 * Loads the page, lets a second writer move the record while this page is not
 * looking, and then edits a different field so the next autosave collides.
 * Waiting for version one first is what makes the collision certain: an
 * other-writer call that lands before the page has read the record would be
 * the version the page loads, and there would be nothing to conflict with.
 */
async function conflict(page: Page) {
  await page.goto(`${PAGE}?debounceMs=100&latencyMs=100`)
  await expect(page.getByTestId('record-version')).toHaveText('1')

  await page.request.post('/api/app/autosave/other-writer')

  await page.getByTestId('field-title').fill('Winter timetable')
  await expect(page.getByTestId('conflict')).toBeVisible()
}

test('the record starts unwritten, and the indicator already says saved', async ({ page }) => {
  await page.goto(PAGE)

  await expect(page.getByTestId('record-version')).toHaveText('1')
  await expect(page.getByTestId('updated-by')).toHaveText('nobody')
  await expect(page.getByTestId('save-count')).toHaveText('0')
  await expect(page.getByTestId('save-state')).toHaveText('saved')
})

test('waiting for “saved” is satisfied by the word that was already there', async ({ page }) => {
  await page.goto(`${PAGE}?debounceMs=10000`)

  await page.getByTestId('field-title').fill('Winter timetable')

  // This passes, and it passes instantly, because the indicator describes the
  // autosave loop and the loop has nothing to report until the debounce ends.
  // Ten seconds of it are still ahead. This is the whole point of the page.
  await expect(page.getByTestId('save-state')).toHaveText('saved')

  // What actually happened: nothing left the browser.
  await expect(page.getByTestId('record-version')).toHaveText('1')
  await expect(page.getByTestId('save-count')).toHaveText('0')
})

test('the version is the fact worth waiting for', async ({ page }) => {
  await page.goto(`${PAGE}?debounceMs=100&latencyMs=100`)

  await page.getByTestId('field-title').fill('Winter timetable')

  // Only the server moves either of these, so neither can be satisfied by a
  // value the page was already showing.
  await expect(page.getByTestId('record-version')).toHaveText('2')
  await expect(page.getByTestId('save-count')).toHaveText('1')
  await expect(page.getByTestId('updated-by')).toHaveText('this page')

  const stored = await (await page.request.get('/api/app/autosave/record')).json()
  expect(stored.record.fields.title).toBe('Winter timetable')
})

test('the middle state is real, and it is where the guard applies', async ({ page }) => {
  await page.goto(`${PAGE}?debounceMs=0&latencyMs=2000`)

  await page.getByTestId('field-owner').fill('Dana Okonkwo')
  await expect(page.getByTestId('save-state')).toHaveText('saving')
  await expect(page.getByTestId('save-state')).toHaveText('saved')
  await expect(page.getByTestId('save-count')).toHaveText('1')
})

test('the server refuses a stale write and hands back what it is holding', async ({ page }) => {
  const fields = { title: 'Once', owner: 'unassigned', notes: 'n' }

  const first = await page.request.put('/api/app/autosave/record?latencyMs=0', {
    data: { version: 1, fields },
  })
  expect(first.status()).toBe(200)
  expect((await first.json()).record.version).toBe(2)

  const stale = await page.request.put('/api/app/autosave/record?latencyMs=0', {
    data: { version: 1, fields: { ...fields, title: 'Twice' } },
  })
  expect(stale.status()).toBe(409)

  const body = await stale.json()
  expect(body.record.version).toBe(2)
  expect(body.record.fields.title, 'a refused write must not half-apply').toBe('Once')
})

test('the other writer can be aimed, and refuses a field that does not exist', async ({ page }) => {
  const aimed = await page.request.post('/api/app/autosave/other-writer', {
    data: { field: 'notes', value: 'Cancelled in high winds.' },
  })
  expect(aimed.status()).toBe(200)
  expect((await aimed.json()).record.fields.notes).toBe('Cancelled in high winds.')

  // Refused rather than ignored: a bumped version for a misspelt field would
  // send you looking for the bug in the page.
  const wrong = await page.request.post('/api/app/autosave/other-writer', {
    data: { field: 'titel', value: 'x' },
  })
  expect(wrong.status()).toBe(400)
  expect((await wrong.json()).error).toContain('no such field')
})

test('the button plays the other writer, and tells this page nothing', async ({ page }) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('other-writer-note')).toHaveText('nobody else has written yet')

  await page.getByTestId('simulate-other-writer').click()
  await expect(page.getByTestId('other-writer-note')).toContainText('version 2')

  // The page is still holding the version it loaded, which is the position a
  // real editor would be in and the reason its next autosave collides.
  await expect(page.getByTestId('record-version')).toHaveText('1')
})

test('a stale autosave surfaces as a conflict rather than as a save', async ({ page }) => {
  await conflict(page)

  await expect(page.getByTestId('conflict-versions')).toContainText('version 1')
  await expect(page.getByTestId('conflict-versions')).toContainText('version 2')

  // The field still shows text the server has never accepted, and the word the
  // indicator offers for that is "idle".
  await expect(page.getByTestId('field-title')).toHaveValue('Winter timetable')
  await expect(page.getByTestId('save-state')).toHaveText('idle')
  await expect(page.getByTestId('save-count')).toHaveText('0')
})

test('keeping yours discards the change you never saw', async ({ page }) => {
  await conflict(page)

  await page.getByTestId('keep-mine').click()
  await expect(page.getByTestId('conflict')).toHaveCount(0)
  await expect(page.getByTestId('record-version')).toHaveText('3')
  await expect(page.getByTestId('save-count')).toHaveText('1')

  const stored = await (await page.request.get('/api/app/autosave/record')).json()
  expect(stored.record.fields.title).toBe('Winter timetable')
  expect(stored.record.fields.owner, 'their edit is gone and nothing said so').toBe('unassigned')
})

test('taking theirs discards yours, and writes nothing at all', async ({ page }) => {
  await conflict(page)

  await page.getByTestId('take-theirs').click()
  await expect(page.getByTestId('conflict')).toHaveCount(0)

  await expect(page.getByTestId('field-title')).toHaveValue('Ferry timetable rewrite')
  await expect(page.getByTestId('field-owner')).toHaveValue('Priya Raman')
  await expect(page.getByTestId('record-version')).toHaveText('2')
  await expect(page.getByTestId('updated-by')).toHaveText('another writer')

  // Adopting a record is not a write, so nothing was acknowledged.
  await expect(page.getByTestId('save-count')).toHaveText('0')
})

test('only the merge keeps both changes', async ({ page }) => {
  await conflict(page)
  await page.getByTestId('show-merge').click()
  await expect(page.getByTestId('merge-view')).toBeVisible()

  // The merge opens on the union of the two sets of edits: the field this page
  // changed takes mine, the field only they changed takes theirs.
  await expect(mergeRow(page, 'title').getByTestId('merge-choice')).toHaveText('mine')
  await expect(mergeRow(page, 'owner').getByTestId('merge-choice')).toHaveText('theirs')
  await expect(mergeRow(page, 'title').getByTestId('merge-theirs')).toHaveText(
    'Ferry timetable rewrite',
  )
  await expect(mergeRow(page, 'owner').getByTestId('merge-mine')).toHaveText('unassigned')

  await mergeRow(page, 'owner').getByTestId('merge-pick').click()
  await expect(mergeRow(page, 'owner').getByTestId('merge-choice')).toHaveText('mine')
  await mergeRow(page, 'owner').getByTestId('merge-pick').click()
  await expect(mergeRow(page, 'owner').getByTestId('merge-choice')).toHaveText('theirs')

  await page.getByTestId('save-merge').click()
  await expect(page.getByTestId('conflict')).toHaveCount(0)
  await expect(page.getByTestId('record-version')).toHaveText('3')

  const stored = await (await page.request.get('/api/app/autosave/record')).json()
  expect(stored.record.fields.title).toBe('Winter timetable')
  expect(stored.record.fields.owner).toBe('Priya Raman')
})

test('leaving is refused while a write is in flight', async ({ page }) => {
  await page.goto(`${PAGE}?debounceMs=0&latencyMs=2000`)

  await page.getByTestId('field-notes').fill('Four crossings in summer, weather permitting.')
  await expect(page.getByTestId('save-state')).toHaveText('saving')

  await page.getByTestId('leave-link').click()
  await expect(page.getByTestId('leave-blocked')).toBeVisible()
  await expect(page.getByTestId('field-notes')).toBeVisible()

  // The same click, once the server has acknowledged, goes through.
  await expect(page.getByTestId('save-count')).toHaveText('1')
  await page.getByTestId('leave-link').click()
  await expect(page.getByTestId('field-notes')).toHaveCount(0)
})

test('the beforeunload guard exists only while a write is in flight', async ({ page }) => {
  await page.goto(`${PAGE}?debounceMs=0&latencyMs=2000`)

  // Dispatched rather than navigated: whether the browser actually shows the
  // prompt depends on interaction heuristics that differ per engine, and the
  // handler's presence is the contract here.
  const guarded = () =>
    page.evaluate(() => {
      const event = new Event('beforeunload', { cancelable: true })
      window.dispatchEvent(event)
      return event.defaultPrevented
    })

  expect(await guarded()).toBe(false)

  await page.getByTestId('field-title').fill('Winter timetable')
  await expect(page.getByTestId('save-state')).toHaveText('saving')
  expect(await guarded()).toBe(true)

  await expect(page.getByTestId('save-count')).toHaveText('1')
  expect(await guarded()).toBe(false)
})

test('two workers edit their own copy of the record', async ({ page, playwright, baseURL }) => {
  await page.goto(`${PAGE}?debounceMs=100&latencyMs=100`)
  await page.getByTestId('field-title').fill('Winter timetable')
  await expect(page.getByTestId('record-version')).toHaveText('2')

  const other = await playwright.request.newContext({
    baseURL,
    extraHTTPHeaders: { 'X-Playground-Session': 'autosave-two' },
  })
  const theirs = await (await other.get('/api/app/autosave/record')).json()

  expect(theirs.record.version, 'a shared record would make every conflict a lottery').toBe(1)
  expect(theirs.record.fields.title).toBe('Ferry timetable rewrite')
  await other.dispose()
})
