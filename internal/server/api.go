package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/build"
	"github.com/heyrmi/testground/internal/render"
	"github.com/heyrmi/testground/internal/session"
)

// apiError is the one error shape every JSON route returns.
type apiError struct {
	Status int    `json:"status"`
	Error  string `json:"error"`
}

// writeJSON emits indented JSON. The manifest is meant to be read by people as
// well as machines, and stable formatting keeps diffs of it meaningful.
func writeJSON(w http.ResponseWriter, status int, body any) {
	encoded, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		http.Error(w, `{"status":500,"error":"could not encode response"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	w.Write(append(encoded, '\n'))
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, apiError{Status: status, Error: message})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"challenges": s.opts.Registry.Len(),
		"sessions":   s.opts.Sessions.Len(),
	})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, build.Current())
}

// handleManifest serves the self-describing catalogue. It is session-scoped:
// the seed it reports is the seed the caller's own pages are generated from.
func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	sess := session.MustFromContext(r.Context())
	writeJSON(w, http.StatusOK, s.opts.Registry.Manifest(s.opts.Version, string(sess.ID), sess.RNG.Seed()))
}

func (s *Server) handleChallenge(w http.ResponseWriter, r *http.Request) {
	found, ok := s.opts.Registry.Lookup(chi.URLParam(r, "id"))
	if !ok {
		writeError(w, http.StatusNotFound, "no such challenge")
		return
	}
	writeJSON(w, http.StatusOK, found)
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	if wantsJSON(r) {
		writeError(w, http.StatusNotFound, "no such route")
		return
	}
	s.opts.Renderer.PageStatus(w, r, http.StatusNotFound, "not-found", render.View{
		Title: "Not found",
		Data:  notFoundView{Path: r.URL.Path},
	})
}

func (s *Server) handleMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed on this route")
}

func wantsJSON(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/api/") ||
		strings.Contains(r.Header.Get("Accept"), "application/json")
}
