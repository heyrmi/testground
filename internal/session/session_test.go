package session

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/heyrmi/testground/internal/clock"
)

type counter struct {
	mu sync.Mutex
	n  int
}

func (c *counter) inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
}

func (c *counter) get() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

func newCounter() *counter { return &counter{} }

func newStore() *Store { return NewStore(Options{Seed: 42}) }

func TestSessionsDoNotShareState(t *testing.T) {
	store := newStore()

	a := store.Open("worker-1")
	b := store.Open("worker-2")

	Value(a, "counter", newCounter).inc()
	Value(a, "counter", newCounter).inc()

	if got := Value(b, "counter", newCounter).get(); got != 0 {
		t.Fatalf("worker-2 observed worker-1's mutations: counter = %d", got)
	}
}

func TestValueCreatesOnceUnderConcurrency(t *testing.T) {
	store := newStore()
	sess := store.Create()

	created := newCounter()
	var wg sync.WaitGroup
	for range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Value(sess, "shared", func() *counter {
				created.inc()
				return newCounter()
			}).inc()
		}()
	}
	wg.Wait()

	if got := created.get(); got != 1 {
		t.Fatalf("factory ran %d times, want 1", got)
	}
	if got := Value(sess, "shared", newCounter).get(); got != 64 {
		t.Fatalf("lost writes: counter = %d, want 64", got)
	}
}

func TestResetClearsStateButKeepsSeed(t *testing.T) {
	store := newStore()
	sess := store.Create()

	Value(sess, "counter", newCounter).inc()
	sess.Reseed(99)
	sess.Clock.Freeze()
	sess.Reset()

	if got := Value(sess, "counter", newCounter).get(); got != 0 {
		t.Fatalf("reset left state behind: counter = %d", got)
	}
	if sess.Clock.Frozen() {
		t.Fatal("reset left the clock frozen")
	}
	if got := sess.RNG.Seed(); got != 99 {
		t.Fatalf("reset discarded the chosen seed: got %d, want 99", got)
	}
}

func TestReseedDiscardsDerivedState(t *testing.T) {
	store := newStore()
	sess := store.Create()

	Value(sess, "counter", newCounter).inc()
	sess.Reseed(7)

	if got := Value(sess, "counter", newCounter).get(); got != 0 {
		t.Fatalf("reseed kept state derived from the old seed: counter = %d", got)
	}
}

func TestClocksAreIsolatedPerSession(t *testing.T) {
	store := newStore()

	frozen := store.Open("frozen")
	running := store.Open("running")
	frozen.Clock.Freeze()

	if running.Clock.Frozen() {
		t.Fatal("freezing one session froze another")
	}
}

func TestSweepDropsIdleSessions(t *testing.T) {
	base := clock.New(clock.Real{})
	store := NewStore(Options{TTL: time.Minute, Clock: base})
	store.Open("stale")

	base.Advance(2 * time.Minute)

	if dropped := store.Sweep(); dropped != 1 {
		t.Fatalf("dropped %d sessions, want 1", dropped)
	}
	if store.Len() != 0 {
		t.Fatalf("%d sessions survived the sweep", store.Len())
	}
}

func TestEvictionEnforcesTheCap(t *testing.T) {
	base := clock.New(clock.Real{})
	store := NewStore(Options{MaxSessions: 2, Clock: base})

	store.Open("oldest")
	base.Advance(time.Second)
	store.Open("middle")
	base.Advance(time.Second)
	store.Open("newest")

	if store.Len() != 2 {
		t.Fatalf("%d sessions live, want 2", store.Len())
	}
	if _, ok := store.Get("oldest"); ok {
		t.Fatal("eviction kept the least recently seen session")
	}
}

func TestMiddlewareHonoursTheHeaderOverTheCookie(t *testing.T) {
	store := newStore()
	handler := store.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(MustFromContext(r.Context()).ID))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(Header, "from-header")
	req.AddCookie(&http.Cookie{Name: Cookie, Value: "from-cookie"})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Body.String(); got != "from-header" {
		t.Fatalf("handler saw session %q, want from-header", got)
	}
	if got := rec.Header().Get(Header); got != "from-header" {
		t.Fatalf("response advertised session %q, want from-header", got)
	}
}

func TestMiddlewareReusesTheCookieSession(t *testing.T) {
	store := newStore()
	handler := store.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))

	id := first.Header().Get(Header)
	if id == "" {
		t.Fatal("no session id advertised on a cold request")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: Cookie, Value: id})
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, req)

	if got := second.Header().Get(Header); got != id {
		t.Fatalf("second request landed in %q, want %q", got, id)
	}
	if cookies := second.Result().Cookies(); len(cookies) != 0 {
		t.Fatalf("re-set a cookie that was already correct: %v", cookies)
	}
	if store.Len() != 1 {
		t.Fatalf("%d sessions created, want 1", store.Len())
	}
}

func TestMiddlewareRejectsAMalformedHeader(t *testing.T) {
	store := newStore()
	handler := store.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler ran for a malformed session id")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(Header, "not a valid id")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", rec.Code)
	}
}

func TestMiddlewareIgnoresACorruptCookie(t *testing.T) {
	store := newStore()
	handler := store.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: Cookie, Value: "wrecked value"})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	if rec.Header().Get(Header) == "" {
		t.Fatal("no replacement session issued")
	}
}
