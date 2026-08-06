package fake

import (
	"math/rand/v2"
	"testing"
)

// The values a released challenge shows for a given seed are part of its
// stability contract. Rearranging the draws inside NewPerson would change
// every generated page for every existing user without changing a line of
// their tests, so the sequence is pinned here rather than trusted to review.
func TestDrawOrderIsPinned(t *testing.T) {
	stream := rand.New(rand.NewPCG(42, 99))

	want := []Person{
		{Name: "Jarrah Ferreira", Email: "jarrah.ferreira0@example.test", Status: "pending", Amount: "6256.43"},
	}

	for i, expected := range want {
		if got := NewPerson(stream, i); got != expected {
			t.Fatalf("draw %d changed:\n got  %+v\n want %+v", i, got, expected)
		}
	}
}

// NewPerson must consume exactly five values. Drawing a different number
// would shift every record after the first, so a challenge that generates a
// list would change from row two onwards.
func TestPersonConsumesFiveDraws(t *testing.T) {
	drawn := rand.New(rand.NewPCG(7, 7))
	NewPerson(drawn, 0)
	after := drawn.Uint64()

	skipped := rand.New(rand.NewPCG(7, 7))
	for range 5 {
		skipped.Uint64()
	}

	if got := skipped.Uint64(); got != after {
		t.Fatalf("NewPerson did not consume exactly five draws (next value %#x, want %#x)", after, got)
	}
}
