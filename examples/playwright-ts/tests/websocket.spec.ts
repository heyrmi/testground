import { expect, test } from './fixtures'

const PAGE = '/live/websocket'

test('the echo is a round trip, so the click and the reply are two events', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('echo-connect').click()
  await expect(page.getByTestId('echo-state')).toHaveText('open')

  await page.getByTestId('echo-input').fill('marco')
  await page.getByTestId('echo-send').click()

  // Reading the log straight after the click reads it before the server spoke.
  await expect(page.getByTestId('echo-last')).toHaveText('echo: marco')
  await expect(page.getByTestId('echo-count')).toHaveText('1')
})

test('the ticker pushes with nothing to wait after', async ({ page }) => {
  // count makes the server stop, which is what turns a moving number into a
  // settled one. Waiting for an exact value on a counter that increments
  // every sixty milliseconds is a race: the poll can arrive at four and again
  // at six, and never see five at all. That failed once under full parallel
  // load, which is exactly when it would.
  await page.goto(`${PAGE}?ms=60&count=5`)
  await page.getByTestId('ticker-connect').click()

  await expect(page.getByTestId('ticker-count')).toHaveText('5', { timeout: 10_000 })
  await expect(page.getByTestId('ticker-last-seq')).toHaveText('5')
})

test('the interval is the caller to choose, so a suite need not run at demo speed', async ({
  page,
}) => {
  await page.goto(`${PAGE}?ms=30&count=4`)
  await page.getByTestId('ticker-connect').click()

  await expect(page.getByTestId('ticker-count')).toHaveText('4')
  await expect(page.getByTestId('ticker-last-seq')).toHaveText('4')

  // count made the server stop, so the socket closes on its own.
  await expect(page.getByTestId('ticker-state')).toHaveText('closed')
})

test('the connection state is a better signal than the messages', async ({ page }) => {
  await page.goto(`${PAGE}?ms=40`)
  await page.getByTestId('ticker-connect').click()
  await expect(page.getByTestId('ticker-state')).toHaveText('open')

  await page.getByTestId('ticker-stop').click()
  await expect(page.getByTestId('ticker-state')).toHaveText('closed')

  // The messages already received stay on screen after the socket is gone,
  // which is why they cannot tell you whether it still is.
  const settled = await page.getByTestId('ticker-count').textContent()
  await expect(page.getByTestId('ticker-count')).toHaveText(settled!)
})
