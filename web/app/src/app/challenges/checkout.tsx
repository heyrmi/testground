import { useCallback, useEffect, useState } from 'react'
import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/checkout',
  component: Checkout,
})

interface Product {
  sku: string
  name: string
  category: string
  priceCents: number
  stock: number
}

interface Line {
  sku: string
  name: string
  priceCents: number
  quantity: number
  lineCents: number
}

interface Totals {
  subtotalCents: number
  discountCents: number
  shippingCents: number
  totalCents: number
  coupon?: string
  couponNote?: string
}

interface CartState {
  items: Line[]
  totals: Totals
  count: number
  coupons: { code: string; note: string }[]
}

const money = (cents: number) =>
  (cents / 100).toLocaleString('en-GB', { style: 'currency', currency: 'GBP' })

function Checkout() {
  const [products, setProducts] = useState<Product[]>([])
  const [categories, setCategories] = useState<string[]>([])
  const [cart, setCart] = useState<CartState | null>(null)
  const [query, setQuery] = useState('')
  const [category, setCategory] = useState('')
  const [note, setNote] = useState('')
  const [coupon, setCoupon] = useState('')
  const [step, setStep] = useState<'browse' | 'pay' | 'done'>('browse')
  const [email, setEmail] = useState('buyer@example.test')
  const [card, setCard] = useState('4242 4242 4242 4242')
  const [paymentError, setPaymentError] = useState('')
  const [orderNumber, setOrderNumber] = useState('')

  const refreshCart = useCallback(async () => {
    setCart((await (await fetch('/api/app/shop/cart')).json()) as CartState)
  }, [])

  useEffect(() => {
    const params = new URLSearchParams({ q: query, category })
    fetch(`/api/app/shop/catalogue?${params}`)
      .then((res) => res.json() as Promise<{ products: Product[]; categories: string[] }>)
      .then((body) => {
        setProducts(body.products)
        setCategories(body.categories)
      })
  }, [query, category])

  useEffect(() => {
    void refreshCart()
  }, [refreshCart])

  async function post(path: string, body?: unknown) {
    const res = await fetch(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
    return { res, body: (await res.json()) as Record<string, unknown> }
  }

  async function add(sku: string) {
    const { res, body } = await post('/api/app/shop/cart/items', { sku, quantity: 1 })
    if (!res.ok) {
      setNote(String(body.error))
      return
    }
    setNote('')
    setCart(body as unknown as CartState)
  }

  async function remove(sku: string) {
    const res = await fetch(`/api/app/shop/cart/items/${sku}`, { method: 'DELETE' })
    if (res.ok) setCart((await res.json()) as CartState)
  }

  async function applyCoupon() {
    const { res, body } = await post('/api/app/shop/cart/coupon', { code: coupon })
    if (!res.ok) {
      setNote(String(body.error))
      return
    }
    setNote('')
    setCart(body as unknown as CartState)
  }

  async function placeOrder() {
    const { res, body } = await post('/api/app/shop/checkout', { email, card })
    if (!res.ok) {
      setPaymentError(String(body.error))
      return
    }
    setPaymentError('')
    setOrderNumber(String(body.id))
    setStep('done')
    void refreshCart()
  }

  const totals = cart?.totals

  return (
    <ChallengePage id="checkout">
      <p className="stage__label">
        step <b data-testid="step">{step}</b>
      </p>

      {step === 'done' ? (
        <section>
          <h2 className="m-0 text-lg font-semibold">Thank you</h2>
          <p className="mt-2">
            Your order number is{' '}
            <b className="font-mono" data-testid="order-number">
              {orderNumber}
            </b>
            . It exists nowhere else, so read it before leaving this page.
          </p>
          <button className="mt-4" data-testid="shop-again" onClick={() => setStep('browse')}>
            Buy something else
          </button>
        </section>
      ) : (
        <div className="grid gap-8 md:grid-cols-[1fr_20rem]">
          <section>
            {step === 'browse' ? (
              <>
                <div className="flex flex-wrap gap-2">
                  <input
                    data-testid="search"
                    type="search"
                    placeholder="filter by name"
                    className="rounded-md border border-line bg-sunken px-2 py-1 text-sm"
                    value={query}
                    onChange={(event) => setQuery(event.target.value)}
                  />
                  <select
                    data-testid="category"
                    className="rounded-md border border-line bg-sunken px-2 py-1 text-sm"
                    value={category}
                    onChange={(event) => setCategory(event.target.value)}
                  >
                    <option value="">every category</option>
                    {categories.map((name) => (
                      <option key={name} value={name}>
                        {name}
                      </option>
                    ))}
                  </select>
                </div>

                <ul className="mt-4 list-none p-0">
                  {products.map((product) => (
                    <li
                      key={product.sku}
                      data-testid="product"
                      data-sku={product.sku}
                      className="flex items-center gap-3 border-b border-line py-2 last:border-b-0"
                    >
                      <span className="flex-1">{product.name}</span>
                      <span className="font-mono text-sm">{money(product.priceCents)}</span>
                      <button
                        data-testid="add-to-cart"
                        disabled={product.stock === 0}
                        onClick={() => add(product.sku)}
                      >
                        {product.stock === 0 ? 'Out of stock' : 'Add'}
                      </button>
                    </li>
                  ))}
                </ul>
              </>
            ) : (
              <>
                <h2 className="m-0 text-lg font-semibold">Payment</h2>
                <div className="mt-3 flex flex-col gap-3">
                  <label className="text-sm">
                    Email
                    <input
                      data-testid="checkout-email"
                      className="mt-1 block w-72 rounded-md border border-line bg-sunken px-2 py-1"
                      value={email}
                      onChange={(event) => setEmail(event.target.value)}
                    />
                  </label>
                  <label className="text-sm">
                    Card number
                    <input
                      data-testid="checkout-card"
                      className="mt-1 block w-72 rounded-md border border-line bg-sunken px-2 py-1 font-mono"
                      value={card}
                      onChange={(event) => setCard(event.target.value)}
                    />
                  </label>
                </div>

                <table className="results mt-4">
                  <tbody>
                    <tr><th>4242 4242 4242 4242</th><td>accepted</td></tr>
                    <tr><th>4000 0000 0000 0002</th><td>declined</td></tr>
                    <tr><th>4000 0000 0000 9995</th><td>insufficient funds</td></tr>
                  </tbody>
                </table>

                {paymentError && (
                  <p className="mt-3 text-sm" data-testid="payment-error">
                    {paymentError}
                  </p>
                )}

                <div className="mt-4 flex gap-2">
                  <button className="primary" data-testid="place-order" onClick={placeOrder}>
                    Pay {totals ? money(totals.totalCents) : ''}
                  </button>
                  <button data-testid="back-to-browse" onClick={() => setStep('browse')}>
                    Back
                  </button>
                </div>
              </>
            )}
          </section>

          <aside className="rounded-lg border border-line p-4">
            <h2 className="m-0 text-sm font-semibold uppercase tracking-wide text-muted">
              Cart · <span data-testid="cart-count">{cart?.count ?? 0}</span>
            </h2>

            <ul className="mt-3 list-none p-0 text-sm">
              {cart?.items.map((line) => (
                <li
                  key={line.sku}
                  data-testid="cart-line"
                  data-sku={line.sku}
                  className="flex items-center gap-2 border-b border-line py-1 last:border-b-0"
                >
                  <span className="flex-1">
                    {line.name} × {line.quantity}
                  </span>
                  <span className="font-mono">{money(line.lineCents)}</span>
                  <button data-testid="remove-line" onClick={() => remove(line.sku)}>
                    ×
                  </button>
                </li>
              ))}
            </ul>

            <div className="mt-3 flex gap-1">
              <input
                data-testid="coupon-code"
                placeholder="coupon"
                className="w-full rounded-md border border-line bg-sunken px-2 py-1 text-sm"
                value={coupon}
                onChange={(event) => setCoupon(event.target.value)}
              />
              <button data-testid="apply-coupon" onClick={applyCoupon}>
                Apply
              </button>
            </div>

            {(note || totals?.couponNote) && (
              <p className="mt-2 text-xs text-muted" data-testid="coupon-note">
                {note || totals?.couponNote}
              </p>
            )}

            <dl className="mt-3 grid grid-cols-[1fr_auto] gap-y-1 font-mono text-sm">
              <dt className="text-muted">subtotal</dt>
              <dd data-testid="subtotal">{money(totals?.subtotalCents ?? 0)}</dd>
              <dt className="text-muted">discount</dt>
              <dd data-testid="discount">{money(totals?.discountCents ?? 0)}</dd>
              <dt className="text-muted">shipping</dt>
              <dd data-testid="shipping">{money(totals?.shippingCents ?? 0)}</dd>
              <dt className="font-semibold">total</dt>
              <dd className="font-semibold" data-testid="total">
                {money(totals?.totalCents ?? 0)}
              </dd>
            </dl>

            {step === 'browse' && (
              <button
                className="primary mt-4 w-full"
                data-testid="go-to-payment"
                disabled={!cart?.count}
                onClick={() => setStep('pay')}
              >
                Checkout
              </button>
            )}

            <p className="mt-3 text-xs text-muted">
              The same cart is at <code>/classic/cart</code> with no JavaScript at all.
            </p>
          </aside>
        </div>
      )}
    </ChallengePage>
  )
}
