package legacy

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/fake"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/render"
	"github.com/heyrmi/testground/internal/session"
)

const (
	searchCorpusSize = 400
	searchPageSize   = 25
)

type searchResult struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

type searchResponse struct {
	Query   string         `json:"query"`
	Matches []searchResult `json:"matches"`
	// Shown is how many rows came back; Total is how many matched. They differ
	// once a query matches more than the page size, and a test that reads the
	// wrong one cannot tell two different searches apart.
	Shown int `json:"shown"`
	Total int `json:"total"`
}

func ajaxSearch() page {
	meta := challenge.Challenge{
		ID:       "ajax-search",
		Title:    "Debounced search that replaces its own results",
		URL:      "/legacy/ajax-search",
		Zone:     challenge.ZoneLegacy,
		Tier:     challenge.T2,
		Category: "C. Waits & Timing",
		Summary: "A search box that waits 300 ms after you stop typing before asking the " +
			"server, shows a spinner while it waits, and then replaces the entire results " +
			"list with new markup. A counter records how many requests were actually sent.",
		WhyHard: "Typing and asserting immediately measures a prefix of what you typed, or " +
			"the spinner, or nothing at all -- the request for the full term has not been " +
			"sent yet. The results list is replaced wholesale rather than updated, so a " +
			"row located before the swap is detached afterwards even though a row that " +
			"looks exactly like it is on screen. And because responses can arrive out of " +
			"order, the list you are looking at is not guaranteed to belong to the last " +
			"thing you typed.",
		Hint: "Wait for the state you want rather than for a duration: the spinner leaving " +
			"and the result count settling both say more than 300 ms of sleeping does. " +
			"Re-locate rows after each search instead of holding them across it. The " +
			"request counter is there so you can prove the debounce did its job -- typing " +
			"eight characters should not produce eight requests.",
		Tags:     []string{"ajax", "debounce", "jquery", "detachment"},
		Concepts: []string{"debounced input", "waiting for a state not a duration", "wholesale DOM replacement", "request counting"},
		Selectors: []challenge.Selector{
			{TestID: "search-input", Role: "searchbox", Note: "Fires a request 300 ms after typing stops"},
			{TestID: "search-spinner", Transient: true, Note: "Present only while a request is in flight"},
			{TestID: "search-results", Note: "Replaced wholesale on every response"},
			{TestID: "search-row", Transient: true, Note: "One per match; detached by the next search"},
			{TestID: "search-count", Note: "How many records matched in total, which can exceed the rows shown"},
			{TestID: "request-count", Note: "How many requests were actually sent; proves the debounce works"},
			{TestID: "search-shown", Note: "How many rows are on screen, capped at 25"},
			{TestID: "search-empty", Note: "Shown when nothing matches, and before the first search"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/legacy/search?q=", Note: "Case-insensitive name match over 400 seeded records"},
		},
		Stability: challenge.Stable,
	}

	return page{
		meta: meta,
		mount: func(r chi.Router, renderer *render.Renderer) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "legacy/ajax-search", render.View{Title: meta.Title, Challenge: &meta})
			})
		},
	}
}

// API serves the JSON this zone's challenges call, mounted at /api/legacy.
func API() http.Handler {
	r := chi.NewRouter()
	r.Get("/search", handleSearch)
	return r
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	sess := session.MustFromContext(r.Context())
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))

	stream := sess.RNG.Stream("legacy-search")
	matches := make([]searchResult, 0, searchPageSize)
	total := 0

	for i := range searchCorpusSize {
		person := fake.NewPerson(stream, i)
		if query != "" && !strings.Contains(strings.ToLower(person.Name), query) {
			continue
		}
		total++
		if len(matches) < searchPageSize {
			matches = append(matches, searchResult{
				Name:   person.Name,
				Email:  person.Email,
				Status: person.Status,
			})
		}
	}

	httpx.JSON(w, http.StatusOK, searchResponse{
		Query:   query,
		Matches: matches,
		Shown:   len(matches),
		Total:   total,
	})
}
