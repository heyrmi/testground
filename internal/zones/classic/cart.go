package classic

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/render"
	"github.com/heyrmi/testground/internal/session"
	"github.com/heyrmi/testground/internal/shop"
)

type cartView struct {
	Items   []shop.Item
	Totals  shop.Totals
	Count   int
	Orders  []shop.Order
	Message string
}

func cartFallback() page {
	meta := challenge.Challenge{
		ID:       "cart",
		Title:    "The same cart, with no JavaScript at all",
		URL:      "/classic/cart",
		Zone:     challenge.ZoneClassic,
		Tier:     challenge.T2,
		Category: "V. Composite Scenarios",
		Summary: "The cart built in the modern zone, rendered here from the server with form " +
			"posts and no script. Adding, removing and clearing all work, and every change " +
			"made here shows up over there.",
		WhyHard: "Two frontends over one cart is a thing most suites never test and most " +
			"applications quietly have -- a mobile client, an admin view, a second tab. It " +
			"is also the cheapest way to prove where state actually lives: if the cart " +
			"survives being read by a page that runs no script at all, it was never in the " +
			"component. The trap is assuming otherwise, and writing a test that sets up " +
			"through one interface and asserts through another without realising the two " +
			"can disagree about anything cached client-side.",
		Hint: "Use this page to set up state cheaply for tests of the modern flow, and to " +
			"check that a change made there really reached the server rather than only the " +
			"screen. The session header is what ties them together: the same header on " +
			"both is the same cart, and a different one is a different shop.",
		Tags:     []string{"composite", "cross-zone", "no-javascript", "cart"},
		Concepts: []string{"one state, two frontends", "proving state is server-side", "setting up through a second interface", "sessions tie the zones together"},
		Selectors: []challenge.Selector{
			{TestID: "cart-count", Note: "Items in the cart, shared with the modern zone"},
			{TestID: "cart-line", Transient: true, Note: "One per line; narrow by data-sku"},
			{TestID: "cart-empty", Note: "Stands in for the lines while there is nothing in the cart"},
			{TestID: "add-form", Note: "Adds a product without any script"},
			{TestID: "field-sku", Role: "combobox", Note: "Which product to add"},
			{TestID: "add-submit", Role: "button", Note: "Posts the addition"},
			{TestID: "remove-submit", Role: "button", Transient: true, Note: "Inside a line; removes it"},
			{TestID: "clear-cart", Role: "button", Note: "Empties the cart"},
			{TestID: "subtotal", Note: "In pounds and pence"},
			{TestID: "discount", Note: "What a coupon took off, if one is in force"},
			{TestID: "shipping", Note: "Free over seventy-five pounds"},
			{TestID: "total", Note: "What would be charged"},
			{TestID: "coupon", Transient: true, Note: "The code in force; there is no coupon field here, so it was applied in the modern zone"},
			{TestID: "order-row", Transient: true, Note: "One per order placed in this session, from either zone"},
			{TestID: "cart-message", Transient: true, Note: "Why the last action was refused"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodPost, Path: "/classic/cart", Note: "Adds a sku, answers 303"},
			{Method: http.MethodPost, Path: "/classic/cart/remove", Note: "Removes a line"},
			{Method: http.MethodPost, Path: "/classic/cart/clear", Note: "Empties it"},
		},
		Stability: challenge.Stable,
	}

	view := func(req *http.Request, message string) render.View {
		cart := shop.For(session.MustFromContext(req.Context()))
		items := cart.Items()

		count := 0
		for _, item := range items {
			count += item.Quantity
		}

		return render.View{
			Title:     meta.Title,
			Challenge: &meta,
			Data: cartView{
				Items:   items,
				Totals:  cart.Totals(),
				Count:   count,
				Orders:  cart.Orders(),
				Message: message,
			},
		}
	}

	return page{
		meta: meta,
		mount: func(r chi.Router, renderer *render.Renderer) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "classic/cart", view(req, ""))
			})

			r.Post("/", func(w http.ResponseWriter, req *http.Request) {
				req.ParseForm()
				cart := shop.For(session.MustFromContext(req.Context()))

				if err := cart.Add(req.PostFormValue("sku"), 1); err != nil {
					renderer.PageStatus(w, req, http.StatusConflict, "classic/cart", view(req, err.Error()))
					return
				}
				http.Redirect(w, req, meta.URL, http.StatusSeeOther)
			})

			r.Post("/remove", func(w http.ResponseWriter, req *http.Request) {
				req.ParseForm()
				shop.For(session.MustFromContext(req.Context())).Remove(req.PostFormValue("sku"))
				http.Redirect(w, req, meta.URL, http.StatusSeeOther)
			})

			r.Post("/clear", func(w http.ResponseWriter, req *http.Request) {
				shop.For(session.MustFromContext(req.Context())).Clear()
				http.Redirect(w, req, meta.URL, http.StatusSeeOther)
			})
		},
	}
}

// Products is the catalogue, for the template's select.
func (v cartView) Products() []shop.Product { return shop.Catalogue }

// Pounds renders whole cents as money, so the template does no arithmetic.
func (v cartView) Pounds(cents int) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	return sign + "£" + itoa(cents/100) + "." + pad(cents%100)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}
	return string(out)
}

func pad(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}
