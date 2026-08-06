import { expect, test } from './fixtures'

const PAGE = '/app/pointer-menus'

test('the menu exists only while the pointer is inside the group', async ({ page }) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('hover-menu')).toHaveCount(0)

  await page.getByTestId('hover-trigger').hover()
  await expect(page.getByTestId('hover-menu')).toBeVisible()

  await page.getByTestId('gesture-target').hover()
  await expect(page.getByTestId('hover-menu')).toHaveCount(0)
})

test('an item can be chosen by moving onto the menu, not away from it', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('hover-trigger').hover()
  await page.getByTestId('menu-open').click()

  await expect(page.getByTestId('menu-choice')).toHaveText('open')
})

test('the submenu needs the parent hovered first', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('hover-trigger').hover()
  await expect(page.getByTestId('submenu')).toHaveCount(0)

  await page.getByTestId('menu-more').hover()
  await expect(page.getByTestId('submenu')).toBeVisible()

  await page.getByTestId('menu-archive').click()
  await expect(page.getByTestId('menu-choice')).toHaveText('archive')
})

test('right-click raises the page own menu, not the browser one', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('gesture-target').click({ button: 'right' })
  await expect(page.getByTestId('context-menu')).toBeVisible()

  await page.getByTestId('context-rename').click()
  await expect(page.getByTestId('menu-choice')).toHaveText('rename')
  await expect(page.getByTestId('context-menu')).toHaveCount(0)
})

test('a double click is also two single clicks', async ({ page }) => {
  await page.goto(PAGE)

  await page.getByTestId('gesture-target').dblclick()

  await expect(page.getByTestId('double-clicks')).toHaveText('1')

  // Asserting that exactly one click happened would be asserting something
  // false, however reasonable it sounds.
  await expect(page.getByTestId('single-clicks')).toHaveText('2')
})

test('a hold is three steps, because a click is all of them at once', async ({ page }) => {
  await page.goto(PAGE)
  const target = page.getByTestId('gesture-target')

  await target.click()
  await expect(page.getByTestId('long-presses')).toHaveText('0')

  const box = (await target.boundingBox())!
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2)
  await page.mouse.down()
  await page.waitForTimeout(650)
  await page.mouse.up()

  await expect(page.getByTestId('long-presses')).toHaveText('1')
})
