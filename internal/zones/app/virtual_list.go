package app

import (
	"fmt"
	"net/http"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/fake"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

const (
	virtualListRows    = 10_000
	virtualListMaxRows = 100_000
)

func virtualList() challenge.Challenge {
	return challenge.Challenge{
		ID:       "virtual-list",
		Title:    "Ten thousand rows, twenty in the DOM",
		URL:      "/app/virtual-list",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "I. Tables, Lists, Data",
		Summary: "Ten thousand rows are windowed into a fixed-height scroll container. Only " +
			"the rows near the viewport exist as elements; the rest are a number in a " +
			"style attribute.",
		WhyHard: "A locator for row 9,999 finds nothing, because that row has never been " +
			"rendered. Counting nodes measures the window rather than the data. Scrolling " +
			"the window does nothing at all, because the scroll container is an inner " +
			"element, and a row scrolled out of view detaches while a test still holds it.",
		Hint: "Scroll the container, not the window. Rows are a fixed height and positioned " +
			"by index, so row N sits at N times the row height and can be reached in one " +
			"jump instead of by repeated scrolling. Assert against the declared total " +
			"rather than the node count, and read the rows endpoint directly when the " +
			"whole data set is what you actually need.",
		Tags:     []string{"virtualisation", "lists", "scrolling", "large-data"},
		Concepts: []string{"windowed rendering", "inner scroll containers", "element detachment", "data through the API"},
		Selectors: []challenge.Selector{
			{TestID: "viewport", Note: "The scroll container; scrolling the window has no effect"},
			{TestID: "row", Role: "row", Note: "One rendered row; narrow by its data-index attribute"},
			{TestID: "row-index", Note: "Index cell, inside a row"},
			{TestID: "row-name", Note: "Name cell, inside a row"},
			{TestID: "row-status", Note: "Status cell, inside a row"},
			{TestID: "row-total", Note: "How many rows the data set holds"},
			{TestID: "row-rendered", Note: "How many rows are elements right now"},
			{TestID: "row-height", Note: "Row height in pixels, fixed so offsets are computable"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/app/virtual-list/rows", Note: "The whole data set, generated from the session seed"},
		},
		Controls: []challenge.Control{
			{
				Name:    "count",
				Kind:    "query",
				Default: fmt.Sprint(virtualListRows),
				Note:    "Rows to generate, clamped to 0-100000. Applies to the page and the endpoint.",
			},
		},
		Stability: challenge.Stable,
	}
}

type listRow struct {
	Index  int    `json:"index"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Status string `json:"status"`
	Amount string `json:"amount"`
}

type rowsResponse struct {
	Count int       `json:"count"`
	Seed  uint64    `json:"seed"`
	Rows  []listRow `json:"rows"`
}

func handleVirtualListRows(w http.ResponseWriter, r *http.Request) {
	sess := session.MustFromContext(r.Context())
	count := httpx.QueryInt(r, "count", virtualListRows, 0, virtualListMaxRows)

	httpx.JSON(w, http.StatusOK, rowsResponse{
		Count: count,
		Seed:  sess.RNG.Seed(),
		Rows:  generateRows(sess, count),
	})
}

// Rows are regenerated per request rather than cached. They are a pure
// function of the seed, so caching would only trade a millisecond of work for
// megabytes held per session.
func generateRows(sess *session.Session, count int) []listRow {
	stream := sess.RNG.Stream("virtual-list")
	rows := make([]listRow, count)

	for i := range rows {
		person := fake.NewPerson(stream, i)
		rows[i] = listRow{
			Index:  i,
			Name:   person.Name,
			Email:  person.Email,
			Status: person.Status,
			Amount: person.Amount,
		}
	}
	return rows
}
