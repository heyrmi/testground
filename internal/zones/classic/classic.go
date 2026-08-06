// Package classic serves Zone 1, the no-JavaScript zone.
//
// Nothing here runs a script. Every interaction is a form post answered with a
// redirect, which is the shape a great deal of real software still has and the
// shape most modern practice sites no longer offer at all.
package classic

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/render"
)

const prefix = "/classic"

// page couples a challenge's declaration with the routes that serve it, so
// everything about one challenge lives in one file.
type page struct {
	meta  challenge.Challenge
	mount func(chi.Router, *render.Renderer)
}

func pages() []page {
	return []page{
		textInputs(),
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
	zone, _ := challenge.LookupZone(challenge.ZoneClassic)

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
