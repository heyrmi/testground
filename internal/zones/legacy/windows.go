package legacy

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/render"
)

type popupView struct {
	Kind string
}

func windows() page {
	meta := challenge.Challenge{
		ID:       "windows",
		Title:    "New tabs, popups and one that writes back",
		URL:      "/legacy/windows",
		Zone:     challenge.ZoneLegacy,
		Tier:     challenge.T2,
		Category: "H. Windows, Tabs, Navigation",
		Summary: "A target=_blank link, a window.open popup with dimensions, a popup that " +
			"closes itself after two seconds, and one that writes a value back into this " +
			"page before it goes.",
		WhyHard: "The new tab does not become the one your test is driving. Every locator " +
			"keeps pointing at the page it started on, so an assertion about the popup's " +
			"content fails against the opener while the popup sits there working " +
			"perfectly. Getting the handle means listening for the window before the click " +
			"that creates it, because after the click the event has already gone. The " +
			"self-closing one races you: reach for it too late and it is gone, and the " +
			"error you get is about a closed target rather than about timing. And the one " +
			"that writes back changes this page from a context your locators are not " +
			"looking at.",
		Hint: "Wait for the new page before the click that opens it, not after, and switch " +
			"to it deliberately. For the self-closing popup, capture what you need " +
			"immediately or assert on what it left behind in the opener instead -- the " +
			"opener outlives it, which makes it the more reliable target. Remember to " +
			"switch back; a test that finishes pointing at a closed window fails on its " +
			"next step for reasons that have nothing to do with the next step.",
		Tags:     []string{"windows", "tabs", "popups", "window-open", "opener"},
		Concepts: []string{"a new tab is a new context", "listen before the click", "windows that close themselves", "writing back to the opener"},
		Selectors: []challenge.Selector{
			{TestID: "blank-link", Role: "link", Note: "target=_blank; opens a tab this page keeps no handle on"},
			{TestID: "open-popup", Role: "button", Note: "window.open with explicit dimensions"},
			{TestID: "open-closing", Role: "button", Note: "Opens a popup that closes itself after two seconds"},
			{TestID: "open-writer", Role: "button", Note: "Opens a popup that writes into this page and then closes"},
			{TestID: "from-popup", Note: "In this page; what the popup wrote back"},
			{TestID: "popup-target", Transient: true, Note: "Inside the popup document, not this one"},
		},
		Stability: challenge.Stable,
	}

	return page{
		meta: meta,
		mount: func(r chi.Router, renderer *render.Renderer) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "legacy/windows", render.View{Title: meta.Title, Challenge: &meta})
			})

			r.Get("/popup", func(w http.ResponseWriter, req *http.Request) {
				kind := req.URL.Query().Get("kind")
				if kind == "" {
					kind = "tab"
				}
				renderer.Page(w, req, "legacy/windows-popup", render.View{
					Title: "Popup",
					Data:  popupView{Kind: kind},
				})
			})
		},
	}
}
