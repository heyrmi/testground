// Package shop is the state behind the checkout composite.
//
// It lives outside any one zone because more than one drives it: the SPA runs
// the whole flow, and the no-JavaScript zone renders the same cart. Two
// frontends over one cart is the point -- it is what lets a test mutate in one
// place and observe in another, and what proves the state is on the server
// rather than in a component somewhere.
package shop

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/heyrmi/testground/internal/session"
)

// Key is the session state key the cart lives under.
const Key = "shop"

const (
	// ShippingCents is flat, and free once the subtotal passes the threshold,
	// which gives a coupon something to interact with.
	ShippingCents      = 499
	FreeShippingCents  = 7500
	minimumSpendCents  = 10000
	orderNumberPrefix  = "TG-"
	maxQuantityPerItem = 99
)

// Product is one catalogue entry.
type Product struct {
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	PriceCents int    `json:"priceCents"`
	Stock      int    `json:"stock"`
}

// Item is one line in a cart.
type Item struct {
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	PriceCents int    `json:"priceCents"`
	Quantity   int    `json:"quantity"`
	LineCents  int    `json:"lineCents"`
}

// Totals is what the cart adds up to. Every figure is in whole cents, because
// money in a float is a bug waiting for a rounding boundary.
type Totals struct {
	SubtotalCents int    `json:"subtotalCents"`
	DiscountCents int    `json:"discountCents"`
	ShippingCents int    `json:"shippingCents"`
	TotalCents    int    `json:"totalCents"`
	Coupon        string `json:"coupon,omitempty"`
	CouponNote    string `json:"couponNote,omitempty"`
}

// Cart is one session's basket and its accepted coupon.
type Cart struct {
	mu     sync.Mutex
	lines  map[string]*Item
	coupon string
	// dropped remembers that a coupon stopped qualifying. Without it the
	// explanation would appear only in the one response where the drop
	// happened and vanish on the next read, which is precisely the silent
	// failure this challenge exists to make visible.
	dropped string
	orders  []Order
}

// Order is a placed order. It exists only after payment succeeds, and its
// number is the only lasting record of the flow having completed.
type Order struct {
	ID       string    `json:"id"`
	Items    []Item    `json:"items"`
	Totals   Totals    `json:"totals"`
	Email    string    `json:"email"`
	PlacedAt time.Time `json:"placedAt"`
}

// For returns the session's cart, creating it on first use.
func For(sess *session.Session) *Cart {
	return session.Value(sess, Key, func() *Cart {
		return &Cart{lines: make(map[string]*Item)}
	})
}

// Catalogue is fixed rather than generated. A shop whose products change with
// the seed would make every assertion about a price seed-dependent, and the
// challenge is the flow rather than the data.
var Catalogue = []Product{
	{SKU: "TG-KEY-01", Name: "Mechanical keyboard", Category: "peripherals", PriceCents: 12900, Stock: 8},
	{SKU: "TG-MSE-01", Name: "Trackball mouse", Category: "peripherals", PriceCents: 6400, Stock: 14},
	{SKU: "TG-PAD-01", Name: "Desk mat", Category: "peripherals", PriceCents: 2400, Stock: 40},
	{SKU: "TG-HUB-01", Name: "Seven-port hub", Category: "peripherals", PriceCents: 4900, Stock: 0},
	{SKU: "TG-MON-01", Name: "Portable monitor", Category: "displays", PriceCents: 21900, Stock: 3},
	{SKU: "TG-ARM-01", Name: "Monitor arm", Category: "displays", PriceCents: 8900, Stock: 6},
	{SKU: "TG-CAB-01", Name: "Braided cable", Category: "cables", PriceCents: 1200, Stock: 120},
	{SKU: "TG-CAB-02", Name: "Right-angle adapter", Category: "cables", PriceCents: 900, Stock: 75},
	{SKU: "TG-BAG-01", Name: "Laptop sleeve", Category: "bags", PriceCents: 3500, Stock: 22},
	{SKU: "TG-BAG-02", Name: "Cable pouch", Category: "bags", PriceCents: 1500, Stock: 31},
}

// Find returns a catalogue entry by SKU.
func Find(sku string) (Product, bool) {
	for _, product := range Catalogue {
		if product.SKU == sku {
			return product, true
		}
	}
	return Product{}, false
}

// Search filters the catalogue. Both filters are optional and combine.
func Search(query, category string) []Product {
	query = strings.ToLower(strings.TrimSpace(query))
	out := make([]Product, 0, len(Catalogue))

	for _, product := range Catalogue {
		if category != "" && product.Category != category {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(product.Name), query) {
			continue
		}
		out = append(out, product)
	}
	return out
}

// Categories lists the catalogue's categories in a stable order.
func Categories() []string {
	seen := map[string]bool{}
	var out []string
	for _, product := range Catalogue {
		if !seen[product.Category] {
			seen[product.Category] = true
			out = append(out, product.Category)
		}
	}
	sort.Strings(out)
	return out
}

// ErrOutOfStock and friends are separate values because "we do not sell that",
// "we have none left" and "you asked for more than we have" are three
// different messages to a shopper and three different assertions to a test.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrNoSuchProduct  = Error("no such product")
	ErrOutOfStock     = Error("out of stock")
	ErrNotEnoughStock = Error("not enough stock for that quantity")
	ErrBadQuantity    = Error("quantity must be between 1 and 99")
)

// Add puts a quantity of a product in the cart, or raises the line if it is
// already there.
func (c *Cart) Add(sku string, quantity int) error {
	product, ok := Find(sku)
	if !ok {
		return ErrNoSuchProduct
	}
	if quantity < 1 || quantity > maxQuantityPerItem {
		return ErrBadQuantity
	}
	if product.Stock == 0 {
		return ErrOutOfStock
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	existing := c.lines[sku]
	wanted := quantity
	if existing != nil {
		wanted += existing.Quantity
	}
	if wanted > product.Stock {
		return ErrNotEnoughStock
	}

	if existing == nil {
		c.lines[sku] = &Item{SKU: sku, Name: product.Name, PriceCents: product.PriceCents}
		existing = c.lines[sku]
	}
	existing.Quantity = wanted
	existing.LineCents = existing.Quantity * existing.PriceCents
	return nil
}

// Remove takes a line out entirely.
func (c *Cart) Remove(sku string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.lines[sku]
	delete(c.lines, sku)
	return ok
}

// Items lists the cart in a stable order, so two reads render the same.
func (c *Cart) Items() []Item {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.itemsLocked()
}

func (c *Cart) itemsLocked() []Item {
	out := make([]Item, 0, len(c.lines))
	for _, line := range c.lines {
		out = append(out, *line)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SKU < out[j].SKU })
	return out
}

// Coupon describes one of the fixed discount codes.
type Coupon struct {
	Code    string `json:"code"`
	Note    string `json:"note"`
	Percent int    `json:"percent,omitempty"`
	// FreeShipping and MinimumCents make the codes interact with the totals
	// rather than all being the same rule with a different number.
	FreeShipping bool `json:"freeShipping,omitempty"`
	MinimumCents int  `json:"minimumCents,omitempty"`
	Expired      bool `json:"expired,omitempty"`
}

// Coupons are published so the flow is about applying one correctly rather
// than about guessing a code.
var Coupons = []Coupon{
	{Code: "SAVE10", Note: "Ten per cent off the subtotal", Percent: 10},
	{Code: "FREESHIP", Note: "No delivery charge", FreeShipping: true},
	{Code: "BIGSPEND", Note: "Twenty per cent off, over £100 only", Percent: 20, MinimumCents: minimumSpendCents},
	{Code: "LASTYEAR", Note: "Expired; always refused", Expired: true},
}

func findCoupon(code string) (Coupon, bool) {
	for _, coupon := range Coupons {
		if strings.EqualFold(coupon.Code, code) {
			return coupon, true
		}
	}
	return Coupon{}, false
}

const (
	ErrNoSuchCoupon = Error("no such coupon")
	ErrCouponSpent  = Error("that coupon has expired")
	ErrBelowMinimum = Error("the subtotal is below this coupon's minimum")
)

// ApplyCoupon validates a code against the cart as it stands now. It is
// re-validated on every read, which is what stops a coupon accepted on a large
// cart from surviving the cart shrinking underneath it.
func (c *Cart) ApplyCoupon(code string) error {
	coupon, ok := findCoupon(code)
	if !ok {
		return ErrNoSuchCoupon
	}
	if coupon.Expired {
		return ErrCouponSpent
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if coupon.MinimumCents > 0 && subtotal(c.itemsLocked()) < coupon.MinimumCents {
		return ErrBelowMinimum
	}
	c.coupon = coupon.Code
	c.dropped = ""
	return nil
}

// ClearCoupon removes any applied code.
func (c *Cart) ClearCoupon() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.coupon = ""
	c.dropped = ""
}

func subtotal(items []Item) int {
	total := 0
	for _, item := range items {
		total += item.LineCents
	}
	return total
}

// Totals recalculates from scratch every time, including whether the applied
// coupon still qualifies. A coupon accepted at one subtotal is silently
// dropped if the cart no longer earns it, and the note says so -- that is the
// composite's sharpest edge and it has to be observable.
func (c *Cart) Totals() Totals {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.totalsLocked()
}

func (c *Cart) totalsLocked() Totals {
	items := c.itemsLocked()
	totals := Totals{SubtotalCents: subtotal(items)}

	if totals.SubtotalCents >= FreeShippingCents {
		totals.CouponNote = "free delivery over £75"
	} else if totals.SubtotalCents > 0 {
		totals.ShippingCents = ShippingCents
	}

	if c.coupon != "" {
		coupon, _ := findCoupon(c.coupon)
		switch {
		case coupon.MinimumCents > 0 && totals.SubtotalCents < coupon.MinimumCents:
			// Accepted earlier, no longer earned. Dropping it silently would
			// be the bug; saying so, and keeping saying so, is the lesson.
			c.coupon = ""
			c.dropped = coupon.Code + " no longer applies at this subtotal"
		case coupon.FreeShipping:
			totals.ShippingCents = 0
			totals.Coupon = coupon.Code
			totals.CouponNote = coupon.Note
		default:
			totals.DiscountCents = totals.SubtotalCents * coupon.Percent / 100
			totals.Coupon = coupon.Code
			totals.CouponNote = coupon.Note
		}
	}

	if c.dropped != "" && totals.Coupon == "" {
		totals.CouponNote = c.dropped
	}

	totals.TotalCents = totals.SubtotalCents - totals.DiscountCents + totals.ShippingCents
	return totals
}

// Clear empties the cart and forgets the coupon, leaving placed orders alone.
func (c *Cart) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lines = make(map[string]*Item)
	c.coupon = ""
	c.dropped = ""
}

// Orders lists what this session has placed, newest last.
func (c *Cart) Orders() []Order {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Order(nil), c.orders...)
}

// Order finds a placed order by its number.
func (c *Cart) Order(id string) (Order, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, order := range c.orders {
		if order.ID == id {
			return order, true
		}
	}
	return Order{}, false
}

const (
	ErrEmptyCart  = Error("there is nothing in the cart")
	ErrNoEmail    = Error("an email address is required")
	ErrCardNumber = Error("that card number is not one this shop recognises")
)

// Card outcomes. Fixed numbers rather than random behaviour, so a test can
// choose which path it is exercising instead of discovering it.
const (
	CardAccepted  = "4242424242424242"
	CardDeclined  = "4000000000000002"
	CardNoFunds   = "4000000000009995"
	ErrDeclined   = Error("the card was declined")
	ErrNoFunds    = Error("the card has insufficient funds")
	ErrCardExpiry = Error("that expiry date is in the past")
)

// Place turns the cart into an order, or explains why it cannot.
func (c *Cart) Place(now time.Time, email, card string) (Order, error) {
	card = strings.ReplaceAll(card, " ", "")

	c.mu.Lock()
	defer c.mu.Unlock()

	items := c.itemsLocked()
	switch {
	case len(items) == 0:
		return Order{}, ErrEmptyCart
	case !strings.Contains(email, "@"):
		return Order{}, ErrNoEmail
	}

	switch card {
	case CardAccepted:
	case CardDeclined:
		return Order{}, ErrDeclined
	case CardNoFunds:
		return Order{}, ErrNoFunds
	default:
		return Order{}, ErrCardNumber
	}

	order := Order{
		ID:       fmt.Sprintf("%s%d", orderNumberPrefix, 1000+len(c.orders)+1),
		Items:    items,
		Totals:   c.totalsLocked(),
		Email:    email,
		PlacedAt: now,
	}
	c.orders = append(c.orders, order)

	// The cart is emptied by placing the order, which is what makes a second
	// checkout from a stale page fail rather than charge twice.
	c.lines = make(map[string]*Item)
	c.coupon = ""

	return order, nil
}
