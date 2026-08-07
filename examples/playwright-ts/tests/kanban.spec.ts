import { expect, test } from './fixtures'
import type { Locator, Page } from '@playwright/test'

const PAGE = '/app/kanban'

const card = (page: Page, id: string) => page.locator(`[data-testid="card"][data-card-id="${id}"]`)
const column = (page: Page, id: string) =>
  page.locator(`[data-testid="column"][data-column="${id}"]`)

async function centre(target: Locator) {
  const box = await target.boundingBox()
  if (!box) throw new Error('nothing to aim at: the element has no box')
  return { x: box.x + box.width / 2, y: box.y + box.height / 2 }
}

/**
 * Press on a card and keep holding it, which is where every drag here starts.
 *
 * The scroll is not optional and not incidental. The board sits below the
 * description panel, so on a short viewport its cards are outside it, and a
 * bounding box is measured against the viewport rather than the document:
 * aiming the mouse at one of those coordinates dispatches the events to
 * nothing, elementFromPoint returns null, and the drag silently does nothing at
 * all. Clicking is exempt only because the click helper scrolls for you.
 */
async function hold(page: Page, id: string) {
  await card(page, id).scrollIntoViewIfNeeded()
  const at = await centre(card(page, id))
  await page.mouse.move(at.x, at.y)
  await page.mouse.down()
}

/**
 * Travel to the target rather than jumping to it. The page reads the drop
 * position off the pointer's last move, so a drag with no intervening moves
 * releases on the position it started from and changes nothing.
 */
async function travelTo(page: Page, target: Locator) {
  const at = await centre(target)
  await page.mouse.move(at.x, at.y, { steps: 8 })
}

async function dragOnto(page: Page, id: string, target: Locator) {
  // Both ends are brought into view before the pointer goes down. Scrolling
  // mid-drag would move the target out from under coordinates already measured.
  await target.scrollIntoViewIfNeeded()
  await hold(page, id)
  await travelTo(page, target)
  await page.mouse.up()
}

/** The board as the server holds it, which is not always the board on screen. */
async function serverColumn(page: Page, id: string): Promise<string[]> {
  const body = await (await page.request.get('/api/app/kanban/board')).json()
  const found = body.board.columns.find((one: { id: string }) => one.id === id)
  return found.cards.map((one: { id: string }) => one.id)
}

test('a card dragged between columns lands where it was aimed', async ({ page }) => {
  await page.goto(PAGE)
  await expect(card(page, 'card-1')).toHaveAttribute('data-column', 'todo')

  await dragOnto(page, 'card-1', card(page, 'card-4'))

  await expect(card(page, 'card-1')).toHaveAttribute('data-column', 'doing')
  await expect(card(page, 'card-1')).toHaveAttribute('data-position', '0')
  await expect(card(page, 'card-4')).toHaveAttribute('data-position', '1')
  await expect(column(page, 'doing').getByTestId('column-count')).toHaveText('2')
  await expect(page.getByTestId('board-rev')).toHaveText('1')
  await expect(page.getByTestId('last-drop')).toContainText('card-1 to doing 0')
})

test('a move made through the API arrives with nothing to wait after', async ({ page }) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('watchers')).toHaveText('1')

  // Nothing on the page was touched. The only thing that will make this card
  // move is a message the page did not ask for, so the wait has to be for the
  // board itself rather than for anything the test did.
  await page.request.post('/api/app/kanban/moves', {
    data: { card: 'card-2', column: 'doing', index: 0 },
  })

  await expect(card(page, 'card-2')).toHaveAttribute('data-column', 'doing')
  await expect(page.getByTestId('last-event')).toHaveText('board')
  await expect(page.getByTestId('board-rev')).toHaveText('1')
})

test('presence counts the tabs, and closing one puts the count back', async ({ page, context }) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('watchers')).toHaveText('1')

  const second = await context.newPage()
  await second.goto(PAGE)
  await expect(page.getByTestId('watchers')).toHaveText('2')

  // The count is a fact the server publishes, so a departure is something to
  // wait for rather than something to sleep off.
  await second.close()
  await expect(page.getByTestId('watchers')).toHaveText('1')
})

// The composite's sharpest edge: the drop target is chosen on the last pointer
// move and used on release, and the board is free to move in between.
test('a card arriving mid-drag sends the drop to a gap nobody aimed at', async ({ page }) => {
  await page.goto(PAGE)
  await expect(card(page, 'card-3')).toHaveAttribute('data-position', '2')

  // Aimed at the gap above card-2, which is where a release would land now.
  await hold(page, 'card-3')
  await travelTo(page, card(page, 'card-2'))
  await expect(page.getByTestId('drop-target')).toHaveText('todo 1')

  // The other writer, driven from the test so the arrival is something this
  // test caused and can wait for.
  await page.request.post('/api/app/kanban/moves', {
    data: { card: 'card-4', column: 'todo', index: 0 },
  })
  await expect(page.getByTestId('board-changed')).toBeVisible()
  await expect(card(page, 'card-4')).toHaveAttribute('data-column', 'todo')

  await page.mouse.up()

  // Index one now names the gap above card-1 rather than the gap above
  // card-2, because everything shifted down when card-4 arrived. The board
  // looks entirely plausible afterwards, which is why nothing reports it.
  await expect(card(page, 'card-4')).toHaveAttribute('data-position', '0')
  await expect(card(page, 'card-3')).toHaveAttribute('data-position', '1')
  await expect(card(page, 'card-1')).toHaveAttribute('data-position', '2')
  await expect(card(page, 'card-2')).toHaveAttribute('data-position', '3')
})

test('the board shown while offline is a local fiction, and the flush corrects it', async ({
  page,
}) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('connection-state')).toHaveText('online')

  await page.getByTestId('offline-toggle').click()
  await expect(page.getByTestId('connection-state')).toHaveText('offline')

  await dragOnto(page, 'card-1', card(page, 'card-4'))
  await dragOnto(page, 'card-2', card(page, 'card-1'))

  // The approach that looks like it works. Three cards really are in the
  // column, on screen, and this assertion passes -- it is just not an
  // assertion about the product.
  await expect(column(page, 'doing').getByTestId('column-count')).toHaveText('3')
  await expect(page.getByTestId('queued-count')).toHaveText('2')
  await expect(page.getByTestId('queued-move')).toHaveCount(2)

  expect(
    await serverColumn(page, 'doing'),
    'the server never heard about either move',
  ).toEqual(['card-4'])

  await page.getByTestId('offline-toggle').click()
  await expect(page.getByTestId('connection-state')).toHaveText('online')

  // Replayed in order against the board as it is now. The first move fills the
  // column's last free place and the second has nowhere to go.
  await expect(page.getByTestId('flush-note')).toContainText('1 of 2 queued moves applied')
  await expect(page.getByTestId('refused-move')).toHaveCount(1)
  await expect(page.locator('[data-testid="refused-move"][data-card="card-2"]')).toContainText(
    'at its limit',
  )

  await expect(card(page, 'card-1')).toHaveAttribute('data-column', 'doing')
  await expect(card(page, 'card-2')).toHaveAttribute('data-column', 'todo')
  await expect(column(page, 'doing').getByTestId('column-count')).toHaveText('2')
})

test('the column limit refuses a third card while online, and says so', async ({ page }) => {
  await page.goto(PAGE)

  await page.request.post('/api/app/kanban/moves', {
    data: { card: 'card-2', column: 'doing', index: 0 },
  })
  await expect(column(page, 'doing').getByTestId('column-count')).toHaveText('2')

  await dragOnto(page, 'card-1', card(page, 'card-2'))

  await expect(page.getByTestId('refusal')).toContainText('at its limit')
  await expect(card(page, 'card-1')).toHaveAttribute('data-column', 'todo')
})

test('done is one way, which is a different refusal from a full column', async ({ page }) => {
  await page.goto(PAGE)

  await dragOnto(page, 'card-5', card(page, 'card-1'))

  await expect(page.getByTestId('refusal')).toContainText('one way')
  await expect(card(page, 'card-5')).toHaveAttribute('data-column', 'done')
  await expect(page.getByTestId('board-rev')).toHaveText('0')
})

test('two workers have their own board and their own presence count', async ({
  page,
  playwright,
  baseURL,
}) => {
  await page.goto(PAGE)
  await expect(page.getByTestId('watchers')).toHaveText('1')

  const other = await playwright.request.newContext({
    baseURL,
    extraHTTPHeaders: { 'X-Playground-Session': 'kanban-two' },
  })
  await other.post('/api/app/kanban/moves', {
    data: { card: 'card-1', column: 'doing', index: 0 },
  })

  const theirs = await (await other.get('/api/app/kanban/board')).json()
  expect(theirs.board.watchers, 'a shared hub would make presence meaningless').toBe(0)
  await other.dispose()

  // A shared board would have moved this card, and a shared hub would have
  // told this page about it.
  await expect(card(page, 'card-1')).toHaveAttribute('data-column', 'todo')
  await expect(page.getByTestId('board-rev')).toHaveText('0')
  await expect(page.getByTestId('last-event')).toHaveText('presence')
})
