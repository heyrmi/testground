package app

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/control"
	"github.com/heyrmi/testground/internal/fake"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

const (
	adminStateKey  = "admin-crud"
	adminSeeded    = 12
	adminLatencyMs = 800
	adminUndoMs    = 4000
	// Every fourth seeded account refuses to change. A published rule rather
	// than a random one means a test can state which outcome it is exercising
	// instead of racing whichever one it happens to observe, and with the roles
	// dealt round-robin it also leaves exactly one locked account in each role.
	adminLockEvery = 4
)

func adminCRUD() challenge.Challenge {
	return challenge.Challenge{
		ID:       "admin-crud",
		Title:    "Admin CRUD with optimistic writes and rollback",
		URL:      "/app/admin-crud",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "V. Composite Scenarios",
		Summary: "A table of accounts you can create, rename and delete. Every write lands on " +
			"the page before the server has agreed to it and is confirmed or undone " +
			"eight hundred milliseconds later. A delete does not even leave the browser " +
			"straight away: the row goes, a toast offers to put it back, and the request " +
			"is only sent once that window closes. Three of the twelve seeded accounts " +
			"are locked and the server refuses to change or delete them, and a create is " +
			"refused when the name is already taken. Select-all covers the rows the " +
			"filters leave on screen; the bulk delete takes everything selected.",
		WhyHard: "Four different ways to assert on a state that never existed on the server, " +
			"in one flow. A created row appears at once carrying a temporary id, so a " +
			"locator built from it stops matching the moment the server answers with the " +
			"real one, and the row it pointed at is gone rather than renamed. An edit or " +
			"a delete the server refuses is undone eight hundred milliseconds later, so " +
			"a row that is missing now can be back before the test ends and the " +
			"assertion in between passed against nothing. A queued delete has not been " +
			"sent at all, so reading the server immediately after the click finds the " +
			"row still there -- and waiting for the toast to disappear is waiting out a " +
			"duration rather than an outcome, because the request only starts when the " +
			"toast ends. And select-all means the rows the filters left on screen rather " +
			"than the table: tick it, change the filter, and that selection is still " +
			"there and still about to be deleted, though the box now reads unchecked and " +
			"the rows it covers are nowhere on the page.",
		Hint: "Nothing here settles at one moment, so the page publishes the two things " +
			"that have to settle separately: how many deletes are still waiting for " +
			"their undo window, and how many writes the server has not answered. The " +
			"page and the server agree only when both read zero, and asserting before " +
			"that is asserting on a guess. Find a created row by something that survives " +
			"the round trip rather than by the id it was born with. Treat a locked " +
			"account and a duplicate name as outcomes you choose rather than accidents " +
			"you wait out -- both are published before you write. And read what " +
			"select-all actually selected rather than what the filter happens to be " +
			"showing.",
		Tags: []string{"composite", "crud", "optimistic-ui", "undo", "bulk-actions", "rollback"},
		Concepts: []string{
			"optimistic writes and rollback",
			"an undo window delays the request",
			"identity changes on confirmation",
			"select-all means the filtered set",
			"settling has more than one meaning",
		},
		Selectors: []challenge.Selector{
			{TestID: "account-row", Role: "row", Note: "One per account on screen; narrow by data-id, or by data-locked to find the rows the server refuses"},
			{TestID: "account-name", Note: "Inside a row; replaced by an input while that row is being edited"},
			{TestID: "account-role", Note: "Inside a row; admin, editor or viewer"},
			{TestID: "account-state", Note: "Inside a row; reads saved, creating or saving, and only saved means the server agrees"},
			{TestID: "account-locked", Note: "Inside a row, on the accounts the server refuses to change or delete"},
			{TestID: "select-row", Role: "checkbox", Note: "Inside a row"},
			{TestID: "row-edit", Role: "button", Note: "Inside a row; turns the name and role into inputs"},
			{TestID: "row-delete", Role: "button", Note: "Inside a row; opens the undo window rather than sending anything"},
			{TestID: "search", Role: "searchbox", Note: "Filters by name, combining with the role filter"},
			{TestID: "role-filter", Role: "combobox", Note: "Filters by role, combining with the search"},
			{TestID: "select-all", Role: "checkbox", Note: "Selects the rows the filters leave on screen, and adds them to a selection that survives the filter changing"},
			{TestID: "selected-count", Note: "How many accounts are selected, on screen or not"},
			{TestID: "row-count", Note: "How many rows the filters leave, of how many the page holds"},
			{TestID: "bulk-delete", Role: "button", Note: "Deletes everything selected, including rows the filter is hiding"},
			{TestID: "new-name", Role: "textbox", Note: "The name for a new account; a name already in the table is refused"},
			{TestID: "new-role", Role: "combobox", Note: "The role for a new account"},
			{TestID: "create-account", Role: "button", Note: "Adds the row immediately, with an id the server has not issued yet"},
			{TestID: "edit-name", Role: "textbox", Transient: true, Note: "Replaces the name of the row being edited"},
			{TestID: "edit-role", Role: "combobox", Transient: true, Note: "Replaces the role of the row being edited"},
			{TestID: "edit-save", Role: "button", Transient: true, Note: "Applies the edit to the page and then asks the server"},
			{TestID: "edit-cancel", Role: "button", Transient: true, Note: "Leaves the row alone; nothing is sent"},
			{TestID: "undo-toast", Role: "status", Transient: true, Note: "Present exactly while a delete is queued and unsent"},
			{TestID: "undo-delete", Role: "button", Transient: true, Note: "Puts the rows back; the server never hears about the delete"},
			{TestID: "rollback-notice", Role: "status", Transient: true, Note: "Names the last write the server refused and undid"},
			{TestID: "rollback-count", Note: "How many writes the server has refused so far"},
			{TestID: "queued-deletes", Note: "Rows deleted on screen whose request has not been sent yet"},
			{TestID: "in-flight", Note: "Writes the server has not answered yet"},
			{TestID: "reload", Role: "button", Note: "Replaces the table with what the server actually holds"},
			{TestID: "latency-ms", Note: "How long the server waits before answering a write"},
			{TestID: "undo-ms", Note: "How long a delete waits in the browser before it is sent"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/app/admin-crud/accounts", Note: "Every account this session holds, and the roles it accepts"},
			{Method: http.MethodPost, Path: "/api/app/admin-crud/accounts", Note: "Creates one and issues its id; refuses a name already taken"},
			{Method: http.MethodPatch, Path: "/api/app/admin-crud/accounts/{id}", Note: "Renames or re-roles one; refuses a locked account"},
			{Method: http.MethodDelete, Path: "/api/app/admin-crud/accounts/{id}", Note: "Deletes one; refuses a locked account"},
		},
		Controls: []challenge.Control{
			{
				Name:    "latencyMs",
				Kind:    "query",
				Default: fmt.Sprint(adminLatencyMs),
				Note:    "Milliseconds the write endpoints wait before answering, clamped to 0-30000.",
			},
			{
				Name:    "undoMs",
				Kind:    "query",
				Default: fmt.Sprint(adminUndoMs),
				Note: "Milliseconds a delete waits in the browser before its request is sent, " +
					"clamped to 0-30000. At zero the toast still appears and the request " +
					"leaves immediately.",
			},
			{
				Name:    "flake",
				Kind:    "control-plane",
				Default: "0",
				Note: "POST /api/control/flake {\"challenge\":\"admin-crud\"} refuses that share of " +
					"writes the server would otherwise have accepted, so any row can be made " +
					"to roll back rather than only the ones published as locked.",
			},
		},
		Stability: challenge.Stable,
	}
}

// adminAccount is one row of the admin table.
type adminAccount struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
	// Locked is published rather than hidden: the exercise is surviving a
	// rollback, not guessing which row will produce one.
	Locked bool `json:"locked"`
}

// adminRoles is a closed set, so a role filter has a fixed vocabulary and a
// mistyped role is a refusal rather than a new category appearing in the table.
var adminRoles = []string{"admin", "editor", "viewer"}

func validAdminRole(role string) bool {
	for _, known := range adminRoles {
		if role == known {
			return true
		}
	}
	return false
}

// adminBook is one session's accounts. It lives on the session so two workers
// creating an account with the same name never refuse each other.
type adminBook struct {
	mu       sync.Mutex
	accounts []adminAccount
	// next issues the ids the client cannot predict, which is what makes the
	// temporary id an optimistic row carries different from the real one.
	next int
}

func adminBookFor(sess *session.Session) *adminBook {
	return session.Value(sess, adminStateKey, func() *adminBook {
		stream := sess.RNG.Stream(adminStateKey)
		accounts := make([]adminAccount, adminSeeded)
		for i := range accounts {
			person := fake.NewPerson(stream, i)
			accounts[i] = adminAccount{
				ID:     fmt.Sprintf("acct-%d", i+1),
				Name:   person.Name,
				Email:  person.Email,
				Role:   adminRoles[i%len(adminRoles)],
				Locked: i%adminLockEvery == adminLockEvery-1,
			}
		}
		return &adminBook{accounts: accounts, next: adminSeeded + 1}
	})
}

// adminError names which refusal it was, because "locked", "already taken" and
// "refused this time" are three different things to an operator and three
// different assertions to a test.
type adminError string

func (e adminError) Error() string { return string(e) }

const (
	errAdminNoName    = adminError("an account needs a name")
	errAdminBadRole   = adminError("role must be admin, editor or viewer")
	errAdminDuplicate = adminError("an account with that name already exists")
	errAdminLocked    = adminError("this account is locked; the server refuses to change it")
	errAdminRefused   = adminError("the server refused this write on this occasion")
)

func (b *adminBook) all() []adminAccount {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]adminAccount, 0, len(b.accounts))
	return append(out, b.accounts...)
}

// create issues an account, or explains why it will not. refuse is asked last,
// once the request is one the server would otherwise have accepted: a flake
// decision draws from the session's seeded stream, so letting a malformed
// request consume a draw would shift every later decision and stop the same
// sequence of real writes replaying.
func (b *adminBook) create(name, role string, refuse func() bool) (adminAccount, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return adminAccount{}, errAdminNoName
	case !validAdminRole(role):
		return adminAccount{}, errAdminBadRole
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for _, existing := range b.accounts {
		if strings.EqualFold(existing.Name, name) {
			return adminAccount{}, errAdminDuplicate
		}
	}
	if refuse() {
		return adminAccount{}, errAdminRefused
	}

	created := adminAccount{
		ID:    fmt.Sprintf("acct-%d", b.next),
		Name:  name,
		Email: adminEmail(name),
		Role:  role,
	}
	b.next++
	b.accounts = append(b.accounts, created)
	return created, nil
}

// update renames and re-roles an account in one write, reporting the row the
// server holds either way so a client that guessed wrong can correct itself
// from the answer rather than from a second request.
func (b *adminBook) update(id, name, role string, refuse func() bool) (current adminAccount, found bool, err error) {
	name = strings.TrimSpace(name)

	b.mu.Lock()
	defer b.mu.Unlock()

	for i := range b.accounts {
		if b.accounts[i].ID != id {
			continue
		}
		switch {
		case name == "":
			return b.accounts[i], true, errAdminNoName
		case !validAdminRole(role):
			return b.accounts[i], true, errAdminBadRole
		case b.accounts[i].Locked:
			return b.accounts[i], true, errAdminLocked
		}
		for j := range b.accounts {
			if j != i && strings.EqualFold(b.accounts[j].Name, name) {
				return b.accounts[i], true, errAdminDuplicate
			}
		}
		if refuse() {
			return b.accounts[i], true, errAdminRefused
		}

		b.accounts[i].Name = name
		b.accounts[i].Role = role
		return b.accounts[i], true, nil
	}
	return adminAccount{}, false, nil
}

func (b *adminBook) remove(id string, refuse func() bool) (current adminAccount, found bool, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := range b.accounts {
		if b.accounts[i].ID != id {
			continue
		}
		if b.accounts[i].Locked {
			return b.accounts[i], true, errAdminLocked
		}
		if refuse() {
			return b.accounts[i], true, errAdminRefused
		}

		removed := b.accounts[i]
		b.accounts = append(b.accounts[:i], b.accounts[i+1:]...)
		return removed, true, nil
	}
	return adminAccount{}, false, nil
}

// adminEmail derives an address from a name, so a created account gains a field
// nobody typed and the optimistic row is missing something real.
func adminEmail(name string) string {
	parts := strings.Fields(strings.ToLower(name))
	return strings.Join(parts, ".") + "@example.test"
}

type adminListResponse struct {
	Accounts []adminAccount `json:"accounts"`
	Roles    []string       `json:"roles"`
}

// adminWriteResponse is the one shape every write answers with, refusal
// included. Carrying the row the server holds even when it says no is what lets
// the page put back exactly what was there rather than what it remembered.
type adminWriteResponse struct {
	Account  *adminAccount `json:"account,omitempty"`
	Accepted bool          `json:"accepted"`
	Status   int           `json:"status,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// adminStatus separates a request the server could not read from one it read
// and refused. Only the second is the rollback this page is about, and a client
// that cannot tell them apart reports a typed role as a locked account.
func adminStatus(err error) int {
	switch err {
	case errAdminNoName, errAdminBadRole:
		return http.StatusBadRequest
	default:
		return http.StatusConflict
	}
}

// adminRefuser defers the flake decision until the caller has found the row, so
// a request naming an account that does not exist neither consumes a draw nor
// puts X-Playground-Flaked on a 404.
func adminRefuser(w http.ResponseWriter, r *http.Request) func() bool {
	return func() bool { return control.Flaked(w, r, adminStateKey) }
}

// adminDelay is the window the optimistic row lives in: long enough that a test
// asserting straight after the click reads the value the client invented.
func adminDelay(r *http.Request) error {
	latency := httpx.QueryInt(r, "latencyMs", adminLatencyMs, 0, 30_000)
	return sleep(r.Context(), time.Duration(latency)*time.Millisecond)
}

func handleAdminAccounts(w http.ResponseWriter, r *http.Request) {
	sess := session.MustFromContext(r.Context())
	httpx.JSON(w, http.StatusOK, adminListResponse{
		Accounts: adminBookFor(sess).all(),
		Roles:    adminRoles,
	})
}

func handleAdminCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	decodeJSON(r, &body)

	if err := adminDelay(r); err != nil {
		return // the client went away mid-flight, which is itself a lesson
	}

	sess := session.MustFromContext(r.Context())
	created, err := adminBookFor(sess).create(body.Name, body.Role, adminRefuser(w, r))
	if err != nil {
		status := adminStatus(err)
		httpx.JSON(w, status, adminWriteResponse{Status: status, Error: err.Error()})
		return
	}
	httpx.JSON(w, http.StatusCreated, adminWriteResponse{Account: &created, Accepted: true})
}

func handleAdminUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Role string `json:"role"`
	}
	decodeJSON(r, &body)

	if err := adminDelay(r); err != nil {
		return
	}

	sess := session.MustFromContext(r.Context())
	current, found, err := adminBookFor(sess).update(chi.URLParam(r, "id"), body.Name, body.Role, adminRefuser(w, r))
	switch {
	case !found:
		httpx.Fail(w, http.StatusNotFound, "no such account")
	case err != nil:
		status := adminStatus(err)
		httpx.JSON(w, status, adminWriteResponse{Account: &current, Status: status, Error: err.Error()})
	default:
		httpx.JSON(w, http.StatusOK, adminWriteResponse{Account: &current, Accepted: true})
	}
}

func handleAdminDelete(w http.ResponseWriter, r *http.Request) {
	if err := adminDelay(r); err != nil {
		return
	}

	sess := session.MustFromContext(r.Context())
	current, found, err := adminBookFor(sess).remove(chi.URLParam(r, "id"), adminRefuser(w, r))
	switch {
	case !found:
		httpx.Fail(w, http.StatusNotFound, "no such account")
	case err != nil:
		httpx.JSON(w, http.StatusConflict, adminWriteResponse{
			Account: &current, Status: http.StatusConflict, Error: err.Error(),
		})
	default:
		httpx.JSON(w, http.StatusOK, adminWriteResponse{Account: &current, Accepted: true})
	}
}
