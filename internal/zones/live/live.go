// Package live serves Zone 6, the realtime zone.
//
// Vanilla TypeScript over WebSocket and server-sent events, with no framework
// between the test and the socket. Every failure here has the same shape: the
// page is not waiting for anything the test did, so nothing about the test's
// own actions tells it when to look.
package live

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/render"
)

const prefix = "/live"

type page struct {
	meta  challenge.Challenge
	mount func(chi.Router, *render.Renderer)
}

func pages() []page {
	return []page{
		websocketBasics(),
		reconnects(),
		serverSentEvents(),
	}
}

// Challenges declares every challenge this zone serves.
func Challenges() []challenge.Challenge {
	all := pages()
	out := make([]challenge.Challenge, 0, len(all))
	for _, p := range all {
		out = append(out, p.meta)
	}
	return out
}

// Pages serves the zone index and one page per challenge.
func Pages(renderer *render.Renderer) http.Handler {
	r := chi.NewRouter()
	zone, _ := challenge.LookupZone(challenge.ZoneRealtime)

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		renderer.Page(w, req, "zone", render.View{
			Title: zone.Title,
			Data:  render.ZoneView{Zone: zone, Challenges: Challenges()},
		})
	})

	for _, p := range pages() {
		sub := chi.NewRouter()
		p.mount(sub, renderer)
		r.Mount(strings.TrimPrefix(p.meta.URL, prefix), sub)
	}

	return r
}

// API serves the sockets and streams, mounted at /api/live.
func API() http.Handler {
	r := chi.NewRouter()
	r.Get("/echo", handleEcho)
	r.Get("/ticker", handleTicker)
	r.Get("/flaky", handleFlaky)
	r.Get("/shuffled", handleShuffled)
	r.Get("/events", handleEvents)
	r.Get("/stall", handleStall)
	r.Get("/stream", handleStream)
	return r
}

// simplePage is the shape of a Zone 6 challenge: one template, all the
// interesting behaviour on a socket.
func simplePage(meta challenge.Challenge) page {
	return page{
		meta: meta,
		mount: func(r chi.Router, renderer *render.Renderer) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "live/"+meta.ID, render.View{
					Title:     meta.Title,
					Challenge: &meta,
				})
			})
		},
	}
}
