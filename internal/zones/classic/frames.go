package classic

import (
	"net"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/render"
)

// nestingDepth is how many frames sit between the page and the deepest target.
const nestingDepth = 3

type framesView struct {
	// CrossOriginBase is built from the host the browser actually used, so the
	// frame resolves whether the playground was reached on a laptop, on a LAN
	// address, or from inside a container.
	CrossOriginBase string
}

type nestedView struct {
	Level int
	Next  int
	Last  bool
}

func crossOriginFrame(opts Options) page {
	meta := challenge.Challenge{
		ID:       "frames",
		Title:    "Same-origin, cross-origin and nested frames",
		URL:      "/classic/frames",
		Zone:     challenge.ZoneClassic,
		Tier:     challenge.T3,
		Category: "E. Frames & Contexts",
		Summary: "Four frames on one page: a same-origin frame, a genuinely cross-origin " +
			"one served from a second port bound by this same binary, a chain nested " +
			"three deep, and a srcdoc frame whose content is an attribute rather than a " +
			"URL. Nothing on this page runs a script.",
		WhyHard: "A locator that searches the page does not search inside a frame, so every " +
			"target here is invisible until the search is pointed at the right context -- " +
			"and for the nested chain that means descending three of them in order. The " +
			"cross-origin frame draws a line the other two do not: script running in the " +
			"parent cannot read its document at all, and no header or wait changes that, " +
			"because the browser decides same-origin from scheme, host and port alone. " +
			"Your framework can still enter it, because it drives the browser rather than " +
			"running inside the page, and knowing which of those two positions your code " +
			"is in is the whole lesson.",
		Hint: "Enter each frame explicitly instead of searching the page, and descend the " +
			"nested chain a level at a time. For the cross-origin frame, notice what does " +
			"and does not work: reading contentDocument from page script gives you " +
			"nothing, while your framework switches into it without complaint. Read the " +
			"session id shown inside it too -- it matches the parent's, because cookies " +
			"are scoped to a host and ignore the port, so these two origins share state " +
			"while sharing no DOM at all.",
		Tags:     []string{"frames", "iframe", "cross-origin", "nesting", "srcdoc"},
		Concepts: []string{"frame contexts", "same-origin policy", "nested browsing contexts", "srcdoc", "in-page script versus the driver"},
		Selectors: []challenge.Selector{
			{TestID: "frame-same-origin", Note: "Same-origin iframe; page script can read straight into it"},
			{TestID: "frame-cross-origin", Note: "Served from the second port, so page script cannot read it at all"},
			{TestID: "frame-nested", Note: "Outermost of the nested chain"},
			{TestID: "frame-srcdoc", Note: "Content comes from the srcdoc attribute, not from a URL"},
			{TestID: "embedded-target", Frame: []string{"frame-same-origin"}, Note: "One frame down, same origin"},
			{TestID: "cross-origin-target", Frame: []string{"frame-cross-origin"}, Note: "One frame down, different origin"},
			{TestID: "cross-origin-session", Frame: []string{"frame-cross-origin"}, Note: "The session the second origin resolved to; it matches the parent's"},
			{TestID: "srcdoc-target", Frame: []string{"frame-srcdoc"}, Note: "Inside a frame that has no URL"},
			{TestID: "deepest-target", Frame: []string{"frame-nested", "frame-level-2", "frame-level-3"}, Note: "Three frames down"},
			{TestID: "parent-session", Note: "The parent page's session, for comparing with the frame's"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/frame", Note: "On the second origin: the embedded document"},
			{Method: http.MethodGet, Path: "/whoami", Note: "On the second origin: the session it resolved to"},
		},
		Controls: []challenge.Control{
			{
				Name:    "--cross-origin-addr",
				Kind:    "flag",
				Default: "127.0.0.1:7374",
				Note:    "The second address to bind. Setting it empty unregisters this challenge rather than serving it broken.",
			},
		},
		Stability: challenge.Stable,
	}

	return page{
		meta: meta,
		mount: func(r chi.Router, renderer *render.Renderer) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "classic/frames", render.View{
					Title:     meta.Title,
					Challenge: &meta,
					Data:      framesView{CrossOriginBase: crossOriginBase(req, opts.CrossOriginPort)},
				})
			})

			r.Get("/embedded", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "classic/frames-embedded", render.View{Title: "Embedded"})
			})

			// One route serves the whole nested chain, so the depth is a
			// number in one place rather than a template per level.
			r.Get("/nested/{level}", func(w http.ResponseWriter, req *http.Request) {
				level, err := strconv.Atoi(chi.URLParam(req, "level"))
				if err != nil || level < 1 || level > nestingDepth {
					httpx.Fail(w, http.StatusNotFound, "no such nesting level")
					return
				}
				renderer.Page(w, req, "classic/frames-nested", render.View{
					Title: "Nested frame",
					Data:  nestedView{Level: level, Next: level + 1, Last: level == nestingDepth},
				})
			})
		},
	}
}

// crossOriginBase rebuilds the second origin's base URL from the host the
// browser used, so the frame resolves without the operator configuring
// anything.
func crossOriginBase(r *http.Request, port string) string {
	host := r.Host
	if split, _, err := net.SplitHostPort(host); err == nil {
		host = split
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + net.JoinHostPort(host, port)
}
