import { expect, test } from './fixtures'
import type { Page } from '@playwright/test'

const PAGE = '/app/checkout'

const add = (page: Page, sku: string) =>
  page.locator(`[data-testid="product"][data-sku="${sku}"]`).getByTestId('add-to-cart').click()

// The cart is on the server, not in the page, and this is how you prove it.
test('the same cart is visible in the zone that runs no JavaScript', async ({ page }) => {
  await page.goto(PAGE)
  await add(page, 'TG-MON-01')
  await expect(page.getByTestId('cart-count')).toHaveText('1')

  await page.goto('/classic/cart')
  await expect(page.getByTestId('cart-count')).toHaveText('1')
  await expect(page.getByTestId('total')).toHaveText('£219.00')
  await expect(page.locator('[data-testid="cart-line"][data-sku="TG-MON-01"]')).toBeVisible()
})
test('and a change made there shows up here', async ({ page }) => {
  await page.goto('/classic/cart')
  await page.getByTestId('field-sku').selectOption('TG-BAG-01')
  await page.getByTestId('add-submit').click()
  await expect(page.getByTestId('cart-count')).toHaveText('1')

  // Setting up through the cheap interface and asserting through the real one.
  await page.goto(PAGE)
  await expect(page.getByTestId('cart-count')).toHaveText('1')
  await expect(page.locator('[data-testid="cart-line"][data-sku="TG-BAG-01"]')).toBeVisible()
})
test('an order placed in one zone is listed in the other', async ({ page }) => {
  await page.goto(PAGE)
  await add(page, 'TG-PAD-01')
  await page.getByTestId('go-to-payment').click()
  await page.getByTestId('place-order').click()
  const number = await page.getByTestId('order-number').textContent()

  await page.goto('/classic/cart')
  await expect(page.locator(`[data-testid="order-row"][data-order="${number}"]`)).toBeVisible()
  await expect(page.getByTestId('cart-empty')).toBeVisible()
})
