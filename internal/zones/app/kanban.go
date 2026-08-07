package app

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/hub"
	"github.com/heyrmi/testground/internal/session"
)

const (
	kanbanStateKey = "kanban"
	// kanbanDoneColumn is one way. A finished card refusing to go back gives
	// the offline queue a refusal that has nothing to do with capacity, so a
	// flush cannot be read as "the column was full" every time something
	// bounced.
	kanbanDoneColumn = "done"
	// kanbanSocketLifetime bounds a connection nobody closed, so an abandoned
	// browser tab cannot keep a goroutine and a watcher slot for the life of
	// the process.
	kanbanSocketLifetime = 10 * time.Minute
)

// The two message kinds a watcher sees. Both carry the whole board, so the
// difference is what caused the message rather than what is in it.
const (
	kanbanKindBoard    = "board"
	kanbanKindPresence = "presence"
)

func kanban() challenge.Challenge {
	return challenge.Challenge{
		ID:       "kanban",
		Title:    "Kanban with drag, realtime sync and an offline queue",
		URL:      "/app/kanban",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "V. Composite Scenarios",
		Summary: "Three columns of cards moved by pointer drag. Every accepted move is " +
			"broadcast on a socket, so a card moved through the API lands on the board " +
			"with nothing for a test to have clicked. A control takes the board offline: " +
			"moves then queue in the browser and the board shows them applied, and " +
			"reconnecting replays the queue against the board as it is by then, refusing " +
			"whatever no longer fits.",
		WhyHard: "Four hazards compose here and a test has to survive all of them in order. " +
			"The board changes with nothing the test did to wait after, so an assertion " +
			"timed off the last click reads whichever version happened to be on screen. " +
			"The drop target is chosen on the last pointer move and used on release, so a " +
			"card arriving between the two sends the drop to a position nobody aimed at -- " +
			"and the card that was aimed at is still there, one place further down, which " +
			"is why the board looks plausible afterwards. Offline, the board is a local " +
			"fiction: it shows moves the server has not seen and will partly refuse, so " +
			"every assertion made in that state is about the browser rather than about the " +
			"product. And the queue is replayed against the board as it stands at " +
			"reconnection rather than as it stood when the moves were made, so which of " +
			"them survive depends on what the other writer did in between.",
		Hint: "Wait for the board's revision to move rather than for a duration, and read a " +
			"card's place from its own attributes rather than from where it sits in a list " +
			"you located earlier. Treat offline as a state to leave before asserting " +
			"anything about the server: the flush reports what it applied and what it " +
			"refused, and the board that comes back with it is the authoritative one. When " +
			"you need the other writer, drive it through the move endpoint from the test " +
			"itself, so the arrival is something you caused and can wait for instead of " +
			"something you hope has happened. And if a drop has to survive the board " +
			"moving, re-establish the target after the change rather than releasing on a " +
			"position that was computed before it.",
		Tags: []string{"composite", "kanban", "drag", "realtime", "offline", "websocket"},
		Concepts: []string{
			"state that arrives without being asked for",
			"a drop target computed before the board moved",
			"an optimistic queue against server truth",
			"presence as something to assert rather than sleep on",
			"replay order decides which moves survive",
		},
		Selectors: []challenge.Selector{
			{TestID: "column", Note: "One column; narrow by data-column, which is todo, doing or done"},
			{TestID: "column-count", Note: "How many cards are in that column, inside it"},
			{TestID: "column-limit", Note: "The most that column will hold; only the limited column renders one"},
			{TestID: "card", Note: "One card; narrow by data-card-id. It carries data-column and data-position, which are the authoritative place to read its position from"},
			{TestID: "board-rev", Note: "Rises on every accepted move, from any writer"},
			{TestID: "watchers", Note: "How many sockets are watching this session's board"},
			{TestID: "last-event", Note: "board or presence, whichever the socket last delivered; none before the first"},
			{TestID: "connection-state", Note: "online, connecting or offline"},
			{TestID: "offline-toggle", Role: "button", Note: "Stops the board reaching the server, and reconnecting flushes what queued up"},
			{TestID: "queued-count", Note: "Moves waiting to be sent; zero while online"},
			{TestID: "queued-move", Transient: true, Note: "One per queued move, in the order it will be replayed"},
			{TestID: "drop-target", Transient: true, Note: "Where a release would land, present only while a card is held"},
			{TestID: "board-changed", Transient: true, Note: "Says the board moved during the drag that is in progress, which is the drop target going stale"},
			{TestID: "last-drop", Transient: true, Note: "The move the page last asked for, which is not always the move it got"},
			{TestID: "refusal", Transient: true, Note: "Why the server refused a move made while online"},
			{TestID: "flush-note", Transient: true, Note: "How much of the queue the server took, after reconnecting"},
			{TestID: "refused-move", Transient: true, Note: "One per queued move the server would not take; narrow by data-card"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/app/kanban/board", Note: "The board, its revision and the current watcher count"},
			{Method: http.MethodPost, Path: "/api/app/kanban/moves", Note: "Moves one card, and broadcasts it; this is also how a test plays the other writer"},
			{Method: http.MethodPost, Path: "/api/app/kanban/queue", Note: "Replays a list of moves in order, reporting which were refused and why"},
			{Method: http.MethodGet, Path: "/api/app/kanban/socket", Note: "WebSocket; pushes the whole board on every change and on every arrival or departure"},
		},
		Stability: challenge.Stable,
	}
}

// kanbanColumnDef declares a column. Limit is zero for a column that takes as
// many cards as it is given.
type kanbanColumnDef struct {
	ID    string
	Title string
	Limit int
}

// The board is fixed rather than generated. A board whose cards changed with
// the seed would make every assertion about a position seed-dependent, and the
// exercise is the sequence of hazards rather than the data.
var kanbanColumns = []kanbanColumnDef{
	{ID: "todo", Title: "To do"},
	{ID: "doing", Title: "Doing", Limit: 2},
	{ID: kanbanDoneColumn, Title: "Done"},
}

var kanbanSeedCards = []struct{ ID, Title, Column string }{
	{"card-1", "Reproduce the flake", "todo"},
	{"card-2", "Pin the failing seed", "todo"},
	{"card-3", "Delete the last sleep", "todo"},
	{"card-4", "Split the shared fixture", "doing"},
	{"card-5", "Retire the staging shim", kanbanDoneColumn},
	{"card-6", "Write up the postmortem", kanbanDoneColumn},
}

// kanbanError is a distinct type per refusal because "there is no such card",
// "that column is full" and "done is one way" are three different things to
// explain to someone reading a flush report, and three different assertions.
type kanbanError string

func (e kanbanError) Error() string { return string(e) }

const (
	kanbanErrNoSuchCard   = kanbanError("no such card")
	kanbanErrNoSuchColumn = kanbanError("no such column")
	kanbanErrColumnFull   = kanbanError("that column is already at its limit")
	kanbanErrDoneIsFinal  = kanbanError("done is one way; a finished card does not come back")
)

// kanbanCard is one card as the board publishes it.
type kanbanCard struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Column string `json:"column"`
}

// kanbanColumnView is one column and its cards, in order.
type kanbanColumnView struct {
	ID    string       `json:"id"`
	Title string       `json:"title"`
	Limit int          `json:"limit"`
	Cards []kanbanCard `json:"cards"`
}

// kanbanSnapshot is the whole board. Rev rises on every accepted move, and
// Watchers is the presence count at the moment the snapshot was taken.
type kanbanSnapshot struct {
	Rev      int                `json:"rev"`
	Columns  []kanbanColumnView `json:"columns"`
	Watchers int                `json:"watchers"`
}

// kanbanState is one session's board and the hub that fans its changes out.
// Both belong to the session: two parallel workers get two boards and two
// hubs, so a presence count is a count of that worker's own tabs rather than
// of whatever else happened to be running at the time.
type kanbanState struct {
	hub *hub.Hub

	// publishing keeps a snapshot and the sequence number it goes out under
	// together. Taking the board and then numbering it are two steps, so
	// without this two writers interleave: one reads the board, the other
	// reads a newer board and numbers it first, and the older board arrives
	// carrying the higher number. A page that keeps the highest sequence it
	// has seen then pins a board the server has already moved past, and
	// because nothing further is due it stays wrong until the next write.
	publishing sync.Mutex

	mu     sync.Mutex
	rev    int
	titles map[string]string
	order  map[string][]string
}

func kanbanFor(sess *session.Session) *kanbanState {
	return session.Value(sess, kanbanStateKey, func() *kanbanState {
		state := &kanbanState{
			hub:    hub.New(),
			titles: make(map[string]string, len(kanbanSeedCards)),
			order:  make(map[string][]string, len(kanbanColumns)),
		}
		for _, seed := range kanbanSeedCards {
			state.titles[seed.ID] = seed.Title
			state.order[seed.Column] = append(state.order[seed.Column], seed.ID)
		}
		return state
	})
}

func kanbanColumnDefOf(id string) (kanbanColumnDef, bool) {
	for _, def := range kanbanColumns {
		if def.ID == id {
			return def, true
		}
	}
	return kanbanColumnDef{}, false
}

// move relocates one card, or explains why it will not.
//
// The index is clamped rather than rejected, and it is interpreted against the
// target column with the card already taken out of it. That is the same
// arithmetic the page does, so a move the page computed and a move a test
// posted by hand land in the same place.
func (s *kanbanState) move(cardID, column string, index int) error {
	target, known := kanbanColumnDefOf(column)
	if !known {
		return kanbanErrNoSuchColumn
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.titles[cardID]; !ok {
		return kanbanErrNoSuchCard
	}
	from := s.columnOfLocked(cardID)
	switch {
	case from == kanbanDoneColumn && column != kanbanDoneColumn:
		return kanbanErrDoneIsFinal
	case from != column && target.Limit > 0 && len(s.order[column]) >= target.Limit:
		return kanbanErrColumnFull
	}

	s.order[from] = kanbanWithout(s.order[from], cardID)
	s.order[column] = slices.Insert(s.order[column], httpx.Clamp(index, 0, len(s.order[column])), cardID)
	s.rev++
	return nil
}

func (s *kanbanState) columnOfLocked(cardID string) string {
	for _, def := range kanbanColumns {
		if slices.Contains(s.order[def.ID], cardID) {
			return def.ID
		}
	}
	return ""
}

func kanbanWithout(ids []string, drop string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != drop {
			out = append(out, id)
		}
	}
	return out
}

func (s *kanbanState) snapshot() kanbanSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	columns := make([]kanbanColumnView, 0, len(kanbanColumns))
	for _, def := range kanbanColumns {
		cards := make([]kanbanCard, 0, len(s.order[def.ID]))
		for _, id := range s.order[def.ID] {
			cards = append(cards, kanbanCard{ID: id, Title: s.titles[id], Column: def.ID})
		}
		columns = append(columns, kanbanColumnView{ID: def.ID, Title: def.Title, Limit: def.Limit, Cards: cards})
	}
	return kanbanSnapshot{Rev: s.rev, Columns: columns, Watchers: s.hub.Watchers()}
}

// publish broadcasts the whole board rather than a description of what changed.
//
// The hub drops a message a watcher is too far behind to take, so a page that
// rebuilt its board from a series of edits would end up rendering a board that
// never existed on the server. Sending the state instead of the delta means a
// missed message costs a watcher nothing but latency.
func (s *kanbanState) publish(kind string) (int, kanbanSnapshot) {
	s.publishing.Lock()
	defer s.publishing.Unlock()

	snapshot := s.snapshot()
	body, err := json.Marshal(snapshot)
	if err != nil {
		return s.hub.Seq(), snapshot
	}
	return s.hub.Publish(kind, body), snapshot
}

// kanbanBoardResponse carries the sequence number the snapshot was taken at, so
// a page can tell a reply to its own write from a broadcast that overtook it.
type kanbanBoardResponse struct {
	Seq   int            `json:"seq"`
	Board kanbanSnapshot `json:"board"`
}

type kanbanMove struct {
	Card   string `json:"card"`
	Column string `json:"column"`
	Index  int    `json:"index"`
}

func handleKanbanBoard(w http.ResponseWriter, r *http.Request) {
	state := kanbanFor(session.MustFromContext(r.Context()))
	httpx.JSON(w, http.StatusOK, kanbanBoardResponse{Seq: state.hub.Seq(), Board: state.snapshot()})
}

func handleKanbanMove(w http.ResponseWriter, r *http.Request) {
	var body kanbanMove
	decodeJSON(r, &body)

	state := kanbanFor(session.MustFromContext(r.Context()))
	if err := state.move(body.Card, body.Column, body.Index); err != nil {
		status := http.StatusConflict
		if err == kanbanErrNoSuchCard || err == kanbanErrNoSuchColumn {
			status = http.StatusNotFound
		}
		// The board travels with the refusal so a page that guessed wrong can
		// correct itself from the same response, rather than needing a second
		// read that a concurrent writer could change underneath it.
		httpx.JSON(w, status, map[string]any{
			"status": status, "error": err.Error(), "card": body.Card,
			"seq": state.hub.Seq(), "board": state.snapshot(),
		})
		return
	}

	seq, snapshot := state.publish(kanbanKindBoard)
	httpx.JSON(w, http.StatusOK, kanbanBoardResponse{Seq: seq, Board: snapshot})
}

type kanbanRefusal struct {
	Card   string `json:"card"`
	Column string `json:"column"`
	Index  int    `json:"index"`
	Reason string `json:"reason"`
}

type kanbanQueueResponse struct {
	Applied int             `json:"applied"`
	Refused []kanbanRefusal `json:"refused"`
	Seq     int             `json:"seq"`
	Board   kanbanSnapshot  `json:"board"`
}

// handleKanbanQueue replays moves that were made while the page could not
// reach the server.
//
// They are applied in the order they were made, against the board as it is
// now. Nothing is reserved for them and nothing is rewound: a move that would
// have fitted when it was made is refused if the space has since gone, which
// is what makes the count that comes back depend on the other writer rather
// than only on what the page did.
func handleKanbanQueue(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Moves []kanbanMove `json:"moves"`
	}
	decodeJSON(r, &body)

	state := kanbanFor(session.MustFromContext(r.Context()))
	applied := 0
	refused := make([]kanbanRefusal, 0, len(body.Moves))
	for _, move := range body.Moves {
		if err := state.move(move.Card, move.Column, move.Index); err != nil {
			refused = append(refused, kanbanRefusal{
				Card: move.Card, Column: move.Column, Index: move.Index, Reason: err.Error(),
			})
			continue
		}
		applied++
	}

	// One broadcast for the whole flush, and none at all when every move was
	// refused: a message that carries no change would advance the sequence
	// other watchers use to tell a stale snapshot from a fresh one.
	seq, snapshot := state.hub.Seq(), state.snapshot()
	if applied > 0 {
		seq, snapshot = state.publish(kanbanKindBoard)
	}
	httpx.JSON(w, http.StatusOK, kanbanQueueResponse{
		Applied: applied, Refused: refused, Seq: seq, Board: snapshot,
	})
}

// kanbanEvent is what the socket speaks. Data is passed through verbatim
// rather than re-encoded, because the body the hub is holding is already the
// JSON snapshot that every other endpoint returns.
type kanbanEvent struct {
	Seq  int             `json:"seq"`
	Kind string          `json:"kind"`
	Data json.RawMessage `json:"data"`
}

// handleKanbanSocket streams this session's board to one connection.
//
// Nothing is read from the client: every mutation arrives over HTTP, which
// keeps the socket a one-way live region and means a test can drive the board
// without holding one open at all.
//
// A socket opened before the session is reset keeps watching a hub the reset
// replaced, so it goes quiet rather than erroring. That is the same shape as a
// server that has forgotten you and it is why the page treats connection state
// as something to render rather than to assume.
func handleKanbanSocket(w http.ResponseWriter, r *http.Request) {
	// The session is read before the upgrade: once Accept returns, the
	// request's context belongs to a hijacked connection and reading from it
	// is undefined. The connection gets a context of its own instead.
	state := kanbanFor(session.MustFromContext(r.Context()))

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// The playground is reachable on whatever host the operator chose, so
		// cross-origin sockets have to be allowed here. This is a local tool,
		// not a service.
		InsecureSkipVerify: true,
	})
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), kanbanSocketLifetime)
	defer cancel()
	defer conn.CloseNow()

	// Reading is still required or control frames are never handled, and the
	// returned context ends when the client goes away -- which is what stops
	// the watcher, the goroutine behind it and this loop all at once.
	ctx = conn.CloseRead(ctx)

	messages, stop := state.hub.Watch(ctx)
	defer func() {
		// The watcher is removed before the departure is announced, so the
		// number the remaining tabs render is the number that is actually
		// left rather than one that includes a connection already going.
		stop()
		state.publish(kanbanKindPresence)
	}()

	// Broadcast rather than written straight to this connection: a tab
	// arriving is exactly what a presence indicator exists to show, and the
	// newcomer is already watching, so one message serves it and everyone
	// else. It carries the board too, which is how this connection gets its
	// first copy.
	state.publish(kanbanKindPresence)

	for {
		select {
		case <-ctx.Done():
			return
		case message, open := <-messages:
			if !open {
				return
			}
			event := kanbanEvent{Seq: message.Seq, Kind: message.Kind, Data: message.Body}
			if err := wsjson.Write(ctx, conn, event); err != nil {
				return
			}
		}
	}
}
