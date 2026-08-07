import { expect, test } from './fixtures'
import type { Page } from '@playwright/test'

const PAGE = '/app/admin-crud'

const row = (page: Page, id: string) => page.locator(`[data-testid="account-row"][data-id="${id}"]`)

// The seeded accounts the server refuses to change or delete. They are published
// on the row rather than discovered, so a test can choose the rollback path.
const lockedRows = (page: Page) => page.locator('[data-testid="account-row"][data-locked="true"]')

test('the table starts as twelve accounts, three of which the server will not change', async ({
  page,
}) => {
  await page.goto(PAGE)

  await expect(page.getByTestId('account-row')).toHaveCount(12)
  await expect(page.getByTestId('row-count')).toHaveText('12 of 12')
  await expect(lockedRows(page)).toHaveCount(3)
  await expect(page.getByTestId('queued-deletes')).toHaveText('0')
  await expect(page.getByTestId('in-flight')).toHaveText('0')
})

// The id is the thing that changes, so it is the one attribute a locator must
// not be built from before the server has answered.
test('a created row is not the row the server stored', async ({ page }) => {
  await page.goto(`${PAGE}?latencyMs=600`)

  await page.getByTestId('new-name').fill('Wilhelmina Vandertramp')
  await page.getByTestId('create-account').click()

  const optimistic = page.locator('[data-testid="account-row"][data-id^="tmp-"]')
  await expect(optimistic).toHaveCount(1)
  await expect(optimistic.getByTestId('account-state')).toHaveText('creating')

  // Settling is what makes the next assertion mean anything.
  await expect(page.getByTestId('in-flight')).toHaveText('0')

  await expect(optimistic).toHaveCount(0)
  await expect(row(page, 'acct-13').getByTestId('account-name')).toHaveText(
    'Wilhelmina Vandertramp',
  )
  await expect(row(page, 'acct-13').getByTestId('account-state')).toHaveText('saved')
})

test('a create the server refuses leaves a row that was never there', async ({ page }) => {
  await page.goto(`${PAGE}?latencyMs=600`)

  const taken = await row(page, 'acct-1').getByTestId('account-name').textContent()
  await page.getByTestId('new-name').fill(taken ?? '')
  await page.getByTestId('create-account').click()

  // Thirteen rows, one of which the server is about to refuse to store.
  await expect(page.getByTestId('account-row')).toHaveCount(13)

  await expect(page.getByTestId('account-row')).toHaveCount(12)
  await expect(page.getByTestId('rollback-notice')).toContainText('already exists')
  await expect(page.getByTestId('rollback-count')).toHaveText('1')
})

test('an edit the server refuses is undone after it has already been shown', async ({ page }) => {
  await page.goto(`${PAGE}?latencyMs=600`)
  const locked = lockedRows(page).first()
  const before = await locked.getByTestId('account-name').textContent()

  await locked.getByTestId('row-edit').click()
  await page.getByTestId('edit-name').fill('Renamed by the suite')
  await page.getByTestId('edit-save').click()

  await expect(locked.getByTestId('account-name')).toHaveText('Renamed by the suite')
  await expect(locked.getByTestId('account-state')).toHaveText('saving')

  await expect(page.getByTestId('in-flight')).toHaveText('0')
  await expect(locked.getByTestId('account-name')).toHaveText(before ?? '')
  await expect(page.getByTestId('rollback-notice')).toContainText('locked')
})

// The approach that looks like it works. It will keep passing while the write
// silently fails, because it asserts on a value the client invented.
test('asserting straight after the click passes against a state the server never had', async ({
  page,
}) => {
  await page.goto(`${PAGE}?latencyMs=1500`)
  const locked = lockedRows(page).first()
  const id = await locked.getAttribute('data-id')

  await locked.getByTestId('row-edit').click()
  await page.getByTestId('edit-name').fill('Never stored')
  await page.getByTestId('edit-save').click()

  await expect(locked.getByTestId('account-name')).toHaveText('Never stored')

  const stored = await (await page.request.get('/api/app/admin-crud/accounts')).json()
  const account = stored.accounts.find((a: { id: string }) => a.id === id)
  expect(account.name, 'the page said one thing and the server holds another').not.toBe(
    'Never stored',
  )
})

test('a queued delete has not been sent, and undo means the server never hears about it', async ({
  page,
}) => {
  await page.goto(`${PAGE}?latencyMs=100&undoMs=8000`)

  await row(page, 'acct-1').getByTestId('row-delete').click()
  await expect(row(page, 'acct-1')).toHaveCount(0)
  await expect(page.getByTestId('undo-toast')).toBeVisible()
  await expect(page.getByTestId('queued-deletes')).toHaveText('1')

  // Gone from the page, untouched on the server. Reading it here is reading too
  // early rather than reading a stale value.
  const during = await (await page.request.get('/api/app/admin-crud/accounts')).json()
  expect(during.accounts.map((a: { id: string }) => a.id)).toContain('acct-1')

  await page.getByTestId('undo-delete').click()
  await expect(row(page, 'acct-1')).toHaveCount(1)
  await expect(page.getByTestId('undo-toast')).toHaveCount(0)
  await expect(page.getByTestId('in-flight')).toHaveText('0')
})

test('the delete leaves when the window closes, and a locked account comes back', async ({
  page,
}) => {
  await page.goto(`${PAGE}?latencyMs=400&undoMs=400`)
  const id = await lockedRows(page).first().getAttribute('data-id')

  await lockedRows(page).first().getByTestId('row-delete').click()
  await expect(row(page, id ?? '')).toHaveCount(0)

  // Two things settle here, and only in this order: the window closes, then the
  // request the window was holding is answered.
  await expect(page.getByTestId('queued-deletes')).toHaveText('0')
  await expect(page.getByTestId('in-flight')).toHaveText('0')

  await expect(row(page, id ?? '')).toHaveCount(1)
  await expect(page.getByTestId('rollback-notice')).toContainText('locked')
  await expect(page.getByTestId('rollback-count')).toHaveText('1')
})

test('select all covers the rows the filter left, not the table', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('role-filter').selectOption('editor')
  await expect(page.getByTestId('account-row')).toHaveCount(4)

  await page.getByTestId('select-all').check()

  // Four, not twelve. In an admin UI the difference between those two numbers
  // is the whole of the damage.
  await expect(page.getByTestId('selected-count')).toHaveText('4')
  await expect(page.getByTestId('row-count')).toHaveText('4 of 12')
})

test('the selection outlives the filter that made it', async ({ page }) => {
  await page.goto(`${PAGE}?latencyMs=600&undoMs=0`)

  await page.getByTestId('role-filter').selectOption('viewer')
  await expect(page.getByTestId('row-count')).toHaveText('4 of 12')
  await page.getByTestId('select-all').check()
  await expect(page.getByTestId('selected-count')).toHaveText('4')

  await page.getByTestId('role-filter').selectOption('')
  await expect(page.getByTestId('account-row')).toHaveCount(12)

  // The box reads unchecked because not every visible row is chosen. The four
  // it did choose are still selected, and still about to be deleted.
  await expect(page.getByTestId('select-all')).not.toBeChecked()
  await expect(page.getByTestId('selected-count')).toHaveText('4')

  await page.getByTestId('bulk-delete').click()
  await expect(page.getByTestId('account-row')).toHaveCount(8)

  // Three deletes are kept and the locked one is undone, so the count that
  // matters is the one after everything settles rather than the one on screen.
  await expect(page.getByTestId('account-row')).toHaveCount(9)
  await expect(page.getByTestId('rollback-count')).toHaveText('1')
  await expect(page.getByTestId('in-flight')).toHaveText('0')

  const left = await (await page.request.get('/api/app/admin-crud/accounts')).json()
  expect(left.accounts).toHaveLength(9)
})

test('two workers administer their own copy', async ({ page, playwright, baseURL }) => {
  await page.goto(`${PAGE}?latencyMs=0&undoMs=0`)

  await row(page, 'acct-1').getByTestId('row-delete').click()
  await expect(page.getByTestId('account-row')).toHaveCount(11)
  await expect(page.getByTestId('in-flight')).toHaveText('0')

  const other = await playwright.request.newContext({
    baseURL,
    extraHTTPHeaders: { 'X-Playground-Session': 'admin-two' },
  })
  const theirs = await (await other.get('/api/app/admin-crud/accounts')).json()
  expect(theirs.accounts, 'a shared table would make parallel tests useless').toHaveLength(12)
  await other.dispose()
})
