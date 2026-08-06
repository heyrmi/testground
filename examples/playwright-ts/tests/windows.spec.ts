import { expect, test } from './fixtures'

const PAGE = '/legacy/windows'

test('a new tab is a context your locators are not pointing at', async ({ page, context }) => {
  await page.goto(PAGE)

  // Waiting for the page before the click. After the click the event is gone.
  const [opened] = await Promise.all([
    context.waitForEvent('page'),
    page.getByTestId('blank-link').click(),
  ])
  await opened.waitForLoadState()

  await expect(opened.getByTestId('popup-kind')).toHaveText('tab')

  // The original page never changed, which is what makes this fail quietly.
  await expect(page.getByTestId('from-popup')).toHaveText('nothing yet')
  await opened.close()
})

test('window.open with dimensions opens the same way', async ({ page, context }) => {
  await page.goto(PAGE)

  const [popup] = await Promise.all([
    context.waitForEvent('page'),
    page.getByTestId('open-popup').click(),
  ])
  await popup.waitForLoadState()

  await expect(popup.getByTestId('popup-kind')).toHaveText('sized')
  await popup.close()
})

test('a popup that closes itself has to be read quickly', async ({ page, context }) => {
  await page.goto(PAGE)

  const [popup] = await Promise.all([
    context.waitForEvent('page'),
    page.getByTestId('open-closing').click(),
  ])
  await popup.waitForLoadState()
  await expect(popup.getByTestId('popup-kind')).toHaveText('closing')

  await popup.waitForEvent('close', { timeout: 5000 })
  expect(popup.isClosed()).toBe(true)
})

test('the opener outlives the popup, which makes it the better target', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('open-writer').click()

  // The popup wrote this and then closed. Asserting here needs no handle on
  // the window at all, and cannot race its closing.
  await expect(page.getByTestId('from-popup')).toHaveText('written by the popup')
})
