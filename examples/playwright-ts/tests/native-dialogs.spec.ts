import { expect, test } from './fixtures'

const PAGE = '/legacy/native-dialogs'

test('the handler is registered before the click, because there is no after', async ({ page }) => {
  await page.goto(PAGE)

  let message = ''
  page.once('dialog', async (dialog) => {
    message = dialog.message()
    await dialog.accept()
  })
  await page.getByTestId('fire-alert').click()

  // The dialog text exists nowhere in the DOM. This is the only place to read it.
  expect(message).toContain('nothing here to locate')
  await expect(page.getByTestId('dialog-result')).toHaveText('alert acknowledged')
})

test('accepting and dismissing a confirm are different features', async ({ page }) => {
  await page.goto(PAGE)

  page.once('dialog', (dialog) => dialog.accept())
  await page.getByTestId('fire-confirm').click()
  await expect(page.getByTestId('dialog-result')).toHaveText('confirm accepted')

  page.once('dialog', (dialog) => dialog.dismiss())
  await page.getByTestId('fire-confirm').click()

  // Same click, same page, opposite behaviour. A framework that answers for
  // you has picked one of these without telling anyone.
  await expect(page.getByTestId('dialog-result')).toHaveText('confirm dismissed')
})

test('a prompt carries a value back', async ({ page }) => {
  await page.goto(PAGE)

  page.once('dialog', (dialog) => {
    expect(dialog.defaultValue()).toBe('default text')
    return dialog.accept('typed by the test')
  })
  await page.getByTestId('fire-prompt').click()

  await expect(page.getByTestId('dialog-result')).toHaveText('prompt returned: typed by the test')
})

test('a chained pair fires the handler twice', async ({ page }) => {
  await page.goto(PAGE)

  const seen: string[] = []
  page.on('dialog', async (dialog) => {
    seen.push(dialog.type())
    await dialog.accept()
  })
  await page.getByTestId('fire-chain').click()

  await expect(page.getByTestId('dialog-result')).toHaveText('chain accepted')
  expect(seen, 'a handler that only expects one dialog leaves the second blocking').toEqual([
    'alert',
    'confirm',
  ])
})

test('a dialog can arrive when nothing asked for one', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('fire-delayed').click()

  // Two seconds later, long after the click resolved. This is how a suite
  // that was passing starts hanging after an unrelated change.
  let arrived = false
  page.once('dialog', async (dialog) => {
    arrived = true
    await dialog.accept()
  })

  await expect(page.getByTestId('dialog-result')).toHaveText('delayed alert acknowledged')
  expect(arrived).toBe(true)
})

test('a beforeunload handler is registered', async ({ page }) => {
  await page.goto(PAGE)

  // Checked by dispatching the event rather than by navigating: whether the
  // browser actually shows the prompt depends on interaction heuristics that
  // differ per engine, and the handler's presence is the contract here.
  const guarded = await page.evaluate(() => {
    const event = new Event('beforeunload', { cancelable: true }) as BeforeUnloadEvent
    window.dispatchEvent(event)
    return event.returnValue !== '' || event.defaultPrevented
  })
  expect(guarded).toBe(true)
})
