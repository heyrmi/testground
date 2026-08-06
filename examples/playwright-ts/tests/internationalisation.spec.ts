import { expect, test } from './fixtures'

const PAGE = '/app/internationalisation'

test('switching locale changes the direction, which anything positional depends on', async ({
  page,
}) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('locale-panel')).toHaveAttribute('data-dir', 'ltr')

  await page.getByTestId('locale-ar-EG').click()

  // Assert on the attribute the panel publishes, not on the words in it.
  await expect(page.getByTestId('locale-panel')).toHaveAttribute('data-dir', 'rtl')
  await expect(page.getByTestId('locale-panel')).toHaveAttribute('data-locale', 'ar-EG')
})

test('a number assertion written for one locale fails in another', async ({ page }) => {
  await page.goto(PAGE)
  const english = await page.getByTestId('format-number').textContent()

  await page.getByTestId('locale-de-DE').click()
  const german = await page.getByTestId('format-number').textContent()

  // Same amount, separators swapped. Neither string is wrong.
  expect(english).toContain(',')
  expect(german).toContain('.')
  expect(german).not.toBe(english)
})

test('the same instant reads as two different days', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('locale-en-GB').click()
  const british = await page.getByTestId('format-date').textContent()

  await page.getByTestId('locale-ja-JP').click()
  const japanese = await page.getByTestId('format-date').textContent()

  expect(british).not.toBe(japanese)
  expect(british, 'day first').toMatch(/^04/)
})

test('currency follows the locale, not the amount', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('locale-en-GB').click()
  await expect(page.getByTestId('format-currency')).toContainText('£')

  await page.getByTestId('locale-ja-JP').click()
  await expect(page.getByTestId('format-currency')).toContainText('￥')
})

test('a translated label is longer, which is a finding rather than a locator problem', async ({
  page,
}) => {
  await page.goto(PAGE)
  const english = Number(await page.getByTestId('label-length').textContent())

  await page.getByTestId('locale-de-DE').click()
  const german = Number(await page.getByTestId('label-length').textContent())

  expect(german).toBeGreaterThan(english)
})

test('two strings render identically and compare as different', async ({ page }) => {
  await page.goto(PAGE)

  // On screen they are the same word. To a comparison they are not.
  await expect(page.getByTestId('naive-equal')).toHaveText('false')
  await expect(page.getByTestId('normalised-equal')).toHaveText('true')

  const [composed, decomposed] = await Promise.all([
    page.getByTestId('nfc').textContent(),
    page.getByTestId('nfd').textContent(),
  ])
  expect(composed).not.toBe(decomposed)
  expect(composed!.normalize()).toBe(decomposed!.normalize())
})

test('one emoji is neither one character nor one code point', async ({ page }) => {
  await page.goto(PAGE)

  const length = Number(await page.getByTestId('family-length').textContent())
  const codepoints = Number(await page.getByTestId('family-codepoints').textContent())

  expect(length).toBeGreaterThan(codepoints)
  expect(codepoints).toBeGreaterThan(1)

  // What a person would call one thing.
  const graphemes = await page.evaluate(
    () => [...new Intl.Segmenter('en', { granularity: 'grapheme' })
      .segment(document.querySelector('[data-testid="family"]')!.textContent!)].length,
  )
  expect(graphemes).toBe(1)
})

test('plural categories are not two everywhere', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('locale-en-GB').click()
  await expect(page.getByTestId('plural-one')).toHaveText('one')
  await expect(page.getByTestId('plural-two')).toHaveText('other')

  await page.getByTestId('locale-ar-EG').click()
  await expect(page.getByTestId('plural-two')).toHaveText('two')
})

test('the input round-trips a non-Latin script', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('locale-hi-IN').click()

  await page.getByTestId('script-input').fill('नमस्ते')
  await expect(page.getByTestId('typed-back')).toHaveText('नमस्ते')
})
