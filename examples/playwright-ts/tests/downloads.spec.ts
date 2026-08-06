import { expect, test } from './fixtures'

const PAGE = '/classic/downloads'

test('a download is not a navigation', async ({ page }) => {
  await page.goto(PAGE)

  // Waiting for the page to change would wait forever: the response is never
  // rendered. The download event is the thing to wait on.
  const [download] = await Promise.all([
    page.waitForEvent('download'),
    page.getByTestId('download-csv').click(),
  ])

  expect(download.suggestedFilename()).toBe('report.csv')
  await expect(page).toHaveURL(new RegExp(`${PAGE}$`))
})

test('generated content is byte-identical for a seed', async ({ page }) => {
  const first = await (await page.request.get('/classic/downloads/report.csv?rows=5')).text()
  const second = await (await page.request.get('/classic/downloads/report.csv?rows=5')).text()

  expect(first).toBe(second)
  expect(first.split('\n')[0]).toBe('index,name,status,amount')
  expect(first.trim().split('\n')).toHaveLength(6)
})

test('the archive really is an archive', async ({ page }) => {
  const response = await page.request.get('/classic/downloads/bundle.zip')

  expect(response.headers()['content-type']).toBe('application/zip')
  const body = await response.body()

  // "PK\x03\x04" — the local file header every zip starts with.
  expect(body.subarray(0, 4)).toEqual(Buffer.from([0x50, 0x4b, 0x03, 0x04]))
})

test('the image really is a PNG', async ({ page }) => {
  const body = await (await page.request.get('/classic/downloads/pixel.png')).body()

  expect(body.subarray(1, 4).toString()).toBe('PNG')
})

test('an inline file is rendered rather than downloaded', async ({ page }) => {
  const response = await page.request.get('/classic/downloads/notes.txt')

  expect(response.headers()['content-disposition']).toContain('inline')

  // No download event fires for this one, because the browser displays it.
  await page.goto('/classic/downloads/notes.txt')
  await expect(page.locator('body')).toContainText('Served inline')
})

test('a non-ASCII filename travels in filename*, not filename', async ({ page }) => {
  const response = await page.request.get('/classic/downloads/unicode.txt')
  const disposition = response.headers()['content-disposition']!

  expect(disposition).toContain("filename*=utf-8''")
  expect(disposition).toContain('r%C3%A9sum%C3%A9')

  // The framework decodes it for you; the raw header does not.
  const [download] = await Promise.all([
    page.waitForEvent('download'),
    page.goto(PAGE).then(() => page.getByTestId('download-unicode').click()),
  ])
  expect(download.suggestedFilename()).toContain('résumé')
})

test('the generated file takes its time, and the click does not', async ({ page }) => {
  await page.goto(PAGE)

  const started = Date.now()
  const [download] = await Promise.all([
    page.waitForEvent('download'),
    page.getByTestId('download-slow').click(),
  ])
  await download.path()

  expect(Date.now() - started, 'the wait is on the transfer, not the click').toBeGreaterThan(2500)
  expect(download.suggestedFilename()).toBe('slow-report.csv')
})
