// Package control is the playground's only source of nondeterminism.
//
// Nothing here does anything until it is asked to. A page is never randomly
// slow, never randomly broken and never randomly flaky unless a rule was set
// saying so, which is what makes the default behaviour worth trusting.
//
// When chaos is requested it is still reproducible. Every draw comes from the
// session's seeded stream indexed by how many times that rule has fired, so
// the same seed and the same sequence of requests produce the same failures on
// every run and every machine. "Flaky on purpose" and "flaky by accident" are
// different things, and only the first is useful.
package control

import (
	"strings"
	"sync"

	"github.com/heyrmi/testground/internal/rng"
)

// Key is the session state key the control surface is stored under.
const Key = "control-plane"

// LatencyRule delays every matching request.
type LatencyRule struct {
	Route string `json:"route"`
	Ms    int    `json:"ms"`
	// Jitter adds up to this many milliseconds, drawn from the session seed,
	// so a bounded random delay still replays identically.
	Jitter int `json:"jitter,omitempty"`
}

// FailureRule answers matching requests with an error instead of serving them.
type FailureRule struct {
	Route  string `json:"route"`
	Status int    `json:"status"`
	// Rate is the share of matching requests to fail, from 0 to 1. Zero with
	// no Times means every one.
	Rate float64 `json:"rate,omitempty"`
	// Times fails the first N matching requests and then stops, which is how
	// retry-and-succeed is exercised. It takes precedence over Rate.
	Times   int    `json:"times,omitempty"`
	Message string `json:"message,omitempty"`
	// Fired counts how many requests this rule has actually failed.
	Fired int `json:"fired"`
}

// FlakeRule makes a named challenge misbehave a share of the time.
type FlakeRule struct {
	Challenge   string  `json:"challenge"`
	Probability float64 `json:"probability"`
}

// State is one session's control surface. Safe for concurrent use.
type State struct {
	source *rng.Source

	mu       sync.Mutex
	latency  []LatencyRule
	failures []FailureRule
	flakes   map[string]FlakeRule
	features map[string]bool
	// counters index the seeded draws, so the nth decision for a rule always
	// reads the same value however the requests interleaved.
	counters map[string]int
}

// New returns an empty control surface drawing from source.
func New(source *rng.Source) *State {
	return &State{
		source:   source,
		flakes:   make(map[string]FlakeRule),
		features: make(map[string]bool),
		counters: make(map[string]int),
	}
}

// Reset clears every rule. The clock and the seed are the session's to reset.
func (s *State) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.latency = nil
	s.failures = nil
	s.flakes = make(map[string]FlakeRule)
	s.features = make(map[string]bool)
	s.counters = make(map[string]int)
}

// SetLatency installs or replaces the rule for a route. Ms of zero removes it.
func (s *State) SetLatency(rule LatencyRule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.latency = replace(s.latency, rule, func(r LatencyRule) string { return r.Route })
	if rule.Ms <= 0 && rule.Jitter <= 0 {
		s.latency = remove(s.latency, rule.Route, func(r LatencyRule) string { return r.Route })
	}
	delete(s.counters, "latency:"+rule.Route)
}

// SetFailure installs or replaces the rule for a route. A status below 400
// removes it.
func (s *State) SetFailure(rule FailureRule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule.Fired = 0
	s.failures = replace(s.failures, rule, func(r FailureRule) string { return r.Route })
	if rule.Status < 400 {
		s.failures = remove(s.failures, rule.Route, func(r FailureRule) string { return r.Route })
	}
	delete(s.counters, "failure:"+rule.Route)
}

// SetFlake installs a flake probability for a challenge. Zero removes it.
func (s *State) SetFlake(rule FlakeRule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rule.Probability <= 0 {
		delete(s.flakes, rule.Challenge)
		delete(s.counters, "flake:"+rule.Challenge)
		return
	}
	s.flakes[rule.Challenge] = rule
}

// SetFeature turns a named flag on or off.
func (s *State) SetFeature(flag string, enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.features[flag] = enabled
}

// Feature reports a flag's value; unknown flags are off.
func (s *State) Feature(flag string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.features[flag]
}

// LatencyFor reports how long a request for path should be delayed.
func (s *State) LatencyFor(path string) (ms int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, rule := range s.latency {
		if !matches(rule.Route, path) {
			continue
		}
		ms = rule.Ms
		if rule.Jitter > 0 {
			ms += s.drawIntLocked("latency:"+rule.Route, rule.Jitter)
		}
		return ms
	}
	return 0
}

// FailureFor reports whether a request for path should be refused, and with
// what. The decision is recorded, so Times counts down and Rate replays.
//
// Rules are considered in the order they were added and the first one that
// decides to fail wins. A rule that declines -- because its Times is spent or
// its Rate did not come up -- does not shadow the rules after it, which is
// what stops an exhausted narrow rule from silently disabling a broader one.
func (s *State) FailureFor(path string) (status int, message string, fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.failures {
		rule := &s.failures[i]
		if !matches(rule.Route, path) {
			continue
		}

		switch {
		case rule.Times > 0:
			if rule.Fired >= rule.Times {
				continue
			}
		case rule.Rate > 0:
			if s.drawFloatLocked("failure:"+rule.Route) >= rule.Rate {
				continue
			}
		}

		rule.Fired++
		return rule.Status, rule.Message, true
	}
	return 0, "", false
}

// Flaked reports whether a challenge should misbehave on this occasion.
func (s *State) Flaked(challenge string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule, ok := s.flakes[challenge]
	if !ok {
		return false
	}
	return s.drawFloatLocked("flake:"+challenge) < rule.Probability
}

// Snapshot describes the control surface for the state dump.
type Snapshot struct {
	Latency  []LatencyRule   `json:"latency"`
	Failures []FailureRule   `json:"failures"`
	Flakes   []FlakeRule     `json:"flakes"`
	Features map[string]bool `json:"features"`
}

// Snapshot copies the current rules for reporting.
func (s *State) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := Snapshot{
		Latency:  append([]LatencyRule(nil), s.latency...),
		Failures: append([]FailureRule(nil), s.failures...),
		Flakes:   make([]FlakeRule, 0, len(s.flakes)),
		Features: make(map[string]bool, len(s.features)),
	}
	if out.Latency == nil {
		out.Latency = []LatencyRule{}
	}
	if out.Failures == nil {
		out.Failures = []FailureRule{}
	}
	for _, rule := range s.flakes {
		out.Flakes = append(out.Flakes, rule)
	}
	for flag, enabled := range s.features {
		out.Features[flag] = enabled
	}
	return out
}

// drawFloatLocked returns the nth value of the named stream. Indexing by the
// call count rather than holding a generator means a rule's decisions are the
// same however other rules interleaved with it.
func (s *State) drawFloatLocked(name string) float64 {
	stream := s.source.Stream(name)
	for range s.counters[name] {
		stream.Float64()
	}
	s.counters[name]++
	return stream.Float64()
}

func (s *State) drawIntLocked(name string, n int) int {
	if n <= 0 {
		return 0
	}
	stream := s.source.Stream(name)
	for range s.counters[name] {
		stream.IntN(n)
	}
	s.counters[name]++
	return stream.IntN(n)
}

// matches reports whether a route pattern covers a path. A trailing * is a
// prefix match and everything else is exact, which is enough to be useful and
// simple enough to predict without reading the code.
func matches(pattern, path string) bool {
	if pattern == "" {
		return false
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == path
}

func replace[T any](rules []T, rule T, key func(T) string) []T {
	for i := range rules {
		if key(rules[i]) == key(rule) {
			rules[i] = rule
			return rules
		}
	}
	return append(rules, rule)
}

func remove[T any](rules []T, route string, key func(T) string) []T {
	out := rules[:0]
	for _, rule := range rules {
		if key(rule) != route {
			out = append(out, rule)
		}
	}
	return out
}
