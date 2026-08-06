package app

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

const retriesStateKey = "retries"

func retries() challenge.Challenge {
	return challenge.Challenge{
		ID:       "retries",
		Title:    "An endpoint that fails until it does not",
		URL:      "/app/retries",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "L. Network & API",
		Summary: "An endpoint that refuses the first few calls with 503 and then answers " +
			"normally. The page retries with growing backoff and shows every attempt; a " +
			"second button asks once and gives up.",
		WhyHard: "The two buttons produce opposite results from the same server, so a test " +
			"that asserts on the outcome without knowing which one it pressed is asserting " +
			"on the retry policy rather than on the feature. Retrying also hides real " +
			"breakage: a suite that retries everything passes against a service that fails " +
			"two calls in three, and reports nothing. And the attempt counter only moves " +
			"between renders, so reading it immediately after the click reads the state " +
			"before the first attempt has been made.",
		Hint: "Decide what you are testing before you press anything. If it is the retry " +
			"policy, assert on the attempt count and on the final outcome together -- the " +
			"outcome alone cannot tell three attempts from one. If it is the feature, use " +
			"the endpoint's own controls to make the server reliable first, so a failure " +
			"means the feature broke rather than the network. The same first-N-then-succeed " +
			"behaviour is available for every route in the playground through the control " +
			"plane's failure rule.",
		Tags:     []string{"network", "retry", "backoff", "flaky-endpoint"},
		Concepts: []string{"retry with backoff", "retries hide breakage", "asserting on policy versus feature", "control-plane failure injection"},
		Selectors: []challenge.Selector{
			{TestID: "fail-first", Role: "spinbutton", Note: "How many calls the endpoint refuses before answering"},
			{TestID: "fetch-retrying", Role: "button", Note: "Asks, and keeps asking with growing backoff"},
			{TestID: "fetch-once", Role: "button", Note: "Asks once and reports whatever came back"},
			{TestID: "reset-endpoint", Role: "button", Note: "Puts the endpoint's refusal counter back to zero"},
			{TestID: "attempt-count", Note: "How many requests the page has made this run"},
			{TestID: "outcome", Note: "succeeded, failed, or in flight"},
			{TestID: "payload", Transient: true, Note: "The body, once a call succeeded"},
			{TestID: "attempt-row", Transient: true, Note: "One per attempt, with the status it got"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/app/retries/data", Note: "Refuses the first failFirst calls in this session, then answers"},
			{Method: http.MethodPost, Path: "/api/app/retries/reset", Note: "Resets the refusal counter"},
		},
		Controls: []challenge.Control{
			{Name: "failFirst", Kind: "query", Default: "3", Note: "Calls to refuse before answering, clamped to 0-20."},
		},
		Stability: challenge.Stable,
	}
}

// retryCounter is per session, so two workers exercising the same endpoint
// never consume each other's failures.
type retryCounter struct {
	mu    sync.Mutex
	calls int
}

func (c *retryCounter) next() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	return c.calls
}

func (c *retryCounter) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = 0
}

func retryCounterFor(sess *session.Session) *retryCounter {
	return session.Value(sess, retriesStateKey, func() *retryCounter { return &retryCounter{} })
}

func handleRetriesData(w http.ResponseWriter, r *http.Request) {
	sess := session.MustFromContext(r.Context())
	failFirst := httpx.QueryInt(r, "failFirst", 3, 0, 20)
	attempt := retryCounterFor(sess).next()

	if attempt <= failFirst {
		w.Header().Set("Retry-After", "1")
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]any{
			"status":    http.StatusServiceUnavailable,
			"error":     "not ready yet",
			"attempt":   attempt,
			"remaining": failFirst - attempt + 1,
		})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"attempt": attempt,
		"seed":    sess.RNG.Seed(),
		"message": "answered on attempt " + strconv.Itoa(attempt),
	})
}

func handleRetriesReset(w http.ResponseWriter, r *http.Request) {
	retryCounterFor(session.MustFromContext(r.Context())).reset()
	httpx.JSON(w, http.StatusOK, map[string]any{"reset": true})
}
