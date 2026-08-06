package server

import (
	"net/http"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/render"
)

type indexView struct {
	Groups []challenge.ZoneGroup
}

type notFoundView struct {
	Path string
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.opts.Renderer.Page(w, r, "index", render.View{
		Title: "Challenges",
		Data:  indexView{Groups: s.opts.Registry.ByZone()},
	})
}
