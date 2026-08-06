// Package rng is the single source of randomness in the playground.
//
// Randomness is drawn from named streams rather than one shared generator.
// A shared generator would make every value depend on how many values other
// challenges happened to draw first, so identical seeds would still produce
// different pages under concurrent traffic. Named streams are derived from the
// seed alone, so a challenge always sees the same sequence regardless of what
// else the server is doing.
package rng

import (
	"hash/fnv"
	"math/rand/v2"
	"sync/atomic"
)

// DefaultSeed is used when no seed is supplied on the command line.
const DefaultSeed uint64 = 42

// Source hands out deterministic named streams for one seed.
type Source struct {
	seed atomic.Uint64
}

// New returns a Source rooted at seed.
func New(seed uint64) *Source {
	s := &Source{}
	s.seed.Store(seed)
	return s
}

// Seed reports the seed currently in effect.
func (s *Source) Seed() uint64 { return s.seed.Load() }

// Reseed changes the seed. Streams created afterwards use the new seed;
// streams already handed out keep running on the old one.
func (s *Source) Reseed(seed uint64) { s.seed.Store(seed) }

// Stream returns a generator whose sequence is fixed by the seed and the name.
// Two calls with the same seed and name always produce the same sequence, on
// any machine and any build.
func (s *Source) Stream(name string) *rand.Rand {
	seed := s.Seed()
	return rand.New(rand.NewPCG(seed, mix(seed, name)))
}

func mix(seed uint64, name string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(name))
	// Golden-ratio constant spreads adjacent seeds across the state space so
	// --seed 1 and --seed 2 do not produce visibly related streams.
	return h.Sum64() ^ (seed * 0x9e3779b97f4a7c15)
}
