package app

import (
	"net/http"
	"time"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

func races() challenge.Challenge {
	return challenge.Challenge{
		ID:       "request-races",
		Title:    "Responses that arrive in the wrong order",
		URL:      "/app/request-races",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "L. Network & API",
		Summary: "Two searches fired half a second apart, where the first one takes longer " +
			"than the second. The page shows the result twice: once written by whichever " +
			"response arrived last, and once by a handler that ignores anything it did not " +
			"ask for most recently. Beside them, a three-step waterfall where each request " +
			"waits for the one before it.",
		WhyHard: "The naive panel settles on the older search, because that response arrived " +
			"second. Everything about the page looks finished -- no spinner, no error, a " +
			"plausible result -- and it is showing the answer to a question the user has " +
			"already moved on from. A test that waits for the network to go quiet and then " +
			"asserts will agree with it. The waterfall is the other half: three dependent " +
			"requests take as long as all of them added together, and a test that waits for " +
			"the first response to land reads a page that is still two requests from ready.",
		Hint: "Assert that the result matches the last thing requested, not merely that a " +
			"result arrived -- those differ exactly when this bug is present. The guarded " +
			"panel is what correct looks like, so comparing the two tells you which " +
			"behaviour you are looking at. For the waterfall, wait for the final step " +
			"rather than for the network, since each step only starts when the one before " +
			"it finished.",
		Tags:     []string{"network", "race", "out-of-order", "waterfall", "cancellation"},
		Concepts: []string{"stale responses overwrite fresh ones", "quiet network is not done", "request waterfalls", "ignoring superseded requests"},
		Selectors: []challenge.Selector{
			{TestID: "run-race", Role: "button", Note: "Fires the slow search, then the fast one"},
			{TestID: "naive-result", Note: "Written by whichever response arrived last"},
			{TestID: "guarded-result", Note: "Written only by the most recently requested search"},
			{TestID: "run-waterfall", Role: "button", Note: "Runs three dependent requests in sequence"},
			{TestID: "waterfall-step", Transient: true, Note: "One per completed step, in the order they finished"},
			{TestID: "waterfall-total", Note: "Milliseconds the whole chain took"},
			{TestID: "waterfall-done", Transient: true, Note: "Present only once every step has finished"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/app/races/search", Note: "Echoes q after waiting ms milliseconds"},
			{Method: http.MethodGet, Path: "/api/app/races/step", Note: "Echoes a step name after waiting ms milliseconds"},
		},
		Controls: []challenge.Control{
			{Name: "ms", Kind: "query", Default: "0", Note: "Milliseconds to wait before answering, clamped to 0-30000."},
		},
		Stability: challenge.Stable,
	}
}

func handleRacesEcho(w http.ResponseWriter, r *http.Request) {
	delay := httpx.QueryInt(r, "ms", 0, 0, 30_000)
	if err := sleep(r.Context(), time.Duration(delay)*time.Millisecond); err != nil {
		return // the client cancelled, which is the point of one of these
	}

	sess := session.MustFromContext(r.Context())
	httpx.JSON(w, http.StatusOK, map[string]any{
		"query":   r.URL.Query().Get("q"),
		"step":    r.URL.Query().Get("step"),
		"ms":      delay,
		"session": string(sess.ID),
	})
}
