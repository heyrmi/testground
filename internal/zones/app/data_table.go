package app

import (
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/control"
	"github.com/heyrmi/testground/internal/fake"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

const tableRows = 120

func dataTable() challenge.Challenge {
	return challenge.Challenge{
		ID:       "data-table",
		Title:    "A table that sorts on the server",
		URL:      "/app/data-table",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "I. Tables, Lists, Data",
		Summary: "A hundred and twenty rows behind server-side sorting, filtering and offset " +
			"pagination, with a select-all checkbox, editable cells, and switches that force " +
			"the loading, empty and error states on demand.",
		WhyHard: "Sorting is a round trip, so a click on a header leaves the old rows on screen " +
			"until the response lands -- assert immediately and you are asserting on the " +
			"previous order. The select-all box has three states rather than two: with some " +
			"rows chosen it is neither checked nor unchecked but indeterminate, which is a " +
			"property rather than an attribute and is invisible to most assertions. An " +
			"edited cell commits on blur, so a value read while the input still has focus " +
			"has not been saved yet. And the empty, loading and error states all look like " +
			"'the rows are not there yet' to a wait that only knows how to look for rows.",
		Hint: "Wait for the table to say it has settled rather than for rows to exist -- it " +
			"publishes the sort it is currently showing, which is the thing that actually " +
			"changed. Read the select-all box's indeterminate property rather than looking " +
			"for an attribute. Commit an edit by moving focus before reading the cell. And " +
			"treat empty and error as outcomes to assert on, not as a slow success: the " +
			"page distinguishes them and a timeout does not.",
		Tags:     []string{"tables", "sorting", "pagination", "select-all", "inline-edit"},
		Concepts: []string{"server-side sort round trips", "indeterminate is a third state", "commit on blur", "empty and error are not slow success"},
		Selectors: []challenge.Selector{
			{TestID: "table", Role: "table", Note: "The table itself"},
			{TestID: "sort-name", Role: "columnheader", Note: "Sorts by name; each header is sort-<column>"},
			{TestID: "sort-amount", Role: "columnheader", Note: "Sorts by amount"},
			{TestID: "current-sort", Note: "The sort the rows on screen were fetched with"},
			{TestID: "filter", Role: "searchbox", Note: "Filters by name on the server"},
			{TestID: "row", Role: "row", Note: "One per row on this page; narrow by data-id"},
			{TestID: "select-all", Role: "checkbox", Note: "Neither checked nor unchecked when only some rows are chosen"},
			{TestID: "select-row", Role: "checkbox", Note: "Inside a row"},
			{TestID: "selected-count", Note: "How many rows are chosen across every page"},
			{TestID: "cell-note", Note: "Click to edit; commits on blur"},
			{TestID: "page-next", Role: "button", Note: "Next page"},
			{TestID: "page-prev", Role: "button", Note: "Previous page"},
			{TestID: "page-label", Note: "Which page of how many"},
			{TestID: "total-rows", Note: "How many rows match the current filter"},
			{TestID: "table-loading", Transient: true, Note: "While a fetch is in flight"},
			{TestID: "table-empty", Transient: true, Note: "When the filter matches nothing"},
			{TestID: "table-error", Transient: true, Note: "When the server refused"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/app/table/rows", Note: "sort, dir, q, page, size and state"},
		},
		Controls: []challenge.Control{
			{Name: "state", Kind: "query", Default: "", Note: "Force error, or slow, on the rows endpoint."},
			{Name: "size", Kind: "query", Default: "10", Note: "Rows per page, clamped to 1-100."},
			{
				Name:    "flake",
				Kind:    "control-plane",
				Default: "0",
				Note: "POST /api/control/flake {\"challenge\":\"data-table\"} reverses that share of " +
					"responses while still reporting the sort that was asked for, so refreshing " +
					"one query answers with two different orders.",
			},
		},
		Stability: challenge.Stable,
	}
}

type tableRow struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Status string `json:"status"`
	Amount string `json:"amount"`
	Note   string `json:"note"`
}

type tableResponse struct {
	Rows  []tableRow `json:"rows"`
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Pages int        `json:"pages"`
	Sort  string     `json:"sort"`
	Dir   string     `json:"dir"`
}

func handleTableRows(w http.ResponseWriter, r *http.Request) {
	sess := session.MustFromContext(r.Context())
	query := r.URL.Query()

	switch query.Get("state") {
	case "error":
		httpx.Fail(w, http.StatusInternalServerError, "the table service is having a moment")
		return
	case "slow":
		if err := sleep(r.Context(), 1500*time.Millisecond); err != nil {
			return
		}
	}

	rows := allTableRows(sess)

	if needle := strings.ToLower(strings.TrimSpace(query.Get("q"))); needle != "" {
		filtered := rows[:0:0]
		for _, row := range rows {
			if strings.Contains(strings.ToLower(row.Name), needle) {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}

	column, dir := query.Get("sort"), query.Get("dir")
	if dir != "desc" {
		dir = "asc"
	}
	if column != "" {
		sortRows(rows, column, dir)
	}

	// A flake rule reorders the whole result set while the response still
	// reports the sort it was asked for, so the same query answers twice with
	// two different orders. Reversing before paging means the rows on a page
	// change rather than merely swapping places, which is what makes a test
	// that asserts on the first row fail on some refreshes and not others.
	if control.Flaked(w, r, "data-table") {
		slices.Reverse(rows)
	}

	size := httpx.QueryInt(r, "size", 10, 1, 100)
	pages := (len(rows) + size - 1) / size
	page := httpx.QueryInt(r, "page", 1, 1, max(pages, 1))

	start := min((page-1)*size, len(rows))
	httpx.JSON(w, http.StatusOK, tableResponse{
		Rows:  rows[start:min(start+size, len(rows))],
		Total: len(rows),
		Page:  page,
		Pages: pages,
		Sort:  column,
		Dir:   dir,
	})
}

// Sorting happens here rather than in the browser, which is what makes a
// header click a round trip and the rows on screen stale until it lands.
func sortRows(rows []tableRow, column, dir string) {
	less := func(i, j int) bool { return rows[i].ID < rows[j].ID }
	switch column {
	case "name":
		less = func(i, j int) bool { return rows[i].Name < rows[j].Name }
	case "status":
		less = func(i, j int) bool { return rows[i].Status < rows[j].Status }
	case "amount":
		less = func(i, j int) bool { return amountOf(rows[i]) < amountOf(rows[j]) }
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if dir == "desc" {
			return less(j, i)
		}
		return less(i, j)
	})
}

// amountOf compares the money column numerically. Sorting it as text would put
// 9.99 above 1000.00, which is a real bug worth not shipping by accident.
func amountOf(row tableRow) float64 {
	var whole, frac float64
	var seenDot bool
	scale := 0.1

	for _, c := range row.Amount {
		switch {
		case c == '.':
			seenDot = true
		case c >= '0' && c <= '9':
			if seenDot {
				frac += float64(c-'0') * scale
				scale /= 10
			} else {
				whole = whole*10 + float64(c-'0')
			}
		}
	}
	return whole + frac
}

func allTableRows(sess *session.Session) []tableRow {
	stream := sess.RNG.Stream("data-table")
	rows := make([]tableRow, tableRows)

	for i := range rows {
		person := fake.NewPerson(stream, i)
		rows[i] = tableRow{
			ID:     i + 1,
			Name:   person.Name,
			Email:  person.Email,
			Status: person.Status,
			Amount: person.Amount,
			Note:   "",
		}
	}
	return rows
}
