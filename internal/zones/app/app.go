// Package app serves Zone 4, the modern SPA.
//
// The zone owns both its pages and the JSON its pages call, so everything a
// challenge needs lives in one package.
package app

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
)

// Challenges declares every challenge this zone serves.
func Challenges() []challenge.Challenge {
	return []challenge.Challenge{
		delayedElement(),
		toast(),
		virtualList(),
		optimisticRevert(),
	}
}

// Pages serves the SPA shell for every path in the zone. Client routing owns
// everything below the prefix, so an unmatched path returns the same document
// and lets the router render its own not-found page.
func Pages(dist fs.FS) http.Handler {
	shell, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return missingBundle()
	}

	r := chi.NewRouter()
	serve := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(shell)
	}
	r.Get("/", serve)
	r.Get("/*", serve)
	return r
}

// API serves the JSON this zone's challenges call, mounted at /api/app.
func API() http.Handler {
	r := chi.NewRouter()
	r.Get("/virtual-list/rows", handleVirtualListRows)
	r.Get("/optimistic-revert/tasks", handleOptimisticTasks)
	r.Post("/optimistic-revert/tasks/{id}/toggle", handleOptimisticToggle)
	return r
}

// missingBundle keeps the rest of the playground usable when someone builds
// the binary without building the frontend, rather than failing at startup.
func missingBundle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`<!doctype html><meta charset="utf-8">
<title>Frontend bundle missing</title>
<link rel="stylesheet" href="/static/shell.css">
<main class="shell"><section class="intro">
<h1>This zone was built without its frontend bundle.</h1>
<p>Run <code>make web</code> (or <code>npm ci &amp;&amp; npm run build</code> in
<code>web/app</code>) and rebuild the binary. Every other zone still works.</p>
<p><a href="/">Back to the challenge index</a></p>
</section></main>`))
	})
}
