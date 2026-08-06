package clock

import (
	"testing"
	"time"
)

func TestFrozenClockDoesNotAdvance(t *testing.T) {
	c := New(Real{})
	c.Freeze()

	first := c.Now()
	time.Sleep(2 * time.Millisecond)

	if got := c.Now(); !got.Equal(first) {
		t.Fatalf("frozen clock advanced: %v then %v", first, got)
	}
}

func TestAdvanceMovesFrozenAndRunningClocks(t *testing.T) {
	base := Fixed(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	running := New(base)
	running.Advance(time.Hour)
	if got := running.Now(); got.Hour() != 1 {
		t.Fatalf("running clock: want hour 1, got %v", got)
	}

	frozen := New(base)
	frozen.Freeze()
	frozen.Advance(time.Hour)
	if got := frozen.Now(); got.Hour() != 1 {
		t.Fatalf("frozen clock: want hour 1, got %v", got)
	}
}

func TestUnfreezePreservesTheInstant(t *testing.T) {
	c := New(Real{})
	c.Advance(24 * time.Hour)
	c.Freeze()

	at := c.Now()
	c.Unfreeze()

	if drift := c.Now().Sub(at); drift < 0 || drift > time.Second {
		t.Fatalf("unfreeze lost the instant: drift %v", drift)
	}
	if c.Frozen() {
		t.Fatal("clock still reports frozen after unfreeze")
	}
}

func TestResetReturnsToBaseTime(t *testing.T) {
	base := Fixed(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	c := New(base)
	c.Set(time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC))
	c.Freeze()
	c.Reset()

	if got := c.Now(); !got.Equal(base.Now()) {
		t.Fatalf("want %v, got %v", base.Now(), got)
	}
}
