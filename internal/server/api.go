package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/build"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/render"
	"github.com/heyrmi/testground/internal/session"
)

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httpx.JSONIndent(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"challenges": s.opts.Registry.Len(),
		"sessions":   s.opts.Sessions.Len(),
	})
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	httpx.JSONIndent(w, http.StatusOK, build.Current())
}

// handleManifest serves the self-describing catalogue. It is session-scoped:
// the seed it reports is the seed the caller's own pages are generated from.
func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	sess := session.MustFromContext(r.Context())
	httpx.JSONIndent(w, http.StatusOK,
		s.opts.Registry.Manifest(s.opts.Version, string(sess.ID), sess.RNG.Seed()))
}

func (s *Server) handleChallenge(w http.ResponseWriter, r *http.Request) {
	found, ok := s.opts.Registry.Lookup(chi.URLParam(r, "id"))
	if !ok {
		httpx.Fail(w, http.StatusNotFound, "no such challenge")
		return
	}
	httpx.JSONIndent(w, http.StatusOK, found)
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	if wantsJSON(r) {
		httpx.Fail(w, http.StatusNotFound, "no such route")
		return
	}
	s.opts.Renderer.PageStatus(w, r, http.StatusNotFound, "not-found", render.View{
		Title: "Not found",
		Data:  notFoundView{Path: r.URL.Path},
	})
}

func (s *Server) handleMethodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	httpx.Fail(w, http.StatusMethodNotAllowed, "method not allowed on this route")
}

func wantsJSON(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/api/") ||
		strings.Contains(r.Header.Get("Accept"), "application/json")
}
