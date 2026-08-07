package live

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/heyrmi/testground/internal/fake"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

// openStream sets the headers a server-sent event stream needs and returns a
// flusher, since nothing arrives until each event is pushed out.
func openStream(w http.ResponseWriter) (*http.ResponseController, bool) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	// Without this a proxy may buffer the whole stream and deliver it at the
	// end, which turns a live feed into a slow page load.
	w.Header().Set("X-Accel-Buffering", "no")

	controller := http.NewResponseController(w)
	// The first flush implicitly commits a 200, so the status has to go out
	// ahead of it. The other order left every connection logging a superfluous
	// response.WriteHeader call, with the explicit status silently discarded.
	w.WriteHeader(http.StatusOK)

	// Flushed immediately so the headers above reach the client before the
	// first event does, which is what stops a buffering proxy from holding the
	// whole stream back. A writer chain with no flusher underneath also shows
	// up here, and the status is already committed by then, so the only way
	// left to report it is in the body the client is reading.
	if err := controller.Flush(); err != nil {
		fmt.Fprint(w, "event: error\ndata: streaming is not supported here\n\n")
		return nil, false
	}
	return controller, true
}

func send(w http.ResponseWriter, controller *http.ResponseController, event, data string) error {
	if event != "" {
		fmt.Fprintf(w, "event: %s\n", event)
	}
	for _, line := range strings.Split(data, "\n") {
		fmt.Fprintf(w, "data: %s\n", line)
	}
	fmt.Fprint(w, "\n")
	return controller.Flush()
}

// handleEvents sends a fixed number of events and then closes, so a test has a
// stream with a definite end to assert on.
func handleEvents(w http.ResponseWriter, r *http.Request) {
	count := httpx.QueryInt(r, "count", 5, 1, 1000)
	every := httpx.QueryInt(r, "ms", 200, 0, 60_000)
	sess := session.MustFromContext(r.Context())

	controller, ok := openStream(w)
	if !ok {
		return
	}

	stream := sess.RNG.Stream("live-events")
	for seq := 1; seq <= count; seq++ {
		if !wait(r, every) {
			return
		}
		person := fake.NewPerson(stream, seq)
		if send(w, controller, "update", fmt.Sprintf(`{"seq":%d,"text":%q}`, seq, person.Name)) != nil {
			return
		}
	}

	send(w, controller, "done", fmt.Sprintf(`{"count":%d}`, count))
}

// handleStall sends a few events and then goes quiet with the connection
// still open, which is the failure that looks exactly like success.
func handleStall(w http.ResponseWriter, r *http.Request) {
	before := httpx.QueryInt(r, "before", 3, 0, 100)
	every := httpx.QueryInt(r, "ms", 150, 0, 60_000)

	controller, ok := openStream(w)
	if !ok {
		return
	}

	for seq := 1; seq <= before; seq++ {
		if !wait(r, every) {
			return
		}
		if send(w, controller, "update", fmt.Sprintf(`{"seq":%d,"text":"still going"}`, seq)) != nil {
			return
		}
	}

	// No close, no error, no final event. The socket stays open and silent
	// until the client gives up, which is the only thing that ends this.
	<-r.Context().Done()
}

// handleStream sends one word at a time, the way a token stream arrives.
func handleStream(w http.ResponseWriter, r *http.Request) {
	every := httpx.QueryInt(r, "ms", 60, 0, 5000)

	controller, ok := openStream(w)
	if !ok {
		return
	}

	words := strings.Fields(streamedText)
	for i, word := range words {
		if !wait(r, every) {
			return
		}
		if send(w, controller, "token", fmt.Sprintf(`{"index":%d,"token":%q}`, i, word+" ")) != nil {
			return
		}
	}

	send(w, controller, "done", fmt.Sprintf(`{"tokens":%d}`, len(words)))
}

// The text is fixed rather than generated, so a test can assert on the whole
// thing once it has finished arriving.
const streamedText = "A stream is not a page. " +
	"It arrives in pieces, each one a complete and correct view of nothing in particular, " +
	"and the only moment the whole thing is true is after the last piece lands."

func wait(r *http.Request, ms int) bool {
	if ms <= 0 {
		return r.Context().Err() == nil
	}
	timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
	defer timer.Stop()

	select {
	case <-r.Context().Done():
		return false
	case <-timer.C:
		return true
	}
}
