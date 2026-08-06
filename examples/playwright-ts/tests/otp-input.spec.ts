import { expect, test } from './fixtures'

const PAGE = '/app/otp-input'
const CODE = '314159'

test('filling the first box with the whole code leaves one digit', async ({ page }) => {
  await page.goto(PAGE)

  // The obvious approach. Each box holds one character, so five of the six
  // digits are simply dropped -- and the failure looks like a truncation bug
  // in the page rather than a mistake in the test.
  await page.getByTestId('otp-0').fill(CODE)

  await expect(page.getByTestId('otp-value')).toHaveText('3')
  await expect(page.getByTestId('otp-verdict')).toHaveText('incomplete')
})

test('one character per box, letting the focus move itself', async ({ page }) => {
  await page.goto(PAGE)

  for (const [index, digit] of [...CODE].entries()) {
    await page.getByTestId(`otp-${index}`).fill(digit)
  }

  await expect(page.getByTestId('otp-value')).toHaveText(CODE)
  await expect(page.getByTestId('otp-verdict')).toHaveText('accepted')
})

test('typing advances the focus without being told to', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('otp-0').fill('3')
  await expect(page.getByTestId('otp-1')).toBeFocused()

  await page.keyboard.type('1')
  await expect(page.getByTestId('otp-2')).toBeFocused()
})

test('pasting spreads the code across every box in one step', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('otp-0').focus()
  await page.evaluate((code) => {
    const transfer = new DataTransfer()
    transfer.setData('text', code)
    document.activeElement?.dispatchEvent(
      new ClipboardEvent('paste', { clipboardData: transfer, bubbles: true, cancelable: true }),
    )
  }, CODE)

  await expect(page.getByTestId('otp-value')).toHaveText(CODE)
  await expect(page.getByTestId('otp-verdict')).toHaveText('accepted')
})

test('backspace on an empty box walks backwards', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('otp-0').fill('3')
  await page.getByTestId('otp-1').fill('1')

  await page.getByTestId('otp-2').press('Backspace')
  await expect(page.getByTestId('otp-1')).toBeFocused()
})

test('a wrong code is rejected rather than left incomplete', async ({ page }) => {
  await page.goto(PAGE)

  for (const [index, digit] of [...'999999'].entries()) {
    await page.getByTestId(`otp-${index}`).fill(digit)
  }
  await expect(page.getByTestId('otp-verdict')).toHaveText('rejected')
})
