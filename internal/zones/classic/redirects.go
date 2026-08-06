package classic

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/render"
)

// chainHops is how many redirects sit between the link and the landing page.
const chainHops = 3

type landedView struct {
	Via  string
	Hops int
	// Method is what actually arrived. It is the only visible difference
	// between the redirect codes that preserve it and the ones that do not.
	Method string
}

func redirects() page {
	meta := challenge.Challenge{
		ID:       "redirects",
		Title:    "Redirect chains, status codes and meta refresh",
		URL:      "/classic/redirects",
		Zone:     challenge.ZoneClassic,
		Tier:     challenge.T2,
		Category: "H. Windows, Tabs, Navigation",
		Summary: "A three-hop redirect chain, one link per redirect status from 301 to 308, " +
			"and a meta-refresh page that is not an HTTP redirect at all.",
		WhyHard: "The browser follows all of this silently, so the URL a test asked for and " +
			"the page it is looking at are different things, and an assertion on the URL " +
			"passes or fails depending on which one it meant. The status codes are not " +
			"interchangeable: 307 and 308 keep the method and body, while 301, 302 and 303 " +
			"are allowed to turn a POST into a GET, which quietly changes what a form " +
			"submission does. Meta refresh is different again -- there is no Location " +
			"header and no redirect, just a document that replaces itself a second after " +
			"it has finished loading, so a navigation wait can resolve on the page you were " +
			"leaving.",
		Hint: "Assert on the final URL rather than the one you requested, and read the " +
			"response chain if your framework exposes it -- that is where the hop count " +
			"lives. For meta refresh, waiting for navigation is not enough on its own " +
			"because the first document really did load; wait for something only the " +
			"destination has.",
		Tags:     []string{"navigation", "redirects", "status-codes", "meta-refresh"},
		Concepts: []string{"redirect chains", "method preservation across 307 and 308", "meta refresh is not a redirect", "asserting on the final URL"},
		Selectors: []challenge.Selector{
			{TestID: "chain-link", Role: "link", Note: "Starts a three-hop chain"},
			{TestID: "meta-link", Role: "link", Note: "Goes to a page that replaces itself after one second"},
			{TestID: "code-301", Role: "link", Note: "Permanent redirect"},
			{TestID: "code-302", Role: "link", Note: "Found; may turn a POST into a GET"},
			{TestID: "code-303", Role: "link", Note: "See Other; always becomes a GET"},
			{TestID: "code-307", Role: "link", Note: "Temporary; keeps the method and body"},
			{TestID: "code-308", Role: "link", Note: "Permanent; keeps the method and body"},
			{TestID: "landed", Transient: true, Note: "Only on the destination page"},
			{TestID: "landed-via", Transient: true, Note: "Says which route arrived here"},
			{TestID: "landed-hops", Transient: true, Note: "How many redirects were followed"},
			{TestID: "landed-method", Transient: true, Note: "The method that arrived; 307 and 308 keep it, the others may not"},
			{TestID: "meta-waiting", Transient: true, Note: "The page that is about to replace itself"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/classic/redirects/chain/1", Note: "First hop; each answers 302 to the next"},
			{Method: http.MethodGet, Path: "/classic/redirects/code/{code}", Note: "Answers that status with a Location header"},
			{Method: "ANY", Path: "/classic/redirects/landed", Note: "The destination; reports the method it received"},
		},
		Stability: challenge.Stable,
	}

	return page{
		meta: meta,
		mount: func(r chi.Router, renderer *render.Renderer) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "classic/redirects", render.View{Title: meta.Title, Challenge: &meta})
			})

			r.Get("/chain/{hop}", func(w http.ResponseWriter, req *http.Request) {
				hop, err := strconv.Atoi(chi.URLParam(req, "hop"))
				if err != nil || hop < 1 {
					httpx.Fail(w, http.StatusNotFound, "no such hop")
					return
				}
				if hop >= chainHops {
					http.Redirect(w, req, "/classic/redirects/landed?via=chain", http.StatusFound)
					return
				}
				http.Redirect(w, req, "/classic/redirects/chain/"+strconv.Itoa(hop+1), http.StatusFound)
			})

			// Any method, so a POST can be followed through each code and the
			// difference between preserving it and rewriting it is observable
			// rather than a 405.
			r.HandleFunc("/code/{code}", func(w http.ResponseWriter, req *http.Request) {
				code, err := strconv.Atoi(chi.URLParam(req, "code"))
				if err != nil || !isRedirect(code) {
					httpx.Fail(w, http.StatusNotFound, "not a redirect status")
					return
				}
				http.Redirect(w, req, "/classic/redirects/landed?via="+strconv.Itoa(code), code)
			})

			r.Get("/meta", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "classic/redirects-meta", render.View{Title: "About to move"})
			})

			r.HandleFunc("/landed", func(w http.ResponseWriter, req *http.Request) {
				via := req.URL.Query().Get("via")
				hops := 0
				if via == "chain" {
					hops = chainHops
				} else if via != "" {
					hops = 1
				}
				renderer.Page(w, req, "classic/redirects-landed", render.View{
					Title: "Arrived",
					Data:  landedView{Via: via, Hops: hops, Method: req.Method},
				})
			})
		},
	}
}

// isRedirect keeps the code route from being a generic open redirect.
func isRedirect(code int) bool {
	switch code {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	}
	return false
}
