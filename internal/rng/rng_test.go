package rng

import "testing"

func draw(s *Source, name string, n int) []uint64 {
	r := s.Stream(name)
	out := make([]uint64, n)
	for i := range out {
		out[i] = r.Uint64()
	}
	return out
}

func equal(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSameSeedAndNameReplaysTheSameSequence(t *testing.T) {
	a := draw(New(42), "virtual-list", 8)
	b := draw(New(42), "virtual-list", 8)

	if !equal(a, b) {
		t.Fatalf("same seed produced different sequences: %v vs %v", a, b)
	}
}

func TestKnownSeedProducesPinnedValues(t *testing.T) {
	// Pinned so a change to the derivation shows up as a failing test rather
	// than as silently different challenge content for every existing user.
	want := []uint64{
		0xb43d2514245f9289,
		0xef838c8566714c48,
		0x7fa7c3c37f9ab976,
	}
	if got := draw(New(DefaultSeed), "pinned", 3); !equal(got, want) {
		t.Fatalf("derivation changed:\n got  %#v\n want %#v", got, want)
	}
}

func TestStreamsAreIndependentOfEachOther(t *testing.T) {
	s := New(42)

	// Drain an unrelated stream first; the target stream must not shift.
	before := draw(s, "target", 4)
	draw(s, "noise", 1000)
	after := draw(s, "target", 4)

	if !equal(before, after) {
		t.Fatalf("stream shifted after unrelated draws: %v vs %v", before, after)
	}
}

func TestDifferentNamesProduceDifferentSequences(t *testing.T) {
	s := New(42)
	if equal(draw(s, "a", 4), draw(s, "b", 4)) {
		t.Fatal("distinct stream names collided")
	}
}

func TestDifferentSeedsProduceDifferentSequences(t *testing.T) {
	if equal(draw(New(1), "rows", 4), draw(New(2), "rows", 4)) {
		t.Fatal("adjacent seeds collided")
	}
}

func TestReseedChangesSubsequentStreams(t *testing.T) {
	s := New(42)
	before := draw(s, "rows", 4)
	s.Reseed(43)

	if s.Seed() != 43 {
		t.Fatalf("want seed 43, got %d", s.Seed())
	}
	if equal(before, draw(s, "rows", 4)) {
		t.Fatal("reseed did not change the stream")
	}
}
