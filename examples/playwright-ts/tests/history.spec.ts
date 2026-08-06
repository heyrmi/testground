import { expect, test } from './fixtures'

const PAGE = '/legacy/history'

test('pushState changes the URL without fetching anything', async ({ page }) => {
  await page.goto(PAGE)

  const documents: string[] = []
  page.on('request', (req) => {
    if (req.resourceType() === 'document') documents.push(req.url())
  })

  // framenavigated is not the signal to use here: it fires for same-document
  // navigations too, so it reports a navigation that never touched the network.
  let framenavigated = false
  page.on('framenavigated', () => {
    framenavigated = true
  })

  await page.getByTestId('push-one').click()

  await expect(page).toHaveURL(/\?step=1$/)
  await expect(page.getByTestId('current-step')).toHaveText('1')
  expect(documents, 'no document was requested').toHaveLength(0)
  expect(framenavigated, 'yet the navigation event still fired').toBe(true)
})

test('back rebuilds the page from the URL alone', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('push-one').click()
  await page.getByTestId('push-two').click()
  await expect(page.getByTestId('current-step')).toHaveText('2')

  await page.goBack()

  // The URL and the rendered state agreeing is the contract. Either alone can
  // be right while the pair is wrong.
  await expect(page).toHaveURL(/\?step=1$/)
  await expect(page.getByTestId('current-step')).toHaveText('1')
  await expect(page.getByTestId('popstate-count')).toHaveText('1')
})

test('replaceState leaves no entry, so back skips past it', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('push-one').click()
  await page.getByTestId('replace').click()
  await expect(page.getByTestId('current-step')).toHaveText('replaced')

  await page.goBack()

  // Not back to step 1: replaceState overwrote that entry rather than adding one.
  await expect(page).toHaveURL(new RegExp(`${PAGE}$`))
  await expect(page.getByTestId('current-step')).toHaveText('none')
})

test('a hash change never reaches the server', async ({ page }) => {
  await page.goto(PAGE)

  const requests: string[] = []
  page.on('request', (req) => requests.push(req.url()))

  await page.getByTestId('hash-link').click()

  await expect(page.getByTestId('current-hash')).toHaveText('#section-two')
  expect(requests.filter((url) => url.includes('/legacy/history'))).toHaveLength(0)
})
