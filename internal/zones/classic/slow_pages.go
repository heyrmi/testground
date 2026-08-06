package classic

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/render"
)

const defaultTTFBMs = 3000

type slowView struct {
	DelayMs int
}

func slowPages() page {
	meta := challenge.Challenge{
		ID:       "slow-pages",
		Title:    "Slow to answer, and never finished",
		URL:      "/classic/slow-pages",
		Zone:     challenge.ZoneClassic,
		Tier:     challenge.T2,
		Category: "H. Windows, Tabs, Navigation",
		Summary: "One page whose server sits on the response for three seconds before " +
			"sending a byte, and one that arrives immediately but embeds a resource that " +
			"never answers, so the document is complete and the load event never fires.",
		WhyHard: "These two look identical from a test that only knows 'the page is not " +
			"ready yet', and they need opposite responses. The slow one is a server that " +
			"has not replied: nothing is in the DOM, and waiting is correct. The hanging " +
			"one has its entire document and every element a test wants, but a wait for " +
			"the load event sits there until the timeout, because one subresource is still " +
			"outstanding. Defaulting to the strictest ready state turns a page that was " +
			"usable at once into a guaranteed timeout.",
		Hint: "Match the wait to what you actually need. Waiting for the document rather " +
			"than for every subresource is enough to interact with the hanging page, and " +
			"is the difference between a two-second test and a thirty-second timeout. For " +
			"the slow one, the delay is a query parameter -- drive it fast in a suite and " +
			"slow when demonstrating.",
		Tags:     []string{"navigation", "waits", "timeouts", "load-events"},
		Concepts: []string{"time to first byte", "document ready versus load", "hanging subresources", "choosing a ready state"},
		Selectors: []challenge.Selector{
			{TestID: "slow-link", Role: "link", Note: "Goes to the page the server delays"},
			{TestID: "hanging-link", Role: "link", Note: "Goes to the page whose load event never fires"},
			{TestID: "slow-body", Transient: true, Note: "On the slow page, once the server finally answers"},
			{TestID: "slow-ms", Transient: true, Note: "The delay that page was served with"},
			{TestID: "hanging-body", Transient: true, Note: "On the hanging page; present immediately, while loading never completes"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/classic/slow-pages/slow?ms=3000", Note: "Waits before the first byte; ms is clamped to 0-30000"},
			{Method: http.MethodGet, Path: "/classic/slow-pages/hanging", Note: "Answers at once but embeds a resource that never does"},
			{Method: http.MethodGet, Path: "/classic/slow-pages/never", Note: "Never responds; held open until the client gives up"},
		},
		Controls: []challenge.Control{
			{Name: "ms", Kind: "query", Default: "3000", Note: "Milliseconds before the slow page starts answering, clamped to 0-30000."},
		},
		Stability: challenge.Stable,
	}

	return page{
		meta: meta,
		mount: func(r chi.Router, renderer *render.Renderer) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "classic/slow-pages", render.View{Title: meta.Title, Challenge: &meta})
			})

			r.Get("/slow", func(w http.ResponseWriter, req *http.Request) {
				delay := httpx.QueryInt(req, "ms", defaultTTFBMs, 0, 30_000)
				if err := stall(req.Context(), time.Duration(delay)*time.Millisecond); err != nil {
					return
				}
				renderer.Page(w, req, "classic/slow-loaded", render.View{
					Title: "Eventually",
					Data:  slowView{DelayMs: delay},
				})
			})

			r.Get("/hanging", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "classic/slow-hanging", render.View{Title: "Never finished"})
			})

			// Held open until the client disconnects. The handler blocks on the
			// request context rather than on a timer, so nothing leaks when the
			// browser or the test gives up.
			r.Get("/never", func(w http.ResponseWriter, req *http.Request) {
				<-req.Context().Done()
			})
		},
	}
}

func stall(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
