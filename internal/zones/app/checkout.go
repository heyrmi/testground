package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
	"github.com/heyrmi/testground/internal/shop"
)

func checkout() challenge.Challenge {
	return challenge.Challenge{
		ID:       "checkout",
		Title:    "Browse, cart, coupon, pay, confirm",
		URL:      "/app/checkout",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "V. Composite Scenarios",
		Summary: "The whole flow end to end: filter a catalogue, build a cart, apply a coupon, " +
			"pay with a card whose outcome you choose, and land on a confirmation carrying " +
			"an order number. The same cart is visible in the no-JavaScript zone, because " +
			"it lives on the server rather than in this page.",
		WhyHard: "Every step depends on the last, so a mistake in the second surfaces as a " +
			"confusing error in the fifth and the failure names the wrong thing. The " +
			"coupon is the trap: one earned by a large cart stops applying when the cart " +
			"shrinks under it, so a total asserted after adding a coupon and then removing " +
			"an item is a total nobody will be charged. Placing an order empties the cart, " +
			"which means a second checkout from a stale page fails rather than charging " +
			"twice -- correct behaviour that reads as a broken test. And the order number " +
			"exists nowhere until the confirmation renders, so a test that navigates away " +
			"first has lost the only record that the flow completed.",
		Hint: "Drive the steps through the API when you are setting up and through the page " +
			"when you are testing it; the endpoints are the same state either way, which " +
			"is what makes a long flow cheap to arrange. Re-read the totals after every " +
			"change rather than carrying a number forward, because the coupon is " +
			"revalidated on every read and the page says when it stopped applying. Choose " +
			"your payment outcome with the card number rather than discovering it. And " +
			"capture the order number before leaving the confirmation.",
		Tags:     []string{"composite", "e-commerce", "cart", "coupon", "payment"},
		Concepts: []string{"failures surface a step late", "revalidated totals", "state on the server not the page", "choosing the outcome rather than discovering it"},
		Selectors: []challenge.Selector{
			{TestID: "search", Role: "searchbox", Note: "Filters the catalogue by name"},
			{TestID: "category", Role: "combobox", Note: "Filters by category; combines with the search"},
			{TestID: "product", Note: "One catalogue entry; narrow by data-sku"},
			{TestID: "add-to-cart", Role: "button", Note: "Inside a product; disabled when out of stock"},
			{TestID: "cart-line", Transient: true, Note: "One per line in the cart; narrow by data-sku"},
			{TestID: "remove-line", Role: "button", Transient: true, Note: "Takes a line out"},
			{TestID: "cart-count", Note: "How many items are in the cart"},
			{TestID: "subtotal", Note: "In pounds and pence"},
			{TestID: "discount", Note: "What the coupon took off, if anything"},
			{TestID: "shipping", Note: "Free over seventy-five pounds"},
			{TestID: "total", Note: "What will actually be charged"},
			{TestID: "coupon-code", Role: "textbox", Note: "The codes are printed beside it"},
			{TestID: "apply-coupon", Role: "button", Note: "Validates against the cart as it stands"},
			{TestID: "coupon-note", Transient: true, Note: "Says why a coupon was refused, or that it stopped applying"},
			{TestID: "checkout-email", Role: "textbox", Transient: true, Note: "On the payment step"},
			{TestID: "checkout-card", Role: "textbox", Transient: true, Note: "Choose the outcome with the number"},
			{TestID: "place-order", Role: "button", Transient: true, Note: "Attempts payment"},
			{TestID: "payment-error", Transient: true, Note: "Why payment was refused"},
			{TestID: "order-number", Transient: true, Note: "Exists nowhere until this renders"},
			{TestID: "step", Note: "browse, pay or done"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/app/shop/catalogue", Note: "q and category filter it"},
			{Method: http.MethodGet, Path: "/api/app/shop/cart", Note: "Lines, totals and the coupon"},
			{Method: http.MethodPost, Path: "/api/app/shop/cart/items", Note: "Adds a quantity of a sku"},
			{Method: http.MethodDelete, Path: "/api/app/shop/cart/items/{sku}", Note: "Removes a line"},
			{Method: http.MethodPost, Path: "/api/app/shop/cart/coupon", Note: "Applies a code, or says why not"},
			{Method: http.MethodPost, Path: "/api/app/shop/checkout", Note: "Takes email and card, returns an order"},
			{Method: http.MethodGet, Path: "/api/app/shop/orders", Note: "Every order this session has placed"},
		},
		Stability: challenge.Stable,
	}
}

type cartResponse struct {
	Items   []shop.Item   `json:"items"`
	Totals  shop.Totals   `json:"totals"`
	Count   int           `json:"count"`
	Coupons []shop.Coupon `json:"coupons"`
}

func cartFor(r *http.Request) *shop.Cart {
	return shop.For(session.MustFromContext(r.Context()))
}

func writeCart(w http.ResponseWriter, cart *shop.Cart) {
	items := cart.Items()
	count := 0
	for _, item := range items {
		count += item.Quantity
	}

	httpx.JSON(w, http.StatusOK, cartResponse{
		Items:   items,
		Totals:  cart.Totals(),
		Count:   count,
		Coupons: shop.Coupons,
	})
}

func handleCatalogue(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"products":   shop.Search(r.URL.Query().Get("q"), r.URL.Query().Get("category")),
		"categories": shop.Categories(),
	})
}

func handleCart(w http.ResponseWriter, r *http.Request) { writeCart(w, cartFor(r)) }

func handleAddToCart(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SKU      string `json:"sku"`
		Quantity int    `json:"quantity"`
	}
	decodeJSON(r, &body)
	if body.Quantity == 0 {
		body.Quantity = 1
	}

	cart := cartFor(r)
	if err := cart.Add(body.SKU, body.Quantity); err != nil {
		// A refusal carries which refusal it was, so the page can say
		// something true rather than "could not add to cart".
		httpx.JSON(w, http.StatusConflict, map[string]any{
			"status": http.StatusConflict, "error": err.Error(), "sku": body.SKU,
		})
		return
	}
	writeCart(w, cart)
}

func handleRemoveFromCart(w http.ResponseWriter, r *http.Request) {
	cart := cartFor(r)
	if !cart.Remove(chi.URLParam(r, "sku")) {
		httpx.Fail(w, http.StatusNotFound, "that is not in the cart")
		return
	}
	writeCart(w, cart)
}

func handleCoupon(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	decodeJSON(r, &body)

	cart := cartFor(r)
	if body.Code == "" {
		cart.ClearCoupon()
		writeCart(w, cart)
		return
	}

	if err := cart.ApplyCoupon(body.Code); err != nil {
		httpx.JSON(w, http.StatusConflict, map[string]any{
			"status": http.StatusConflict, "error": err.Error(), "code": body.Code,
		})
		return
	}
	writeCart(w, cart)
}

func handleCheckout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
		Card  string `json:"card"`
	}
	decodeJSON(r, &body)

	sess := session.MustFromContext(r.Context())
	order, err := shop.For(sess).Place(sess.Clock.Now(), body.Email, body.Card)
	if err != nil {
		httpx.JSON(w, http.StatusPaymentRequired, map[string]any{
			"status": http.StatusPaymentRequired, "error": err.Error(),
		})
		return
	}
	httpx.JSON(w, http.StatusCreated, order)
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"orders": cartFor(r).Orders()})
}
