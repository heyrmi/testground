package app

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

const (
	optimisticStateKey  = "optimistic-revert"
	optimisticTaskCount = 8
	optimisticLatencyMs = 800
	// Every third task is refused. A fixed rule rather than a random one means
	// a test can state which outcome it expects instead of racing it.
	optimisticRejectEvery = 3
)

func optimisticRevert() challenge.Challenge {
	return challenge.Challenge{
		ID:       "optimistic-revert",
		Title:    "Optimistic update that reverts",
		URL:      "/app/optimistic-revert",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "D. Dynamic Content & DOM Instability",
		Summary: "Toggling a task flips it immediately, before the server has agreed. Eight " +
			"hundred milliseconds later the server refuses every third task and the row " +
			"flips back on its own.",
		WhyHard: "An assertion that runs straight after the click reads a value the client " +
			"invented and the server is about to discard. The test passes, the feature is " +
			"broken, and nothing in the run says so. Waiting a fixed time instead just " +
			"moves the race.",
		Hint: "Assert after the write has settled rather than after the click. Each row " +
			"carries a marker while its request is in flight, and the tasks endpoint says " +
			"in advance which ids the server will refuse, so a test can commit to an " +
			"expected outcome instead of accepting whichever one it happens to observe.",
		Tags:     []string{"optimistic-ui", "race", "state", "revert"},
		Concepts: []string{"optimistic updates", "settling before asserting", "server as source of truth", "false green"},
		Selectors: []challenge.Selector{
			{TestID: "task", Role: "listitem", Note: "One task row; narrow by its data-task-id attribute"},
			{TestID: "task-title", Note: "Task name, inside a row"},
			{TestID: "task-toggle", Role: "button", Note: "Toggles the row, inside a row"},
			{TestID: "task-state", Note: "Reads done or todo; this is the value that flips back"},
			{TestID: "task-saving", Transient: true, Note: "Present only while the row's request is in flight"},
			{TestID: "revert-notice", Role: "status", Transient: true, Note: "Names the last task the server refused"},
			{TestID: "rejected-count", Note: "How many toggles the server has refused so far"},
			{TestID: "latency-ms", Note: "How long the server waits before answering"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/app/optimistic-revert/tasks", Note: "Current tasks, including which ids will be refused"},
			{Method: http.MethodPost, Path: "/api/app/optimistic-revert/tasks/{id}/toggle", Note: "Answers 200 and flips, or 409 and does not"},
		},
		Controls: []challenge.Control{
			{
				Name:    "latencyMs",
				Kind:    "query",
				Default: fmt.Sprint(optimisticLatencyMs),
				Note:    "Milliseconds the toggle endpoint waits before answering, clamped to 0-30000.",
			},
		},
		Stability: challenge.Stable,
	}
}

// Task state lives on the session, so two workers toggling at the same time
// never see each other's rows.
type task struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
	// Rejects is published rather than hidden: the challenge is the timing,
	// not guessing which row will fail.
	Rejects bool `json:"rejects"`
}

type taskList struct {
	mu    sync.Mutex
	tasks []task
}

func tasksFor(sess *session.Session) *taskList {
	return session.Value(sess, optimisticStateKey, func() *taskList {
		stream := sess.RNG.Stream(optimisticStateKey)
		tasks := make([]task, optimisticTaskCount)
		for i := range tasks {
			tasks[i] = task{
				ID:      i + 1,
				Title:   taskVerbs[stream.IntN(len(taskVerbs))] + " " + taskNouns[stream.IntN(len(taskNouns))],
				Rejects: i%optimisticRejectEvery == optimisticRejectEvery-1,
			}
		}
		return &taskList{tasks: tasks}
	})
}

func (l *taskList) all() []task {
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]task, len(l.tasks))
	copy(out, l.tasks)
	return out
}

// toggle flips the task unless the server refuses it, and reports the
// authoritative row either way so the client can correct itself.
func (l *taskList) toggle(id int) (updated task, found, accepted bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for i := range l.tasks {
		if l.tasks[i].ID != id {
			continue
		}
		if l.tasks[i].Rejects {
			return l.tasks[i], true, false
		}
		l.tasks[i].Done = !l.tasks[i].Done
		return l.tasks[i], true, true
	}
	return task{}, false, false
}

type tasksResponse struct {
	Tasks []task `json:"tasks"`
}

type toggleResponse struct {
	Task     task   `json:"task"`
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason,omitempty"`
}

func handleOptimisticTasks(w http.ResponseWriter, r *http.Request) {
	sess := session.MustFromContext(r.Context())
	httpx.JSON(w, http.StatusOK, tasksResponse{Tasks: tasksFor(sess).all()})
}

func handleOptimisticToggle(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "task id must be a number")
		return
	}

	latency := httpx.QueryInt(r, "latencyMs", optimisticLatencyMs, 0, 30_000)
	if err := sleep(r.Context(), time.Duration(latency)*time.Millisecond); err != nil {
		return // the client went away mid-flight, which is itself a lesson
	}

	sess := session.MustFromContext(r.Context())
	updated, found, accepted := tasksFor(sess).toggle(id)
	switch {
	case !found:
		httpx.Fail(w, http.StatusNotFound, "no such task")
	case !accepted:
		httpx.JSON(w, http.StatusConflict, toggleResponse{
			Task:   updated,
			Reason: "this task is locked; the server refuses to change it",
		})
	default:
		httpx.JSON(w, http.StatusOK, toggleResponse{Task: updated, Accepted: true})
	}
}

// sleep stalls for d unless the request is abandoned first.
func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

var (
	taskVerbs = []string{"Archive", "Reconcile", "Publish", "Retire", "Approve", "Migrate", "Audit", "Rotate"}
	taskNouns = []string{
		"the quarterly export", "the staging database", "the pricing table",
		"the onboarding emails", "the access tokens", "the vendor contract",
		"the backup schedule", "the feature flags",
	}
)
