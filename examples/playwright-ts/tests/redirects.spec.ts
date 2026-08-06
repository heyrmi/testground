import { expect, test } from './fixtures'

const PAGE = '/classic/redirects'

test('the browser follows the whole chain and lands somewhere else', async ({ page }) => {
  await page.goto(PAGE)
  await page.getByTestId('chain-link').click()

  // The URL asked for and the URL arrived at are different strings.
  await expect(page).toHaveURL(/\/classic\/redirects\/landed\?via=chain$/)
  await expect(page.getByTestId('landed-hops')).toHaveText('3')
})

test('the response chain records every hop', async ({ page }) => {
  const response = await page.goto('/classic/redirects/chain/1')

  expect(response?.status()).toBe(200)
  expect(response?.url()).toContain('via=chain')

  // Walking back up the redirect chain is how a framework tells you what it
  // followed on your behalf.
  let hops = 0
  for (let req = response?.request().redirectedFrom(); req; req = req.redirectedFrom()) hops++
  expect(hops).toBe(3)
})

test('307 and 308 keep the method; the others are free to rewrite it', async ({ page }) => {
  const methodAfter = async (code: number) => {
    const res = await page.request.post(`/classic/redirects/code/${code}`, { form: { x: '1' } })
    const body = await res.text()
    return body.match(/landed-method">([A-Z]+)/)?.[1]
  }

  expect(await methodAfter(301)).toBe('GET')
  expect(await methodAfter(302)).toBe('GET')
  expect(await methodAfter(303)).toBe('GET')
  expect(await methodAfter(307), 'a POST through 307 stays a POST').toBe('POST')
  expect(await methodAfter(308), 'a POST through 308 stays a POST').toBe('POST')
})

test('meta refresh is not a redirect, and a navigation wait resolves too early', async ({
  page,
}) => {
  const response = await page.goto('/classic/redirects/meta')

  // The first document loaded successfully. There was no Location header and
  // no redirect status, so nothing was followed.
  expect(response?.status()).toBe(200)
  expect(response?.request().redirectedFrom()).toBeNull()
  await expect(page.getByTestId('meta-waiting')).toBeVisible()

  // Waiting for something only the destination has is what actually works.
  await expect(page.getByTestId('landed')).toBeVisible()
  await expect(page.getByTestId('landed-via')).toHaveText('meta')
})
