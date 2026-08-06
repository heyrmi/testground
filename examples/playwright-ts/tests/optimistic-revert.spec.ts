import { expect, test } from './fixtures'
import type { Page } from '@playwright/test'

const row = (page: Page, id: number) => page.locator(`[data-testid="task"][data-task-id="${id}"]`)

test('the endpoint says in advance which tasks the server will refuse', async ({ page }) => {
  const body = await (await page.request.get('/api/app/optimistic-revert/tasks')).json()

  expect(body.tasks.filter((t: { rejects: boolean }) => t.rejects).map((t: { id: number }) => t.id))
    .toEqual([3, 6])
})

test('a write the server accepts sticks', async ({ page }) => {
  await page.goto('/app/optimistic-revert?latencyMs=300')
  const task = row(page, 1)

  await task.getByTestId('task-toggle').click()
  await expect(task.getByTestId('task-saving')).toBeVisible()
  await expect(task.getByTestId('task-saving')).toHaveCount(0)

  await expect(task.getByTestId('task-state')).toHaveText('done')
})

test('a write the server refuses flips back', async ({ page }) => {
  await page.goto('/app/optimistic-revert?latencyMs=300')
  const task = row(page, 3)

  await task.getByTestId('task-toggle').click()

  // The optimistic value is real and visible; it is just not agreed yet.
  await expect(task.getByTestId('task-state')).toHaveText('done')

  // Waiting for the write to settle is what makes the assertion mean anything.
  await expect(task.getByTestId('task-saving')).toHaveCount(0)
  await expect(task.getByTestId('task-state')).toHaveText('todo')
  await expect(page.getByTestId('revert-notice')).toBeVisible()
  await expect(page.getByTestId('rejected-count')).toHaveText('1')
})

test('asserting on the click alone gives a green run for a broken write', async ({ page }) => {
  await page.goto('/app/optimistic-revert?latencyMs=1500')
  const task = row(page, 3)

  await task.getByTestId('task-toggle').click()

  // This passes. The feature did not work. This is the whole point of the
  // page: settle the write before believing what the DOM says.
  await expect(task.getByTestId('task-state')).toHaveText('done')

  await expect(task.getByTestId('task-state')).toHaveText('todo')
})

test('the server is the source of truth, whatever the DOM showed', async ({ page }) => {
  await page.goto('/app/optimistic-revert?latencyMs=0')

  await row(page, 3).getByTestId('task-toggle').click()
  await expect(page.getByTestId('rejected-count')).toHaveText('1')

  const body = await (await page.request.get('/api/app/optimistic-revert/tasks')).json()
  expect(body.tasks.find((t: { id: number }) => t.id === 3).done).toBe(false)
})
