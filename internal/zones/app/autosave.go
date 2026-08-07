package app

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

const (
	autosaveStateKey = "autosave"
	// The debounce is the window in which the field and the saved record
	// disagree. Eight hundred milliseconds is long enough to type through and
	// short enough that a person does not think the page is broken.
	autosaveDebounceMs = 800
	// The write itself is slow enough that "saving" is a state somebody can
	// observe. An indicator whose middle state never renders teaches nothing,
	// and the navigation guard has nothing to guard.
	autosaveLatencyMs = 400
	// Who last moved the record. These are values a test asserts on, so they
	// are constants rather than sentences assembled at the call site.
	autosaveWriterNobody = "nobody"
	autosaveWriterPage   = "this page"
	autosaveWriterOther  = "another writer"
)

func autosave() challenge.Challenge {
	return challenge.Challenge{
		ID:       "autosave",
		Title:    "Autosave, with someone else editing the same record",
		URL:      "/app/autosave",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "V. Composite Scenarios",
		Summary: "Three fields that save themselves eight hundred milliseconds after you stop " +
			"typing, under an indicator reading saved, saving or idle. The record carries " +
			"a version, and the server refuses a write that carries a stale one, so a " +
			"second writer -- there is a button that plays one -- turns the next autosave " +
			"into a conflict you resolve by keeping yours, taking theirs, or merging the " +
			"two field by field. While a write is in flight the page refuses to be " +
			"navigated away from.",
		WhyHard: "The indicator reports what the autosave loop is doing and never whether the " +
			"draft has reached the server, so it reads \"saved\" on load and goes on " +
			"reading it for the whole debounce window. A test that types and then waits " +
			"for \"saved\" is satisfied by the word that was already there before it " +
			"typed, and goes green having saved nothing. Widening the wait does not fix " +
			"it, because a real save ends on that same word: the assertion cannot tell " +
			"the two occasions apart. The conflict compounds it -- a refused write leaves " +
			"the field showing text the server has never accepted, which on screen is the " +
			"same picture as a successful one, and the word the indicator offers for it " +
			"is \"idle\".",
		Hint: "Wait for something only the server can move. The version is issued by the " +
			"server and appears on the page, and the count of accepted writes rises once " +
			"per acknowledgement, so either of them separates \"my edit landed\" from " +
			"\"the indicator says a word it has said since load\". Drive the second writer " +
			"from its endpoint when you want a conflict at a moment of your choosing " +
			"rather than a moment of the browser's. And when you resolve one, decide " +
			"first which change you are willing to lose: keeping yours discards theirs, " +
			"taking theirs discards yours, and only the merge is asked which field goes " +
			"which way.",
		Tags:     []string{"composite", "autosave", "debounce", "conflict", "optimistic-concurrency"},
		Concepts: []string{"debounced writes", "waiting on a server-issued fact", "default state that reads as success", "version conflicts", "three-way merge", "navigation guard"},
		Selectors: []challenge.Selector{
			{TestID: "field-title", Role: "textbox", Note: "The record's title; editing it starts the debounce"},
			{TestID: "field-owner", Role: "textbox", Note: "The field the simulated other writer changes"},
			{TestID: "field-notes", Role: "textbox", Note: "A longer field, so a merge has something to be about"},
			{TestID: "save-state", Role: "status", Note: "saved, saving or idle; it describes the autosave loop rather than the draft, and saved is where it starts"},
			{TestID: "record-version", Note: "The version the server has acknowledged to this page; only the server moves it"},
			{TestID: "save-count", Note: "How many writes this page has had accepted"},
			{TestID: "updated-by", Note: "Who last wrote the record this page is holding"},
			{TestID: "debounce-ms", Note: "How long typing must stop before a save is sent"},
			{TestID: "latency-ms", Note: "How long the server takes to answer a write"},
			{TestID: "simulate-other-writer", Role: "button", Note: "Moves the record from under this page, the way a second browser would"},
			{TestID: "other-writer-note", Note: "What the simulated writer last did, or that it has not run"},
			{TestID: "leave-link", Role: "link", Note: "Back to the zone index; refused while a write is in flight"},
			{TestID: "leave-blocked", Transient: true, Note: "Records that a navigation was refused because a write was still in flight"},
			{TestID: "conflict", Role: "alert", Transient: true, Note: "The whole resolution panel; absent until a write is refused"},
			{TestID: "conflict-versions", Transient: true, Note: "The version this page wrote against and the version the server holds"},
			{TestID: "keep-mine", Role: "button", Transient: true, Note: "Rewrites this page's draft at the server's current version, discarding theirs"},
			{TestID: "take-theirs", Role: "button", Transient: true, Note: "Adopts the server's record, discarding this page's edits"},
			{TestID: "show-merge", Role: "button", Transient: true, Note: "Opens the field-by-field comparison"},
			{TestID: "merge-view", Transient: true, Note: "The comparison; one row per field"},
			{TestID: "merge-row", Transient: true, Note: "One field of the comparison; narrow by data-field"},
			{TestID: "merge-mine", Transient: true, Note: "This page's value for that field"},
			{TestID: "merge-theirs", Transient: true, Note: "The server's value for that field"},
			{TestID: "merge-choice", Transient: true, Note: "Which side that field will take: mine or theirs"},
			{TestID: "merge-pick", Role: "button", Transient: true, Note: "Flips that field's side"},
			{TestID: "save-merge", Role: "button", Transient: true, Note: "Writes the assembled record at the server's current version"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/app/autosave/record", Note: "The record as the server holds it, with its version"},
			{Method: http.MethodPut, Path: "/api/app/autosave/record", Note: "Takes {version, fields}; 200 and a new version, or 409 and the record you were behind"},
			{Method: http.MethodPost, Path: "/api/app/autosave/other-writer", Note: "Plays a second editor: takes an optional {field, value}, always bumps the version"},
		},
		Controls: []challenge.Control{
			{
				Name:    "debounceMs",
				Kind:    "query",
				Default: fmt.Sprint(autosaveDebounceMs),
				Note:    "Milliseconds typing must stop for before the page sends a write, clamped to 0-10000.",
			},
			{
				Name:    "latencyMs",
				Kind:    "query",
				Default: fmt.Sprint(autosaveLatencyMs),
				Note: "Milliseconds the write endpoint waits before answering, clamped to 0-30000. " +
					"Widening it widens the window in which the navigation guard applies.",
			},
		},
		Stability: challenge.Stable,
	}
}

// autosaveFields is the record's shape, in render order. The set is fixed
// rather than open so a write carrying a field nobody declared is dropped
// instead of quietly growing the record a column the page never shows.
var autosaveFields = []string{"title", "owner", "notes"}

// autosaveSeed is the record every session starts from. It is fixed rather
// than drawn from the seeded stream because every assertion here is about a
// version number and a merge, and seed-dependent prose would make a test that
// names a field's value fail on a different --seed for no reason anyone could
// act on.
var autosaveSeed = map[string]string{
	"title": "Ferry timetable rewrite",
	"owner": "unassigned",
	"notes": "Two crossings a day in winter, four in summer.",
}

// autosaveOtherWriterField and Value are what the simulated second editor does
// when the caller does not say. It changes a different field from the one a
// test is most likely to be typing in, so a merge has a change on each side to
// keep and "keep mine" has something real to lose.
const (
	autosaveOtherWriterField = "owner"
	autosaveOtherWriterValue = "Priya Raman"
)

// record is one session's copy. The server owns the version: a client may
// state which version it believes it is editing, and never which version it is
// creating, because a client that could choose its own version could overwrite
// a change it never saw.
type record struct {
	mu        sync.Mutex
	fields    map[string]string
	version   int
	updatedBy string
	updatedAt time.Time
}

// recordView is the wire form. Fields is a map rather than three named members
// so the merge on the page can iterate it, and the server always sends every
// declared field so the page never has to reason about a missing one.
type recordView struct {
	Version   int               `json:"version"`
	Fields    map[string]string `json:"fields"`
	UpdatedBy string            `json:"updatedBy"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

func recordFor(sess *session.Session) *record {
	return session.Value(sess, autosaveStateKey, func() *record {
		fields := make(map[string]string, len(autosaveSeed))
		for name, value := range autosaveSeed {
			fields[name] = value
		}
		return &record{
			fields:    fields,
			version:   1,
			updatedBy: autosaveWriterNobody,
			updatedAt: sess.Clock.Now(),
		}
	})
}

func (r *record) view() recordView {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.viewLocked()
}

func (r *record) viewLocked() recordView {
	fields := make(map[string]string, len(r.fields))
	for name, value := range r.fields {
		fields[name] = value
	}
	return recordView{Version: r.version, Fields: fields, UpdatedBy: r.updatedBy, UpdatedAt: r.updatedAt}
}

// write applies the fields if version is the one the server currently holds.
// The comparison and the mutation happen under one lock: checking the version
// and then writing in two steps is the race this whole page is about, and
// reproducing it in the server would make the conflict arrive at random rather
// than exactly when a test asked for it.
func (r *record) write(version int, incoming map[string]string, by string, now time.Time) (recordView, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if version != r.version {
		return r.viewLocked(), false
	}

	for _, name := range autosaveFields {
		if value, ok := incoming[name]; ok {
			r.fields[name] = value
		}
	}
	r.version++
	r.updatedBy = by
	r.updatedAt = now
	return r.viewLocked(), true
}

// otherWriter moves the record without being asked to prove which version it
// was looking at. That is the whole point of it: it stands in for an editor
// this browser cannot see, so one test can create the conflict at a moment it
// chooses rather than by racing a second browser against itself.
func (r *record) otherWriter(field, value string, now time.Time) recordView {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.fields[field] = value
	r.version++
	r.updatedBy = autosaveWriterOther
	r.updatedAt = now
	return r.viewLocked()
}

func knownField(name string) bool {
	for _, declared := range autosaveFields {
		if declared == name {
			return true
		}
	}
	return false
}

// autosaveResponse is what every route here answers with, refusals included.
// Status and Error are the shape the rest of the zone uses for a refusal, so a
// caller that already knows how to read an error body does not need a second
// way for this route; whether a write landed is the HTTP status and not a
// field, because two places to look is one place to disagree.
type autosaveResponse struct {
	Record recordView `json:"record"`
	Status int        `json:"status,omitempty"`
	Error  string     `json:"error,omitempty"`
}

func handleAutosaveRecord(w http.ResponseWriter, r *http.Request) {
	sess := session.MustFromContext(r.Context())
	httpx.JSON(w, http.StatusOK, autosaveResponse{Record: recordFor(sess).view()})
}

func handleAutosaveWrite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Version int               `json:"version"`
		Fields  map[string]string `json:"fields"`
	}
	decodeJSON(r, &body)

	latency := httpx.QueryInt(r, "latencyMs", autosaveLatencyMs, 0, 30_000)
	if err := sleep(r.Context(), time.Duration(latency)*time.Millisecond); err != nil {
		return // the tab went away mid-write, which is what the guard is for
	}

	sess := session.MustFromContext(r.Context())
	view, accepted := recordFor(sess).write(body.Version, body.Fields, autosaveWriterPage, sess.Clock.Now())
	if !accepted {
		// The refusal carries the record the caller was behind, so resolving a
		// conflict does not need a second round trip. A client that had to
		// fetch it would be resolving against a record that may have moved
		// again in between, which is a second conflict hiding inside the first.
		httpx.JSON(w, http.StatusConflict, autosaveResponse{
			Record: view,
			Status: http.StatusConflict,
			Error: fmt.Sprintf("the record is at version %d, this write was against version %d",
				view.Version, body.Version),
		})
		return
	}
	httpx.JSON(w, http.StatusOK, autosaveResponse{Record: view})
}

func handleAutosaveOtherWriter(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Field string `json:"field"`
		Value string `json:"value"`
	}
	decodeJSON(r, &body)

	// A bare POST does the stock thing, so creating a conflict costs one line.
	// Naming a field means naming its value as well: defaulting half of a
	// caller's intention would write the stock name into whichever field they
	// asked about, and an empty value is a legitimate thing to want.
	if body.Field == "" {
		body.Field, body.Value = autosaveOtherWriterField, autosaveOtherWriterValue
	}

	if !knownField(body.Field) {
		// Refused rather than ignored: a caller that misspelt a field would
		// otherwise see a bumped version, believe its change landed, and go
		// looking for the bug in the page instead of in its own request.
		httpx.Fail(w, http.StatusBadRequest, fmt.Sprintf("no such field %q", body.Field))
		return
	}

	sess := session.MustFromContext(r.Context())
	view := recordFor(sess).otherWriter(body.Field, body.Value, sess.Clock.Now())
	httpx.JSON(w, http.StatusOK, autosaveResponse{Record: view})
}
