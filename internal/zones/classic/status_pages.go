package classic

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/render"
)

type statusView struct {
	Code       int
	Reason     string
	RetryAfter string
}

// served lists the statuses this challenge answers with, in display order.
var served = []statusView{
	{Code: 400, Reason: "The request itself was malformed."},
	{Code: 401, Reason: "Authentication is required and was not supplied."},
	{Code: 403, Reason: "Authenticated, and still not allowed."},
	{Code: 404, Reason: "Nothing is served here."},
	{Code: 418, Reason: "A teapot. Real, and in the registry."},
	{Code: 429, Reason: "Too many requests.", RetryAfter: "5"},
	{Code: 500, Reason: "The server broke while handling this."},
	{Code: 502, Reason: "An upstream answered with nonsense."},
	{Code: 503, Reason: "Temporarily unavailable.", RetryAfter: "10"},
	{Code: 504, Reason: "An upstream never answered at all."},
}

func statusPages() page {
	meta := challenge.Challenge{
		ID:       "status-pages",
		Title:    "Error pages that are still pages",
		URL:      "/classic/status-pages",
		Zone:     challenge.ZoneClassic,
		Tier:     challenge.T2,
		Category: "H. Windows, Tabs, Navigation",
		Summary: "Ten status codes, each answered with a complete, styled HTML page. Two of " +
			"them carry a Retry-After header.",
		WhyHard: "A rendered error page is the failure most suites miss. The navigation " +
			"succeeded, the document parsed, the heading is right there, and every " +
			"assertion about content passes -- while the response was a 500. Any check " +
			"that only looks at the DOM cannot tell a working page from a broken one that " +
			"was styled carefully. Retry-After is the other half: a 429 or 503 is telling " +
			"you how long to wait, and code that retries on its own schedule is ignoring " +
			"the only reliable answer it was given.",
		Hint: "Assert on the status code, not only on what rendered. Most frameworks return " +
			"the response object from a navigation, and reading its status is one line. " +
			"When you see a 429 or a 503, read Retry-After and honour it rather than " +
			"inventing a backoff.",
		Tags:     []string{"navigation", "status-codes", "errors", "retry-after"},
		Concepts: []string{"a rendered error page still failed", "asserting on the response", "Retry-After", "content assertions are not health checks"},
		Selectors: []challenge.Selector{
			{TestID: "status-link-404", Role: "link", Family: "status-link-<n>", Note: "Goes to the 404 page; there is one link per status the page offers"},
			{TestID: "status-link-500", Role: "link", Note: "Goes to the 500 page"},
			{TestID: "status-link-503", Role: "link", Note: "Goes to the 503 page, which carries Retry-After"},
			{TestID: "status-code", Transient: true, Note: "On each error page: the code it was served with"},
			{TestID: "status-reason", Transient: true, Note: "On each error page: what that code means here"},
			{TestID: "status-retry-after", Transient: true, Note: "Present only where the response carried the header"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/classic/status-pages/{code}", Note: "Answers that status with a full HTML page"},
		},
		Stability: challenge.Stable,
	}

	return page{
		meta: meta,
		mount: func(r chi.Router, renderer *render.Renderer) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "classic/status-pages", render.View{
					Title:     meta.Title,
					Challenge: &meta,
					Data:      served,
				})
			})

			r.Get("/{code}", func(w http.ResponseWriter, req *http.Request) {
				code, err := strconv.Atoi(chi.URLParam(req, "code"))
				if err != nil {
					httpx.Fail(w, http.StatusNotFound, "not a status this page serves")
					return
				}
				for _, status := range served {
					if status.Code != code {
						continue
					}
					if status.RetryAfter != "" {
						w.Header().Set("Retry-After", status.RetryAfter)
					}
					renderer.PageStatus(w, req, code, "classic/status-page", render.View{
						Title: strconv.Itoa(code),
						Data:  status,
					})
					return
				}
				httpx.Fail(w, http.StatusNotFound, "not a status this page serves")
			})
		},
	}
}
