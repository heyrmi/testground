package live

import (
	"net/http"

	"github.com/heyrmi/testground/internal/challenge"
)

func websocketBasics() page {
	return simplePage(challenge.Challenge{
		ID:       "websocket",
		Title:    "A socket that talks back, and one that just talks",
		URL:      "/live/websocket",
		Zone:     challenge.ZoneRealtime,
		Tier:     challenge.T2,
		Category: "M. Realtime",
		Summary: "Two WebSocket connections. One echoes whatever it is sent; the other pushes " +
			"a message every half second whether anyone asked or not. Both report their " +
			"connection state and keep a running log.",
		WhyHard: "Nothing the test does causes the ticker's next message, so there is no action " +
			"to wait after. Asserting on the log immediately reads whatever happened to " +
			"have arrived, which is a different amount every run, and sleeping long enough " +
			"to be safe makes every test that touches this the slowest one in the suite. " +
			"The echo has the opposite problem in disguise: the reply is a round trip, so " +
			"the click and the answer are two events and code that treats them as one is " +
			"reading the log before the server has spoken.",
		Hint: "Assert on a condition rather than on a moment: the message count reaching a " +
			"number, or a particular sequence number appearing, both settle on their own. " +
			"The ticker's interval is a query parameter, so drive it fast in a suite " +
			"rather than waiting at its demonstration speed. And read the connection state " +
			"the page publishes instead of inferring it from whether messages showed up.",
		Tags:     []string{"websocket", "realtime", "push", "live-region"},
		Concepts: []string{"updates with no triggering action", "waiting for a condition not a duration", "connection state as an assertion target", "round trips are two events"},
		Selectors: []challenge.Selector{
			{TestID: "echo-connect", Role: "button", Note: "Opens the echo socket"},
			{TestID: "echo-input", Role: "textbox", Note: "What to send"},
			{TestID: "echo-send", Role: "button", Note: "Sends it"},
			{TestID: "echo-state", Note: "closed, connecting or open"},
			{TestID: "echo-last", Note: "The most recent reply"},
			{TestID: "echo-count", Note: "How many replies have arrived"},
			{TestID: "ticker-connect", Role: "button", Note: "Opens the pushing socket"},
			{TestID: "ticker-stop", Role: "button", Note: "Closes it from the client side"},
			{TestID: "ticker-state", Note: "closed, connecting or open"},
			{TestID: "ticker-count", Note: "Messages received; this is the thing to wait on"},
			{TestID: "ticker-last-seq", Note: "The highest sequence number seen"},
			{TestID: "log-line", Transient: true, Note: "One per message, oldest first"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/live/echo", Note: "WebSocket; replies to each message"},
			{Method: http.MethodGet, Path: "/api/live/ticker", Note: "WebSocket; pushes every ms milliseconds, count limits it"},
		},
		Controls: []challenge.Control{
			{Name: "ms", Kind: "query", Default: "500", Note: "Milliseconds between pushes, clamped to 20-60000."},
			{Name: "count", Kind: "query", Default: "0", Note: "Stop after this many pushes; zero means never."},
		},
		Stability: challenge.Stable,
	})
}

func reconnects() page {
	return simplePage(challenge.Challenge{
		ID:       "reconnects",
		Title:    "A connection that drops, and messages that arrive out of order",
		URL:      "/live/reconnects",
		Zone:     challenge.ZoneRealtime,
		Tier:     challenge.T3,
		Category: "M. Realtime",
		Summary: "A socket that closes abnormally after two seconds, with a client that " +
			"reconnects on a growing backoff. Beside it, a socket that sends numbered " +
			"messages in an order that is not their numbering.",
		WhyHard: "A dropped socket does not announce itself to anything the test is holding. " +
			"The page keeps rendering the last thing it received, so a test that only " +
			"looks at content sees a page that is merely no longer changing -- which is " +
			"what a working page looks like between updates. Reconnecting makes it worse " +
			"before better: the state is briefly correct, then stale, then correct again, " +
			"and an assertion landing in the middle passes or fails on timing alone. The " +
			"out-of-order socket is the same lesson at message level: sequence numbers " +
			"arrive shuffled, so appending in arrival order renders them wrong.",
		Hint: "Assert on the connection state and the generation counter, not just on the " +
			"messages -- the page publishes both, and together they say whether what you " +
			"are reading came from the connection you think it did. For ordering, sort by " +
			"the sequence number rather than trusting arrival; the page shows both orders " +
			"side by side so the difference is visible rather than theoretical.",
		Tags:     []string{"websocket", "reconnect", "backoff", "out-of-order"},
		Concepts: []string{"a dropped socket looks like a quiet one", "reconnect generations", "sequence versus arrival order", "state that is briefly correct"},
		Selectors: []challenge.Selector{
			{TestID: "flaky-connect", Role: "button", Note: "Opens the socket that will be dropped"},
			{TestID: "flaky-state", Note: "closed, connecting, open or reconnecting"},
			{TestID: "flaky-drops", Note: "How many times the connection has been lost"},
			{TestID: "flaky-generation", Note: "Which connection the messages on screen came from"},
			{TestID: "flaky-count", Note: "Messages received across all connections"},
			{TestID: "shuffled-connect", Role: "button", Note: "Opens the out-of-order socket"},
			{TestID: "arrival-order", Note: "Sequence numbers in the order they arrived"},
			{TestID: "sorted-order", Note: "The same numbers, sorted, which is what a user should see"},
			{TestID: "shuffled-done", Transient: true, Note: "Present once every message has arrived"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/live/flaky", Note: "WebSocket; closes abnormally after dropAfterMs"},
			{Method: http.MethodGet, Path: "/api/live/shuffled", Note: "WebSocket; sends count messages in a fixed non-sequential order"},
		},
		Controls: []challenge.Control{
			{Name: "dropAfterMs", Kind: "query", Default: "2000", Note: "How long a connection survives, clamped to 0-60000."},
			{Name: "count", Kind: "query", Default: "6", Note: "Messages the shuffled socket sends, clamped to 2-100."},
		},
		Stability: challenge.Stable,
	})
}

func serverSentEvents() page {
	return simplePage(challenge.Challenge{
		ID:       "server-sent-events",
		Title:    "A stream that ends, one that stalls, and one that writes",
		URL:      "/live/server-sent-events",
		Zone:     challenge.ZoneRealtime,
		Tier:     challenge.T3,
		Category: "M. Realtime",
		Summary: "Three server-sent event streams: one that sends five updates and closes, " +
			"one that sends three and then goes silent with the connection still open, and " +
			"one that delivers a paragraph a word at a time.",
		WhyHard: "The stalled stream is the interesting one. It has not failed, it has not " +
			"finished, and it is not going to do either -- the connection is open, no error " +
			"is raised, and the page shows a partial result that looks exactly like a " +
			"complete one. A test that waits for the content it expects times out with a " +
			"message about the content rather than about the stream, and a test that waits " +
			"for the network to quieten agrees the page is done. The streaming text has " +
			"the same edge in miniature: every intermediate state is a correct-looking " +
			"sentence, and only the last one is the whole thing.",
		Hint: "The streams that finish say so with a terminating event, and the page reflects " +
			"that as a state rather than leaving you to infer it from silence. Assert on " +
			"that state. For the stalled one, decide what you actually expect -- if it is " +
			"'the stream stops early', assert the count stops at three rather than waiting " +
			"for a fourth that is never coming. For the text, wait for the done state, not " +
			"for a substring that was true three words ago.",
		Tags:     []string{"sse", "streaming", "stall", "tokens"},
		Concepts: []string{"a stalled stream is neither failed nor finished", "terminating events", "partial output looks complete", "quiet network is not done"},
		Selectors: []challenge.Selector{
			{TestID: "events-start", Role: "button", Note: "Opens the stream that ends"},
			{TestID: "events-state", Note: "idle, streaming or done"},
			{TestID: "events-count", Note: "Updates received"},
			{TestID: "stall-start", Role: "button", Note: "Opens the stream that goes quiet"},
			{TestID: "stall-state", Note: "Stays streaming forever; it never becomes done"},
			{TestID: "stall-count", Note: "Stops at three and never moves again"},
			{TestID: "stream-start", Role: "button", Note: "Opens the token stream"},
			{TestID: "stream-state", Note: "idle, streaming or done"},
			{TestID: "stream-text", Note: "Grows a word at a time; only complete once the state says done"},
			{TestID: "stream-tokens", Note: "Tokens received so far"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/live/events", Note: "Sends count updates then a done event"},
			{Method: http.MethodGet, Path: "/api/live/stall", Note: "Sends before updates then holds the connection open in silence"},
			{Method: http.MethodGet, Path: "/api/live/stream", Note: "One word per event, then a done event"},
		},
		Controls: []challenge.Control{
			{Name: "count", Kind: "query", Default: "5", Note: "Updates before the stream ends."},
			{Name: "before", Kind: "query", Default: "3", Note: "Updates before the stall begins."},
			{Name: "ms", Kind: "query", Default: "200", Note: "Milliseconds between events."},
		},
		Stability: challenge.Stable,
	})
}
