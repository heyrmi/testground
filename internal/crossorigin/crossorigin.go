// Package crossorigin serves the second origin.
//
// A genuinely different origin cannot be faked from one listener: the browser
// decides what is same-origin from scheme, host and port, and no header
// changes its mind. So the binary binds a second port and serves a small set
// of documents there. Same binary, same session store, different origin.
//
// Note what this is and is not. A second port on the same host is
// cross-origin, so DOM access across the boundary throws and postMessage
// becomes the only channel -- which is the lesson. It is still the same
// *site*, so cookies continue to flow; third-party cookie blocking needs a
// different host and is not what this exercises.
package crossorigin

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/render"
	"github.com/heyrmi/testground/internal/session"
)

// Routes serves the documents that are embedded from the main origin.
func Routes(renderer *render.Renderer) http.Handler {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		renderer.Page(w, req, "crossorigin/index", render.View{Title: "Second origin"})
	})

	// The document embedded as a cross-origin iframe. It shows its own
	// session id so a test can prove both origins resolved to the same
	// session -- cookies are not port-scoped, which surprises people.
	r.Get("/frame", func(w http.ResponseWriter, req *http.Request) {
		renderer.Page(w, req, "crossorigin/frame", render.View{Title: "Cross-origin frame"})
	})

	// Proof that state is shared across the two origins even though the DOM
	// is not: this reports the session the request resolved to.
	r.Get("/whoami", func(w http.ResponseWriter, req *http.Request) {
		sess := session.MustFromContext(req.Context())
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(`{"session":"` + string(sess.ID) + `","origin":"cross"}` + "\n"))
	})

	return r
}
