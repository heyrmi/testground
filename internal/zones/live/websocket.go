package live

import (
	"context"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/heyrmi/testground/internal/fake"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

// Message is what every socket here speaks.
type Message struct {
	// Seq is the order the server sent things in, which is not always the
	// order they arrive in.
	Seq  int    `json:"seq"`
	Kind string `json:"kind"`
	Text string `json:"text"`
	// At is the session clock, so a frozen clock produces identical payloads.
	At string `json:"at"`
}

// accept upgrades the connection and hands back the session that owns it.
//
// The session has to be read before the upgrade: once Accept returns, the
// request's context belongs to a hijacked connection and reading from it is
// undefined. The connection gets a context of its own instead.
func accept(w http.ResponseWriter, r *http.Request) (*websocket.Conn, *session.Session, context.Context, context.CancelFunc, bool) {
	sess := session.MustFromContext(r.Context())

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// The playground is reachable on whatever host the operator chose, and
		// the second origin embeds pages from the first, so cross-origin
		// sockets have to be allowed here. This is a local tool, not a service.
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, nil, nil, nil, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	return conn, sess, ctx, cancel, true
}

// handleEcho sends back whatever it is given, numbered.
func handleEcho(w http.ResponseWriter, r *http.Request) {
	conn, sess, ctx, cancel, ok := accept(w, r)
	if !ok {
		return
	}
	defer cancel()
	defer conn.CloseNow()

	_ = wsjson.Write(ctx, conn, Message{
		Seq: 0, Kind: "open", Text: "connected as " + string(sess.ID),
		At: sess.Clock.Now().UTC().Format(time.RFC3339Nano),
	})

	for seq := 1; ; seq++ {
		var incoming string
		if err := wsjson.Read(ctx, conn, &incoming); err != nil {
			return
		}
		if err := wsjson.Write(ctx, conn, Message{
			Seq: seq, Kind: "echo", Text: "echo: " + incoming,
			At: sess.Clock.Now().UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return
		}
	}
}

// handleTicker pushes without being asked, which is the shape of every live
// region: nothing the test did causes the next update.
func handleTicker(w http.ResponseWriter, r *http.Request) {
	every := httpx.QueryInt(r, "ms", 500, 20, 60_000)
	limit := httpx.QueryInt(r, "count", 0, 0, 10_000)

	conn, sess, ctx, cancel, ok := accept(w, r)
	if !ok {
		return
	}
	defer cancel()
	defer conn.CloseNow()

	// Reading is still required or control frames are never handled, and the
	// returned context ends when the client goes away.
	ctx = conn.CloseRead(ctx)

	stream := sess.RNG.Stream("live-ticker")
	ticker := time.NewTicker(time.Duration(every) * time.Millisecond)
	defer ticker.Stop()

	for seq := 1; limit == 0 || seq <= limit; seq++ {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		person := fake.NewPerson(stream, seq)
		if err := wsjson.Write(ctx, conn, Message{
			Seq: seq, Kind: "tick", Text: person.Name + " is " + person.Status,
			At: sess.Clock.Now().UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return
		}
	}
	conn.Close(websocket.StatusNormalClosure, "finished")
}

// handleFlaky drops the connection on purpose, so a client's reconnect
// behaviour has something to reconnect from.
func handleFlaky(w http.ResponseWriter, r *http.Request) {
	dropAfter := httpx.QueryInt(r, "dropAfterMs", 2000, 0, 60_000)
	every := httpx.QueryInt(r, "ms", 300, 20, 60_000)

	conn, sess, ctx, cancel, ok := accept(w, r)
	if !ok {
		return
	}
	defer cancel()
	defer conn.CloseNow()

	ctx = conn.CloseRead(ctx)
	deadline := time.After(time.Duration(dropAfter) * time.Millisecond)
	ticker := time.NewTicker(time.Duration(every) * time.Millisecond)
	defer ticker.Stop()

	for seq := 1; ; seq++ {
		select {
		case <-ctx.Done():
			return
		case <-deadline:
			// An abnormal closure rather than a polite one, because that is
			// what a dropped connection looks like and it is what a client's
			// reconnect logic has to cope with.
			conn.Close(websocket.StatusAbnormalClosure, "dropped on purpose")
			return
		case <-ticker.C:
		}

		if err := wsjson.Write(ctx, conn, Message{
			Seq: seq, Kind: "tick", Text: "before the drop",
			At: sess.Clock.Now().UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return
		}
	}
}

// handleShuffled sends numbered messages whose arrival order is not their
// sequence order, by delaying some of them.
func handleShuffled(w http.ResponseWriter, r *http.Request) {
	count := httpx.QueryInt(r, "count", 6, 2, 100)

	conn, sess, ctx, cancel, ok := accept(w, r)
	if !ok {
		return
	}
	defer cancel()
	defer conn.CloseNow()

	ctx = conn.CloseRead(ctx)

	// A fixed reordering rather than a random one: every even message is held
	// back until after the odd one that follows it, so the arrival order is
	// the same on every run and a test can assert on it exactly.
	order := make([]int, 0, count)
	for seq := 1; seq <= count; seq += 2 {
		if seq+1 <= count {
			order = append(order, seq+1)
		}
		order = append(order, seq)
	}

	for _, seq := range order {
		if err := wsjson.Write(ctx, conn, Message{
			Seq: seq, Kind: "out-of-order", Text: "message " + itoa(seq),
			At: sess.Clock.Now().UTC().Format(time.RFC3339Nano),
		}); err != nil {
			return
		}
	}
	conn.Close(websocket.StatusNormalClosure, "all sent")
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
