import { expect, test } from './fixtures'
import type { Page } from '@playwright/test'

const PAGE = '/app/checkout'

const add = (page: Page, sku: string) =>
  page.locator(`[data-testid="product"][data-sku="${sku}"]`).getByTestId('add-to-cart').click()

test('the whole flow, end to end', async ({ page }) => {
  await page.goto(PAGE)

  await add(page, 'TG-PAD-01')
  await expect(page.getByTestId('cart-count')).toHaveText('1')
  await expect(page.getByTestId('subtotal')).toHaveText('£24.00')
  await expect(page.getByTestId('shipping')).toHaveText('£4.99')

  await page.getByTestId('coupon-code').fill('SAVE10')
  await page.getByTestId('apply-coupon').click()
  await expect(page.getByTestId('discount')).toHaveText('£2.40')
  await expect(page.getByTestId('total')).toHaveText('£26.59')

  await page.getByTestId('go-to-payment').click()
  await expect(page.getByTestId('step')).toHaveText('pay')

  await page.getByTestId('checkout-card').fill('4242424242424242')
  await page.getByTestId('place-order').click()

  await expect(page.getByTestId('step')).toHaveText('done')
  await expect(page.getByTestId('order-number')).toHaveText(/^TG-\d+$/)
})

test('filters combine, and out of stock is not addable', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('category').selectOption('cables')
  await expect(page.getByTestId('product')).toHaveCount(2)

  await page.getByTestId('search').fill('adapter')
  await expect(page.getByTestId('product')).toHaveCount(1)

  await page.getByTestId('search').fill('')
  await page.getByTestId('category').selectOption('peripherals')
  await expect(
    page.locator('[data-testid="product"][data-sku="TG-HUB-01"]').getByTestId('add-to-cart'),
  ).toBeDisabled()
})

// The composite's sharpest edge, and the reason a total carried forward from
// an earlier step is a total nobody will be charged.
test('a coupon stops applying when the cart shrinks under it', async ({ page }) => {
  await page.goto(PAGE)
  await add(page, 'TG-MON-01')

  await page.getByTestId('coupon-code').fill('BIGSPEND')
  await page.getByTestId('apply-coupon').click()
  await expect(page.getByTestId('discount')).toHaveText('£43.80')

  await page.locator('[data-testid="cart-line"][data-sku="TG-MON-01"]')
    .getByTestId('remove-line').click()
  await add(page, 'TG-CAB-01')

  await expect(page.getByTestId('discount')).toHaveText('£0.00')
  await expect(page.getByTestId('coupon-note')).toContainText('no longer applies')
})

test('a refused coupon says which refusal it was', async ({ page }) => {
  await page.goto(PAGE)
  await add(page, 'TG-PAD-01')

  for (const [code, expected] of [
    ['NOPE', 'no such coupon'],
    ['LASTYEAR', 'expired'],
    ['BIGSPEND', 'below this coupon'],
  ] as const) {
    await page.getByTestId('coupon-code').fill(code)
    await page.getByTestId('apply-coupon').click()
    await expect(page.getByTestId('coupon-note')).toContainText(expected)
  }
})

test('the payment outcome is chosen, not discovered', async ({ page }) => {
  await page.goto(PAGE)
  await add(page, 'TG-PAD-01')
  await page.getByTestId('go-to-payment').click()

  await page.getByTestId('checkout-card').fill('4000000000000002')
  await page.getByTestId('place-order').click()
  await expect(page.getByTestId('payment-error')).toContainText('declined')

  await page.getByTestId('checkout-card').fill('4000000000009995')
  await page.getByTestId('place-order').click()
  await expect(page.getByTestId('payment-error')).toContainText('insufficient funds')

  await page.getByTestId('checkout-card').fill('4242424242424242')
  await page.getByTestId('place-order').click()
  await expect(page.getByTestId('order-number')).toBeVisible()
})

test('placing an order empties the cart, so a stale page cannot charge twice', async ({ page }) => {
  await page.goto(PAGE)
  await add(page, 'TG-PAD-01')
  await page.getByTestId('go-to-payment').click()
  await page.getByTestId('place-order').click()
  await expect(page.getByTestId('order-number')).toBeVisible()

  // Correct behaviour that reads as a broken test until you know why.
  const second = await page.request.post('/api/app/shop/checkout', {
    data: { email: 'buyer@example.test', card: '4242424242424242' },
  })
  expect(second.status()).toBe(402)
  expect((await second.json()).error).toContain('nothing in the cart')
})

test('two workers shop independently', async ({ page, playwright, baseURL }) => {
  await page.goto(PAGE)
  await add(page, 'TG-KEY-01')
  await expect(page.getByTestId('cart-count')).toHaveText('1')

  const other = await playwright.request.newContext({
    baseURL,
    extraHTTPHeaders: { 'X-Playground-Session': 'shopper-two' },
  })
  const theirs = await (await other.get('/api/app/shop/cart')).json()
  expect(theirs.count, 'a shared cart would make parallel tests useless').toBe(0)
  await other.dispose()
})
