import { expect, test } from './fixtures'

const PAGE = '/classic/uploads'

const file = (name: string, mimeType: string, size = 64) => ({
  name,
  mimeType,
  buffer: Buffer.alloc(size, 'x'),
})

test('a file input is set through the API, never clicked', async ({ page }) => {
  await page.goto(PAGE)

  // Clicking it opens an operating-system picker nothing can drive.
  await page.getByTestId('file-single').setInputFiles(file('notes.txt', 'text/plain'))
  await page.getByTestId('submit').click()

  await expect(page.getByTestId('upload-row')).toHaveCount(1)
  await expect(page.getByTestId('accepted-count')).toHaveText('1')
})

test('several files go in one input', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('file-multiple').setInputFiles([
    file('one.txt', 'text/plain'),
    file('two.csv', 'text/csv'),
    file('three.png', 'image/png'),
  ])
  await page.getByTestId('submit').click()

  await expect(page.getByTestId('upload-row')).toHaveCount(3)
  await expect(page.getByTestId('accepted-count')).toHaveText('3')
})

test('accept filters the picker and stops nothing', async ({ page }) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('file-restricted')).toHaveAttribute('accept', '.png,.jpg,.jpeg')

  // The attribute says images only. The input takes this without complaint,
  // which is why client-side type checking is advisory.
  await page.getByTestId('file-restricted').setInputFiles(file('script.sh', 'application/x-sh'))
  await page.getByTestId('submit').click()

  await expect(page.getByTestId('rejected-count')).toHaveText('1')
  await expect(page.locator('[data-testid="upload-row"][data-name="script.sh"]')).toContainText(
    'rejected',
  )
})

test('the size limit fails only after the whole file has arrived', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('file-single').setInputFiles(file('big.txt', 'text/plain', 100 * 1024))
  await page.getByTestId('submit').click()

  const row = page.locator('[data-testid="upload-row"][data-name="big.txt"]')
  await expect(row).toContainText('larger than 65536 bytes')
  await expect(page.getByTestId('rejected-count')).toHaveText('1')
})

test('the server reports size and type for what it received', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('file-single').setInputFiles(file('exact.csv', 'text/csv', 1234))
  await page.getByTestId('submit').click()

  const row = page.locator('[data-testid="upload-row"][data-name="exact.csv"]')
  await expect(row).toContainText('1234')
  await expect(row).toContainText('text/csv')
  await expect(row).toContainText('accepted')
})
