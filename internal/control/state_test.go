package control

import (
	"strings"
	"testing"

	"github.com/heyrmi/testground/internal/rng"
)

func newState(seed uint64) *State { return New(rng.New(seed)) }

func TestNothingHappensUntilAsked(t *testing.T) {
	state := newState(42)

	if ms := state.LatencyFor("/anything"); ms != 0 {
		t.Errorf("unrequested latency of %d ms", ms)
	}
	if _, _, fail := state.FailureFor("/anything"); fail {
		t.Error("unrequested failure")
	}
	if state.Flaked("anything") {
		t.Error("unrequested flake")
	}
}

func TestFailureRateReplaysForTheSameSeed(t *testing.T) {
	pattern := func() []bool {
		state := newState(42)
		state.SetFailure(FailureRule{Route: "/api/*", Status: 503, Rate: 0.5})

		out := make([]bool, 20)
		for i := range out {
			_, _, out[i] = state.FailureFor("/api/thing")
		}
		return out
	}

	first, second := pattern(), pattern()
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("request %d differed between runs: %v vs %v", i, first, second)
		}
	}

	failures := 0
	for _, failed := range first {
		if failed {
			failures++
		}
	}
	if failures == 0 || failures == len(first) {
		t.Fatalf("a rate of 0.5 produced %d failures out of %d", failures, len(first))
	}
}

func TestADifferentSeedProducesADifferentPattern(t *testing.T) {
	pattern := func(seed uint64) string {
		state := New(rng.New(seed))
		state.SetFailure(FailureRule{Route: "/api/*", Status: 500, Rate: 0.5})

		out := make([]byte, 24)
		for i := range out {
			out[i] = '.'
			if _, _, failed := state.FailureFor("/api/thing"); failed {
				out[i] = 'x'
			}
		}
		return string(out)
	}

	if pattern(1) == pattern(2) {
		t.Fatal("two seeds produced the same failure pattern")
	}
}

func TestFailFirstNThenSucceed(t *testing.T) {
	state := newState(42)
	state.SetFailure(FailureRule{Route: "/api/retry", Status: 503, Times: 3})

	for i := range 3 {
		if _, _, fail := state.FailureFor("/api/retry"); !fail {
			t.Fatalf("call %d should have failed", i+1)
		}
	}
	for i := range 3 {
		if _, _, fail := state.FailureFor("/api/retry"); fail {
			t.Fatalf("call %d should have succeeded", i+4)
		}
	}
}

func TestRulesAreIndependentOfEachOther(t *testing.T) {
	// A rule's decisions must not shift because a different rule fired
	// between them, or two challenges could not be driven at once.
	alone := newState(42)
	alone.SetFailure(FailureRule{Route: "/a", Status: 500, Rate: 0.5})

	together := newState(42)
	together.SetFailure(FailureRule{Route: "/a", Status: 500, Rate: 0.5})
	together.SetFailure(FailureRule{Route: "/b", Status: 500, Rate: 0.5})

	for i := range 10 {
		_, _, want := alone.FailureFor("/a")
		together.FailureFor("/b")
		_, _, got := together.FailureFor("/a")
		if got != want {
			t.Fatalf("call %d: rule /a shifted because /b fired", i)
		}
	}
}

func TestRouteMatching(t *testing.T) {
	cases := []struct {
		pattern, path string
		want          bool
	}{
		{"/api/thing", "/api/thing", true},
		{"/api/thing", "/api/thing/more", false},
		{"/api/*", "/api/thing/more", true},
		{"/api/*", "/other", false},
		{"*", "/anything", true},
		{"", "/anything", false},
	}

	for _, c := range cases {
		if got := matches(c.pattern, c.path); got != c.want {
			t.Errorf("matches(%q, %q) = %v, want %v", c.pattern, c.path, got, c.want)
		}
	}
}

func TestLatencyJitterIsBoundedAndReplayable(t *testing.T) {
	draw := func() []int {
		state := newState(42)
		state.SetLatency(LatencyRule{Route: "/slow", Ms: 100, Jitter: 50})

		out := make([]int, 12)
		for i := range out {
			out[i] = state.LatencyFor("/slow")
		}
		return out
	}

	first, second := draw(), draw()
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("jitter differed between runs at %d: %v vs %v", i, first, second)
		}
		if first[i] < 100 || first[i] >= 150 {
			t.Fatalf("jitter escaped its bounds: %d", first[i])
		}
	}
}

func TestSettingARuleTwiceReplacesIt(t *testing.T) {
	state := newState(42)
	state.SetFailure(FailureRule{Route: "/api/*", Status: 500, Times: 1})
	state.SetFailure(FailureRule{Route: "/api/*", Status: 429, Times: 1})

	if got := len(state.Snapshot().Failures); got != 1 {
		t.Fatalf("%d rules for one route, want 1", got)
	}
	status, _, fail := state.FailureFor("/api/thing")
	if !fail || status != 429 {
		t.Fatalf("got status %d fail=%v, want 429 true", status, fail)
	}
}

func TestClearingRules(t *testing.T) {
	state := newState(42)
	state.SetLatency(LatencyRule{Route: "/slow", Ms: 500})
	state.SetFailure(FailureRule{Route: "/bad", Status: 500})
	state.SetFlake(FlakeRule{Challenge: "toast", Probability: 0.5})
	state.SetFeature("beta", true)

	state.SetLatency(LatencyRule{Route: "/slow", Ms: 0})
	state.SetFailure(FailureRule{Route: "/bad", Status: 0})
	state.SetFlake(FlakeRule{Challenge: "toast", Probability: 0})

	snapshot := state.Snapshot()
	if len(snapshot.Latency) != 0 || len(snapshot.Failures) != 0 || len(snapshot.Flakes) != 0 {
		t.Fatalf("rules survived removal: %+v", snapshot)
	}
	if !state.Feature("beta") {
		t.Error("clearing rules should not clear feature flags")
	}

	state.Reset()
	if state.Feature("beta") {
		t.Error("reset should clear feature flags")
	}
}

func TestFlakeProbabilityReplays(t *testing.T) {
	pattern := func() string {
		state := newState(7)
		state.SetFlake(FlakeRule{Challenge: "toast", Probability: 0.3})

		out := make([]byte, 30)
		for i := range out {
			out[i] = '.'
			if state.Flaked("toast") {
				out[i] = 'x'
			}
		}
		return string(out)
	}

	first, second := pattern(), pattern()
	if first != second {
		t.Fatalf("flake pattern differed between runs:\n %s\n %s", first, second)
	}
	if !strings.Contains(first, "x") {
		t.Fatalf("a probability of 0.3 never fired: %s", first)
	}
}
