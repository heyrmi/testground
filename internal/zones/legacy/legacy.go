// Package legacy serves Zone 2, the jQuery era.
//
// It wears Bootstrap 3 and drives everything through jQuery on purpose. A
// large share of the software people are actually paid to test looks like
// this, and almost no modern practice site offers it any more.
package legacy

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/render"
)

const prefix = "/legacy"

type page struct {
	meta  challenge.Challenge
	mount func(chi.Router, *render.Renderer)
}

func pages() []page {
	return []page{
		nativeDialogs(),
		windows(),
		visibility(),
		ajaxSearch(),
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

// Pages serves the zone index and mounts each challenge under its own path.
func Pages(renderer *render.Renderer) http.Handler {
	r := chi.NewRouter()
	zone, _ := challenge.LookupZone(challenge.ZoneLegacy)

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

// simplePage is the shape of a Zone 2 challenge: one template, no server state,
// with everything interesting happening in jQuery on the client.
func simplePage(meta challenge.Challenge) page {
	return page{
		meta: meta,
		mount: func(r chi.Router, renderer *render.Renderer) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "legacy/"+meta.ID, render.View{
					Title:     meta.Title,
					Challenge: &meta,
				})
			})
		},
	}
}
