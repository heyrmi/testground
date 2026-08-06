package session

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/heyrmi/testground/internal/clock"
	"github.com/heyrmi/testground/internal/rng"
)

// Defaults applied when Options leaves a field zero.
const (
	DefaultTTL         = 2 * time.Hour
	DefaultMaxSessions = 10_000
	sweepInterval      = time.Minute
)

// Options configures a Store.
type Options struct {
	// Seed every new session starts from.
	Seed uint64
	// TTL is how long a session survives without a request.
	TTL time.Duration
	// MaxSessions caps concurrent sessions; the least recently used are
	// evicted past the cap so a crawler cannot exhaust memory.
	MaxSessions int
	// Clock is the real-time source used for expiry bookkeeping and as the
	// base for each session's controllable clock.
	Clock clock.Clock
}

// Store holds every live session. It is the only owner of playground state.
type Store struct {
	opts  Options
	clock clock.Clock

	mu       sync.Mutex
	sessions map[ID]*Session
}

// NewStore returns an empty Store.
func NewStore(opts Options) *Store {
	if opts.TTL <= 0 {
		opts.TTL = DefaultTTL
	}
	if opts.MaxSessions <= 0 {
		opts.MaxSessions = DefaultMaxSessions
	}
	if opts.Clock == nil {
		opts.Clock = clock.Real{}
	}
	return &Store{
		opts:     opts,
		clock:    opts.Clock,
		sessions: make(map[ID]*Session),
	}
}

// Seed reports the seed new sessions start from.
func (s *Store) Seed() uint64 { return s.opts.Seed }

// TTL reports how long an idle session survives.
func (s *Store) TTL() time.Duration { return s.opts.TTL }

// Create opens a session with a generated id.
func (s *Store) Create() *Session { return s.Open("") }

// Open returns the session with the given id, creating it if absent. Clients
// may name their own sessions, which lets parallel workers pin a stable id.
// An empty id opens a fresh session under a generated one.
func (s *Store) Open(id ID) *Session {
	if id == "" {
		id = newID()
	}
	now := s.clock.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.sessions[id]; ok {
		existing.touch(now)
		return existing
	}

	created := &Session{
		ID:       id,
		Clock:    clock.New(s.opts.Clock),
		RNG:      rng.New(s.opts.Seed),
		created:  now,
		baseSeed: s.opts.Seed,
		lastSeen: now,
	}
	s.sessions[id] = created
	s.evictLocked(now)
	return created
}

// Get returns an existing session without creating one.
func (s *Store) Get(id ID) (*Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	found, ok := s.sessions[id]
	if ok {
		found.touch(s.clock.Now())
	}
	return found, ok
}

// Delete drops a session and everything it owned.
func (s *Store) Delete(id ID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.sessions[id]
	delete(s.sessions, id)
	return ok
}

// Len reports the number of live sessions.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

// IDs lists live session ids in a stable order.
func (s *Store) IDs() []ID {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]ID, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// Sweep drops sessions idle for longer than the TTL and returns how many went.
func (s *Store) Sweep() int {
	now := s.clock.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	dropped := 0
	for id, sess := range s.sessions {
		if now.Sub(sess.LastSeen()) > s.opts.TTL {
			delete(s.sessions, id)
			dropped++
		}
	}
	return dropped
}

// Run sweeps expired sessions until ctx is cancelled.
func (s *Store) Run(ctx context.Context) {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Sweep()
		}
	}
}

// evictLocked enforces MaxSessions by dropping the least recently seen.
func (s *Store) evictLocked(now time.Time) {
	over := len(s.sessions) - s.opts.MaxSessions
	if over <= 0 {
		return
	}

	type aged struct {
		id   ID
		seen time.Time
	}
	all := make([]aged, 0, len(s.sessions))
	for id, sess := range s.sessions {
		all = append(all, aged{id: id, seen: sess.LastSeen()})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].seen.Before(all[j].seen) })

	for _, victim := range all[:over] {
		delete(s.sessions, victim.id)
	}
}

type contextKey struct{}

// NewContext returns ctx carrying sess.
func NewContext(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, contextKey{}, sess)
}

// FromContext returns the session attached by the middleware, or nil.
func FromContext(ctx context.Context) *Session {
	sess, _ := ctx.Value(contextKey{}).(*Session)
	return sess
}

// MustFromContext returns the session attached by the middleware and panics if
// there is none, which can only mean a handler was mounted outside it.
func MustFromContext(ctx context.Context) *Session {
	sess := FromContext(ctx)
	if sess == nil {
		panic("session: handler mounted without session middleware")
	}
	return sess
}
