package hub

import (
	"context"
	"testing"
)

func TestPublishReachesEveryWatcher(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	first, stopFirst := h.Watch(ctx)
	second, stopSecond := h.Watch(ctx)
	defer stopFirst()
	defer stopSecond()

	if got := h.Watchers(); got != 2 {
		t.Fatalf("%d watchers, want 2", got)
	}

	h.Publish("moved", []byte(`{"card":1}`))

	for name, ch := range map[string]<-chan Message{"first": first, "second": second} {
		msg := <-ch
		if msg.Kind != "moved" || string(msg.Body) != `{"card":1}` {
			t.Errorf("%s watcher got %+v", name, msg)
		}
		if msg.Seq != 1 {
			t.Errorf("%s watcher got seq %d, want 1", name, msg.Seq)
		}
	}
}

func TestAStoppedWatcherStopsReceiving(t *testing.T) {
	h := New()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, stop := h.Watch(ctx)
	stop()

	if got := h.Watchers(); got != 0 {
		t.Fatalf("%d watchers after stopping the only one, want 0", got)
	}
	// Stop closes the channel, so a reader ranging over it is released rather
	// than left waiting for a message that is never coming.
	if _, open := <-ch; open {
		t.Error("channel was not closed by stop")
	}

	h.Publish("moved", nil) // must not panic on a send to a closed channel
}

func TestCancellingTheContextRemovesTheWatcher(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(context.Background())

	ch, stop := h.Watch(ctx)
	defer stop()
	cancel()

	// The removal happens on the context's goroutine, so wait on the effect
	// rather than on a duration: the close is what proves it ran.
	if _, open := <-ch; open {
		t.Error("cancelling the context left the watcher registered")
	}
}

func TestStopIsSafeToCallTwice(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, stop := h.Watch(ctx)

	stop()
	stop() // a double close would panic
}

func TestASlowWatcherIsDroppedRatherThanBlockingThePublisher(t *testing.T) {
	h := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, stop := h.Watch(ctx)
	defer stop()

	// Nothing drains ch, so everything past the buffer has nowhere to go. The
	// publisher must not wait for it.
	for range backlog + 5 {
		h.Publish("tick", nil)
	}

	if got := h.Dropped(); got != 5 {
		t.Errorf("dropped %d, want 5", got)
	}
	if got := h.Seq(); got != backlog+5 {
		t.Errorf("seq %d, want %d: a drop must still advance the sequence, or nothing reveals the gap", got, backlog+5)
	}
	if len(ch) != backlog {
		t.Errorf("buffered %d, want the full backlog of %d", len(ch), backlog)
	}
}

func TestAHubWithNoWatchersStillCounts(t *testing.T) {
	h := New()

	// A page that publishes before anyone is watching must not lose its place
	// in the sequence, or a later watcher cannot tell how much it missed.
	if got := h.Publish("moved", nil); got != 1 {
		t.Errorf("first publish returned seq %d, want 1", got)
	}
	if got := h.Dropped(); got != 0 {
		t.Errorf("dropped %d with nobody watching, want 0", got)
	}
}
