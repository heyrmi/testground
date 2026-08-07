// Package hub broadcasts one session's changes to everyone watching them.
//
// Until now every realtime handler wrote only to the socket it had just
// accepted, which is enough for a page that talks to itself and not enough for
// anything a second viewer can see. A board that moves under you because
// someone else moved it, a presence list, an unread counter -- all of them need
// one writer's change to reach another reader's connection.
//
// A hub belongs to a session, never to the process. Two parallel workers each
// get their own, so a test that asserts "the other tab saw it" is asserting
// about its own two tabs and not about whatever another worker happened to be
// doing at the time.
package hub

import (
	"context"
	"sync"
)

// backlog is how many messages a watcher may fall behind by before the hub
// starts dropping them. A slow reader must not be able to block a writer:
// blocking would turn one stalled browser tab into a hung request for everyone
// sharing the session, which is a failure mode of this code rather than a
// lesson about anyone's application.
const backlog = 32

// Hub fans messages out to every current watcher. The zero value is not usable;
// call New.
type Hub struct {
	mu      sync.Mutex
	next    int
	seq     int
	watches map[int]chan Message
	dropped int
}

// Message is one broadcast. Seq counts every message the hub has accepted, so a
// watcher can tell "nothing has happened" from "I missed something": a gap in
// the sequence is the only evidence a drop leaves behind.
type Message struct {
	Seq  int    `json:"seq"`
	Kind string `json:"kind"`
	Body []byte `json:"body"`
}

// New returns an empty hub.
func New() *Hub {
	return &Hub{watches: make(map[int]chan Message)}
}

// Watch registers a watcher and returns its channel along with the function
// that removes it. The channel is closed by that function and never by the hub,
// so a reader ranging over it cannot be left waiting on a channel nobody will
// ever close.
//
// Cancelling ctx also removes the watcher, which is what stops a dropped
// connection from accumulating a channel nobody drains.
func (h *Hub) Watch(ctx context.Context) (<-chan Message, func()) {
	h.mu.Lock()
	id := h.next
	h.next++
	ch := make(chan Message, backlog)
	h.watches[id] = ch
	h.mu.Unlock()

	var once sync.Once
	stop := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.watches, id)
			h.mu.Unlock()
			close(ch)
		})
	}

	go func() {
		<-ctx.Done()
		stop()
	}()

	return ch, stop
}

// Publish sends to every watcher and reports the sequence number it was given.
//
// A watcher whose buffer is full misses the message rather than delaying the
// publisher. That is deliberate and it is observable: Dropped counts what was
// lost, and the gap in Seq tells the watcher it happened.
func (h *Hub) Publish(kind string, body []byte) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.seq++
	msg := Message{Seq: h.seq, Kind: kind, Body: body}
	for _, ch := range h.watches {
		select {
		case ch <- msg:
		default:
			h.dropped++
		}
	}
	return h.seq
}

// Watchers reports how many connections are currently listening, which is what
// a presence indicator is made of and what a test asserts against instead of
// sleeping to see whether a socket has gone.
func (h *Hub) Watchers() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.watches)
}

// Dropped reports how many deliveries were abandoned because a watcher was too
// far behind.
func (h *Hub) Dropped() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.dropped
}

// Seq reports the sequence number of the most recent message, so a page that
// reconnects can say what it last saw.
func (h *Hub) Seq() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.seq
}
