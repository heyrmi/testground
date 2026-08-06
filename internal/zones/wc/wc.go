// Package wc serves Zone 5, the web component zone.
package wc

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/render"
)

// Challenges declares every challenge this zone serves.
func Challenges() []challenge.Challenge {
	return []challenge.Challenge{
		nestedShadow(),
	}
}

// Pages serves the zone index and one page per challenge.
func Pages(renderer *render.Renderer) http.Handler {
	r := chi.NewRouter()
	zone, _ := challenge.LookupZone(challenge.ZoneComponents)

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		renderer.Page(w, req, "zone", render.View{
			Title: zone.Title,
			Data:  render.ZoneView{Zone: zone, Challenges: Challenges()},
		})
	})

	for _, c := range Challenges() {
		page := c
		r.Get(routeFor(page), func(w http.ResponseWriter, req *http.Request) {
			renderer.Page(w, req, "wc/"+page.ID, render.View{Title: page.Title, Challenge: &page})
		})
	}

	return r
}

// routeFor turns a challenge's absolute URL into the path this router matches,
// which is everything after the zone prefix.
func routeFor(c challenge.Challenge) string {
	return c.URL[len("/wc"):]
}
