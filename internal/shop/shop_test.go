package shop

import (
	"testing"
	"time"

	"github.com/heyrmi/testground/internal/session"
)

func newCart(t *testing.T) *Cart {
	t.Helper()
	return For(session.NewStore(session.Options{Seed: 42}).Create())
}

func TestTwoSessionsHaveTwoCarts(t *testing.T) {
	store := session.NewStore(session.Options{Seed: 42})
	alice := For(store.Open("alice"))
	bob := For(store.Open("bob"))

	if err := alice.Add("TG-PAD-01", 2); err != nil {
		t.Fatalf("adding: %v", err)
	}
	if len(bob.Items()) != 0 {
		t.Fatal("bob's cart has alice's items in it")
	}
}

func TestStockRefusalsAreDistinguishable(t *testing.T) {
	cart := newCart(t)

	cases := []struct {
		name, sku string
		quantity  int
		want      error
	}{
		{"unknown product", "NOPE", 1, ErrNoSuchProduct},
		{"out of stock", "TG-HUB-01", 1, ErrOutOfStock},
		{"more than we have", "TG-MON-01", 99, ErrNotEnoughStock},
		{"nonsense quantity", "TG-PAD-01", 0, ErrBadQuantity},
	}

	for _, c := range cases {
		if got := cart.Add(c.sku, c.quantity); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestAddingTheSameProductRaisesTheLine(t *testing.T) {
	cart := newCart(t)
	cart.Add("TG-CAB-01", 2)
	cart.Add("TG-CAB-01", 3)

	items := cart.Items()
	if len(items) != 1 {
		t.Fatalf("%d lines, want 1", len(items))
	}
	if items[0].Quantity != 5 {
		t.Fatalf("quantity %d, want 5", items[0].Quantity)
	}
	if items[0].LineCents != 5*1200 {
		t.Fatalf("line total %d, want %d", items[0].LineCents, 5*1200)
	}
}

func TestShippingIsFreeOverTheThreshold(t *testing.T) {
	cart := newCart(t)
	cart.Add("TG-CAB-01", 1) // £12, below the threshold

	if got := cart.Totals().ShippingCents; got != ShippingCents {
		t.Fatalf("shipping %d, want %d", got, ShippingCents)
	}

	cart.Add("TG-MON-01", 1) // takes it well over
	if got := cart.Totals().ShippingCents; got != 0 {
		t.Fatalf("shipping %d over the threshold, want 0", got)
	}
}

func TestCouponRefusalsAreDistinguishable(t *testing.T) {
	cart := newCart(t)
	cart.Add("TG-PAD-01", 1) // £24, under the big-spend minimum

	if got := cart.ApplyCoupon("NOPE"); got != ErrNoSuchCoupon {
		t.Errorf("unknown code: %v", got)
	}
	if got := cart.ApplyCoupon("LASTYEAR"); got != ErrCouponSpent {
		t.Errorf("expired code: %v", got)
	}
	if got := cart.ApplyCoupon("BIGSPEND"); got != ErrBelowMinimum {
		t.Errorf("below minimum: %v", got)
	}
	if got := cart.ApplyCoupon("SAVE10"); got != nil {
		t.Errorf("valid code: %v", got)
	}
}

// The composite's sharpest edge: a coupon earned by a large cart must not
// survive the cart shrinking under it.
func TestACouponStopsApplyingWhenTheCartNoLongerEarnsIt(t *testing.T) {
	cart := newCart(t)
	cart.Add("TG-MON-01", 1) // £219, over the minimum

	if err := cart.ApplyCoupon("BIGSPEND"); err != nil {
		t.Fatalf("applying: %v", err)
	}
	if cart.Totals().DiscountCents == 0 {
		t.Fatal("the coupon was accepted and discounted nothing")
	}

	cart.Remove("TG-MON-01")
	cart.Add("TG-CAB-01", 1) // £12, far below

	totals := cart.Totals()
	if totals.DiscountCents != 0 {
		t.Fatalf("discount %d survived the cart shrinking", totals.DiscountCents)
	}
	if totals.CouponNote == "" {
		t.Fatal("the coupon was dropped silently, which is the bug rather than the lesson")
	}
}

func TestFreeShippingCouponBeatsTheShippingCharge(t *testing.T) {
	cart := newCart(t)
	cart.Add("TG-CAB-01", 1)
	cart.ApplyCoupon("FREESHIP")

	totals := cart.Totals()
	if totals.ShippingCents != 0 {
		t.Fatalf("shipping %d, want 0", totals.ShippingCents)
	}
	if totals.TotalCents != totals.SubtotalCents {
		t.Fatalf("total %d, want the subtotal %d", totals.TotalCents, totals.SubtotalCents)
	}
}

func TestPaymentOutcomesAreChosenRatherThanDiscovered(t *testing.T) {
	now := time.Now()

	for _, c := range []struct {
		name, card string
		want       error
	}{
		{"accepted", CardAccepted, nil},
		{"declined", CardDeclined, ErrDeclined},
		{"no funds", CardNoFunds, ErrNoFunds},
		{"unrecognised", "1234567812345678", ErrCardNumber},
	} {
		cart := newCart(t)
		cart.Add("TG-PAD-01", 1)

		_, err := cart.Place(now, "buyer@example.test", c.card)
		if err != c.want {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
}

func TestPlacingAnOrderEmptiesTheCart(t *testing.T) {
	cart := newCart(t)
	cart.Add("TG-PAD-01", 2)

	order, err := cart.Place(time.Now(), "buyer@example.test", CardAccepted)
	if err != nil {
		t.Fatalf("placing: %v", err)
	}
	if order.ID == "" {
		t.Fatal("the order has no number, and that is the only lasting record")
	}
	if len(cart.Items()) != 0 {
		t.Fatal("the cart survived checkout, so a stale page could charge twice")
	}

	// A second attempt has nothing to buy, which is the protection working.
	if _, err := cart.Place(time.Now(), "buyer@example.test", CardAccepted); err != ErrEmptyCart {
		t.Fatalf("second checkout: %v, want ErrEmptyCart", err)
	}
}

func TestAnOrderKeepsTheTotalsItWasPlacedAt(t *testing.T) {
	cart := newCart(t)
	cart.Add("TG-MON-01", 1)
	cart.ApplyCoupon("SAVE10")
	expected := cart.Totals()

	order, err := cart.Place(time.Now(), "buyer@example.test", CardAccepted)
	if err != nil {
		t.Fatalf("placing: %v", err)
	}
	if order.Totals.TotalCents != expected.TotalCents {
		t.Fatalf("order total %d, cart said %d", order.Totals.TotalCents, expected.TotalCents)
	}

	found, ok := cart.Order(order.ID)
	if !ok || found.Totals.TotalCents != expected.TotalCents {
		t.Fatal("the order could not be read back with the totals it was placed at")
	}
}

func TestSearchCombinesItsFilters(t *testing.T) {
	if got := len(Search("", "cables")); got != 2 {
		t.Errorf("cables: %d results, want 2", got)
	}
	if got := len(Search("cable", "")); got != 2 {
		t.Errorf("name match: %d results, want 2", got)
	}
	if got := len(Search("cable", "bags")); got != 1 {
		t.Errorf("both filters: %d results, want 1", got)
	}
	if got := len(Search("", "")); got != len(Catalogue) {
		t.Errorf("no filters: %d results, want the whole catalogue", got)
	}
}
