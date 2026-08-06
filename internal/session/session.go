// Package session gives every client its own copy of the playground.
//
// Two parallel test workers must never observe each other's mutations, so no
// challenge state lives in package-level variables. Everything mutable hangs
// off a Session, which handlers reach through the request context.
package session

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
	"sync"
	"time"

	"github.com/heyrmi/testground/internal/clock"
	"github.com/heyrmi/testground/internal/rng"
)

// ID identifies one isolated copy of the playground.
type ID string

// Session owns every piece of mutable state one client can observe: its own
// clock, its own seed, and a lazily-populated bag of per-challenge state.
type Session struct {
	ID    ID
	Clock *clock.Controllable
	RNG   *rng.Source

	created  time.Time
	baseSeed uint64

	mu       sync.Mutex
	lastSeen time.Time
	data     map[string]any
}

// Created reports when the session was opened.
func (s *Session) Created() time.Time { return s.created }

// LastSeen reports when a request last touched the session. It is measured on
// the real clock, so freezing session time cannot postpone eviction.
func (s *Session) LastSeen() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSeen
}

func (s *Session) touch(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSeen = now
}

// Reset discards every challenge mutation and returns the clock to real time.
// The seed is left alone, so a suite can pick a seed once and reset between
// tests without re-picking it.
func (s *Session) Reset() {
	s.mu.Lock()
	s.data = nil
	s.mu.Unlock()
	s.Clock.Reset()
}

// Reseed changes the seed and discards challenge state derived from the old
// one, so generated content matches the new seed everywhere.
func (s *Session) Reseed(seed uint64) {
	s.mu.Lock()
	s.data = nil
	s.mu.Unlock()
	s.RNG.Reseed(seed)
}

// BaseSeed reports the seed the session started with.
func (s *Session) BaseSeed() uint64 { return s.baseSeed }

// Value returns the session-scoped state stored under key, calling create once
// if it is not there yet. Challenges use it to keep their own state type:
//
//	items := session.Value(s, "optimistic-revert", newItemList)
//
// create runs while the session is locked, so concurrent requests for the same
// key see the same value. The returned value does its own locking.
func Value[T any](s *Session, key string, create func() T) T {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data == nil {
		s.data = make(map[string]any)
	}
	if existing, ok := s.data[key]; ok {
		if typed, ok := existing.(T); ok {
			return typed
		}
	}
	created := create()
	s.data[key] = created
	return created
}

// Keys lists the challenge state keys currently populated, for state dumps.
func (s *Session) Keys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

var idEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func newID() ID {
	b := make([]byte, 10)
	if _, err := rand.Read(b); err != nil {
		panic("session: no entropy available: " + err.Error())
	}
	return ID(strings.ToLower(idEncoding.EncodeToString(b)))
}

// ValidID reports whether id is safe to use as a client-supplied session name.
// Restricting the charset keeps ids usable verbatim in cookies, headers and
// log lines.
func ValidID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}
