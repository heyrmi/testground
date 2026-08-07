// Package hub broadcasts one session's changes to everyone watching them.
//
// A hub belongs to a session, never to the process, so a test asserting that
// the other tab saw something is asserting about its own two tabs and not about
// whatever another worker was doing at the time.
package hub

import (
	"context"
	"sync"
)

// backlog bounds how far a watcher may fall behind. A slow reader must not
// block a writer: that would turn one stalled tab into a hung request for
// everyone sharing the session.
const backlog = 32

// Hub fans messages out to every current watcher. Use New; the zero value is
// not usable.
type Hub struct {
	mu      sync.Mutex
	next    int
	seq     int
	watches map[int]chan Message
}

// Message is one broadcast. A gap in Seq is the only evidence a drop leaves
// behind, which is what lets a watcher tell silence from a missed message.
type Message struct {
	Seq  int    `json:"seq"`
	Kind string `json:"kind"`
	Body []byte `json:"body"`
}

// New returns an empty hub.
func New() *Hub {
	return &Hub{watches: make(map[int]chan Message)}
}

// Watch registers a watcher, returning its channel and the function that
// removes it. Only that function closes the channel, so a reader ranging over
// it is always released. Cancelling ctx removes the watcher too, which is what
// stops a dropped connection leaving a channel nobody drains.
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
// A watcher whose buffer is full misses the message rather than delaying the
// publisher; the sequence still advances, so the gap remains visible.
func (h *Hub) Publish(kind string, body []byte) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.seq++
	msg := Message{Seq: h.seq, Kind: kind, Body: body}
	for _, ch := range h.watches {
		select {
		case ch <- msg:
		default:
		}
	}
	return h.seq
}

// Watchers reports how many connections are listening, which is what a
// presence indicator is made of and what a test asserts on instead of sleeping
// to see whether a socket has gone.
func (h *Hub) Watchers() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.watches)
}

// Seq reports the most recent sequence number, so a page that reconnects can
// say what it last saw.
func (h *Hub) Seq() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.seq
}
