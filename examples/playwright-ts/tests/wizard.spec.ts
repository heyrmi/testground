import { expect, test } from './fixtures'
import type { Page } from '@playwright/test'

const PAGE = '/app/wizard'

const stepLink = (page: Page, n: number) =>
  page.locator(`[data-testid="step-link"][data-step="${n}"]`)

const errorFor = (page: Page, field: string) =>
  page.locator(`[data-testid="field-error"][data-field="${field}"]`)

const problemFor = (page: Page, field: string) =>
  page.locator(`[data-testid="problem"][data-field="${field}"]`)

async function account(page: Page, type: 'individual' | 'business', email = 'sam@example.test') {
  await page.getByTestId('account-type').selectOption(type)
  await page.getByTestId('email').fill(email)
  await page.getByTestId('next').click()
}

async function contact(page: Page) {
  await page.getByTestId('full-name').fill('Sam Okafor')
  await page.getByTestId('phone').fill('020 7946 0018')
  await page.getByTestId('next').click()
}

async function individualDetails(page: Page, born = '1990-04-12') {
  await page.getByTestId('date-of-birth').fill(born)
  await page.getByTestId('occupation').fill('Ceramicist')
  await page.getByTestId('next').click()
}

const draftOf = (page: Page) =>
  page.request.get('/api/app/wizard/draft').then((res) => res.json())

// The age rule is measured against the session clock, so a test that asserts a
// date of birth is too young is asserting on today's date unless it says
// otherwise. Pinning the clock is what keeps "under eighteen" meaning the same
// thing in 2026 and in 2040, and it is the reason the challenge declares a
// clock control at all.
const pinClock = (page: Page) =>
  page.request.post('/api/control/clock', {
    data: { action: 'set', instant: '2026-01-01T00:00:00Z' },
  })

test('four steps, a review, and a reference', async ({ page }) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('step-counter')).toHaveText('Step 1 of 4')

  await account(page, 'individual')
  await contact(page)
  await individualDetails(page)

  await expect(page.getByTestId('step-counter')).toHaveText('Step 4 of 4')
  await expect(page.locator('[data-testid="review-value"][data-field="email"]')).toHaveText(
    'sam@example.test',
  )

  await page.getByTestId('submit').click()
  await expect(page.getByTestId('reference')).toHaveText(/^WZ-\d+$/)
})

// The hazard the button's own state hides: it is enabled because it is always
// enabled, and the thing worth asserting on does not exist until after the click.
test('Next is enabled on an invalid step, and the error only exists after the click', async ({
  page,
}) => {
  await page.goto(PAGE)

  await expect(page.getByTestId('next')).toBeEnabled()
  await expect(page.getByTestId('field-error')).toHaveCount(0)

  await page.getByTestId('next').click()

  await expect(page.getByTestId('step-counter')).toHaveText('Step 1 of 4')
  await expect(errorFor(page, 'account-type')).toContainText('individual or a business')
  await expect(errorFor(page, 'email')).toContainText('not an email address')
})

test('step three asks a business different questions', async ({ page }) => {
  await page.goto(PAGE)
  await account(page, 'business')
  await contact(page)

  await expect(page.getByTestId('branch')).toHaveText('business')
  await expect(page.getByTestId('company-number')).toBeVisible()

  // Branch on the answer, not on the step number. A locator hard-coded to the
  // other branch reports a missing element, which reads as a broken selector
  // rather than as an answer two steps back deciding what exists.
  await expect(page.getByTestId('date-of-birth')).toHaveCount(0)

  await page.getByTestId('company-number').fill('01234567')
  await page.getByTestId('employees').fill('12')
  await page.getByTestId('next').click()

  await page.getByTestId('submit').click()
  await expect(page.getByTestId('reference')).toBeVisible()
})

test('going back keeps what was typed and tells the server nothing', async ({ page }) => {
  await page.goto(PAGE)
  await account(page, 'individual')

  await page.getByTestId('full-name').fill('Sam Okafor')
  await page.getByTestId('back').click()
  await expect(page.getByTestId('email')).toHaveValue('sam@example.test')

  await stepLink(page, 2).click()
  await expect(page.getByTestId('full-name')).toHaveValue('Sam Okafor')

  const draft = await draftOf(page)
  expect(draft.values['full-name'], 'a step is only stored when Next validates it').toBeUndefined()
  expect(draft.steps).toEqual([1])
})

// The cause is on step one and the message arrives on step four.
test('an address step one accepted is refused at the far end of the flow', async ({ page }) => {
  await page.goto(PAGE)
  await account(page, 'individual', 'sam@rejected.test')

  // Step one took it: the page checks the shape and nothing else.
  await expect(page.getByTestId('step-counter')).toHaveText('Step 2 of 4')

  await contact(page)
  await individualDetails(page)
  await page.getByTestId('submit').click()

  await expect(problemFor(page, 'email')).toContainText('rejected.test')
  await expect(problemFor(page, 'email')).toHaveAttribute('data-step', '1')
  await expect(page.getByTestId('reference')).toHaveCount(0)
})

test('the page checks a date of birth for its shape and the server for its meaning', async ({
  page,
}) => {
  await pinClock(page)
  await page.goto(PAGE)
  await account(page, 'individual')
  await contact(page)

  await page.getByTestId('date-of-birth').fill('12 April 1990')
  await page.getByTestId('occupation').fill('Ceramicist')
  await page.getByTestId('next').click()
  await expect(errorFor(page, 'date-of-birth')).toContainText('YYYY-MM-DD')

  // Well formed, so the page waves it through; too young, so the server does not.
  await page.getByTestId('date-of-birth').fill('2020-04-12')
  await page.getByTestId('next').click()
  await expect(page.getByTestId('step-counter')).toHaveText('Step 4 of 4')

  await page.getByTestId('submit').click()
  await expect(problemFor(page, 'date-of-birth')).toContainText('eighteen')
})

// The looks-like-it-works case. Everything on screen says business, the
// application that was lodged says individual, and nothing failed.
test('an answer changed and jumped past never reaches the server', async ({ page }) => {
  await page.goto(PAGE)
  await account(page, 'individual')
  await contact(page)
  await individualDetails(page)

  await stepLink(page, 1).click()
  await page.getByTestId('account-type').selectOption('business')
  await stepLink(page, 4).click()

  await expect(page.getByTestId('branch')).toHaveText('business')
  await expect(
    page.locator('[data-testid="review-value"][data-field="company-number"]'),
  ).toHaveText('not answered')

  await page.getByTestId('submit').click()
  await expect(page.getByTestId('reference')).toBeVisible()

  const draft = await draftOf(page)
  expect(
    draft.applications[0].values['account-type'],
    'the review showed one application and the server processed another',
  ).toBe('individual')
})

test('a step skipped after the branch changed is refused three steps later', async ({ page }) => {
  await page.goto(PAGE)
  await account(page, 'individual')
  await contact(page)
  await individualDetails(page)

  await stepLink(page, 1).click()
  await page.getByTestId('account-type').selectOption('business')
  await page.getByTestId('next').click() // this one the server does hear
  await stepLink(page, 4).click() // step three is never revisited, so nothing re-checks it

  await page.getByTestId('submit').click()

  await expect(page.getByTestId('submit-error')).toContainText('does not validate')
  await expect(problemFor(page, 'company-number')).toHaveAttribute('data-step', '3')
  await expect(problemFor(page, 'employees')).toBeVisible()
})

test('the step is not in the URL, so a reload restarts a flow the server still remembers', async ({
  page,
}) => {
  await page.goto(PAGE)
  await account(page, 'individual')
  await contact(page)

  await expect(page.getByTestId('step-counter')).toHaveText('Step 3 of 4')
  await expect(page).toHaveURL(/\/app\/wizard$/)

  await page.reload()

  await expect(page.getByTestId('step-counter')).toHaveText('Step 1 of 4')
  await expect(page.getByTestId('email')).toHaveValue('')

  const draft = await draftOf(page)
  expect(draft.values.email, 'the page forgot; the server did not').toBe('sam@example.test')
  expect(draft.steps).toEqual([1, 2])
})

test('the server validates the draft it holds, not the request body', async ({ page }) => {
  await page.goto(PAGE)

  const refused = await page.request.post('/api/app/wizard/submit', {
    data: { values: { 'account-type': 'individual', email: 'sam@example.test' } },
  })

  expect(refused.status()).toBe(409)
  expect((await refused.json()).error).toContain('no draft to submit')
})

test('two workers fill in their own application', async ({ page, playwright, baseURL }) => {
  await page.goto(PAGE)
  await account(page, 'individual')

  const other = await playwright.request.newContext({
    baseURL,
    extraHTTPHeaders: { 'X-Playground-Session': 'wizard-two' },
  })
  const theirs = await (await other.get('/api/app/wizard/draft')).json()
  expect(theirs.values.email, 'a shared draft would make parallel tests useless').toBeUndefined()
  await other.dispose()
})
