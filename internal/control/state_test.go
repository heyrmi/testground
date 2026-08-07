package control

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/heyrmi/testground/internal/rng"
	"github.com/heyrmi/testground/internal/session"
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

// A rule whose Times is spent must not stop a broader rule from firing. The
// first version returned on the first matching rule, so an exhausted narrow
// rule silently disabled every rule after it.
func TestASpentRuleDoesNotShadowLaterOnes(t *testing.T) {
	state := newState(42)
	state.SetFailure(FailureRule{Route: "/classic/*", Status: 503, Times: 1})
	state.SetFailure(FailureRule{Route: "*", Status: 500})

	if status, _, fail := state.FailureFor("/classic/page"); !fail || status != 503 {
		t.Fatalf("first call: status %d fail %v, want 503 true", status, fail)
	}
	if status, _, fail := state.FailureFor("/classic/page"); !fail || status != 500 {
		t.Fatalf("second call should fall through to the catch-all: status %d fail %v", status, fail)
	}
}

// wiredFlakes are the challenges whose handlers ask Flaked what to do, and the
// misbehaviour each one produces. It is the same list docs/control-plane.md
// publishes; keeping it here as well means a caller added without being
// documented shows up as a diff in two places rather than none.
var wiredFlakes = []struct{ challenge, misbehaviour string }{
	{"optimistic-revert", "a toggle the server would have accepted is refused, so the row reverts"},
	{"retries", "the endpoint keeps refusing after its failFirst budget is spent"},
	{"data-table", "the rows come back reversed while the response still reports the sort asked for"},
	{"request-races", "the delay a request asked for is dropped, so the two searches land in another order"},
}

// wiredFeatures are the flags a challenge handler reads, and what each does.
var wiredFeatures = []struct{ flag, effect string }{
	{"visual-regression.diff", "the swatch is one pixel wider, the same difference ?diff=1 makes"},
	{"hostile-locators.rebuild", "every read of the build endpoint ships a new build, renaming every generated class"},
}

// The guarantee the whole design rests on: a page that nobody has asked to
// misbehave behaves exactly as it documents. Every one of these handlers calls
// Flaked on every request, so if an unset rule ever fired the default
// playground would be the flaky one.
func TestWiredChallengesAreQuietUntilAsked(t *testing.T) {
	state := newState(42)

	for _, wired := range wiredFlakes {
		for i := range 50 {
			if state.Flaked(wired.challenge) {
				t.Fatalf("%s flaked on call %d with no rule set: %s",
					wired.challenge, i+1, wired.misbehaviour)
			}
		}
	}
	for _, wired := range wiredFeatures {
		if state.Feature(wired.flag) {
			t.Errorf("%s was on before anyone set it: %s", wired.flag, wired.effect)
		}
	}
}

func TestWiredChallengesMisbehaveWhenAsked(t *testing.T) {
	state := newState(42)

	for _, wired := range wiredFlakes {
		state.SetFlake(FlakeRule{Challenge: wired.challenge, Probability: 1})
	}
	for _, wired := range wiredFlakes {
		if !state.Flaked(wired.challenge) {
			t.Errorf("%s did not flake at a probability of 1", wired.challenge)
		}
	}

	for _, wired := range wiredFeatures {
		state.SetFeature(wired.flag, true)
		if !state.Feature(wired.flag) {
			t.Errorf("%s did not turn on", wired.flag)
		}
		state.SetFeature(wired.flag, false)
		if state.Feature(wired.flag) {
			t.Errorf("%s did not turn off again", wired.flag)
		}
	}
}

// A handler asks Flaked on every request, including every request made before
// anyone wanted chaos. Those calls must not consume the seeded stream, or which
// requests fail would depend on how long the page had been used first -- and a
// rule that cannot be replayed is worth nothing.
func TestAnUnsetFlakeRuleDrawsNothing(t *testing.T) {
	armedFirst := newState(42)
	armedFirst.SetFlake(FlakeRule{Challenge: "data-table", Probability: 0.5})

	usedFirst := newState(42)
	for range 25 {
		if usedFirst.Flaked("data-table") {
			t.Fatal("flaked with no rule set")
		}
	}
	usedFirst.SetFlake(FlakeRule{Challenge: "data-table", Probability: 0.5})

	for i := range 20 {
		if want, got := armedFirst.Flaked("data-table"), usedFirst.Flaked("data-table"); got != want {
			t.Fatalf("draw %d differed: the calls made before the rule existed moved the stream", i)
		}
	}
}

// Four challenges can now be driven chaotically at once, which only stays
// debuggable if each one's sequence is its own.
func TestFlakeRulesDoNotInterfereWithEachOther(t *testing.T) {
	alone := newState(42)
	alone.SetFlake(FlakeRule{Challenge: "retries", Probability: 0.5})

	crowded := newState(42)
	for _, wired := range wiredFlakes {
		crowded.SetFlake(FlakeRule{Challenge: wired.challenge, Probability: 0.5})
	}

	for i := range 20 {
		want := alone.Flaked("retries")
		for _, wired := range wiredFlakes {
			if wired.challenge != "retries" {
				crowded.Flaked(wired.challenge)
			}
		}
		if got := crowded.Flaked("retries"); got != want {
			t.Fatalf("call %d: retries shifted because the other rules fired", i)
		}
	}
}

// The handler-facing form. A refusal a page produces by design and one a flake
// rule produced are the same status with the same body, so the header is the
// only thing that tells a tester which of the two they have just proved.
func TestFlakedMarksTheResponseAndOnlyWhenItFired(t *testing.T) {
	sess := session.NewStore(session.Options{Seed: 42}).Open("flake-header")

	call := func() (bool, string) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/app/retries/data", nil).
			WithContext(session.NewContext(context.Background(), sess))
		return Flaked(w, r, "retries"), w.Header().Get(HeaderFlaked)
	}

	if flaked, header := call(); flaked || header != "" {
		t.Fatalf("no rule set, but flaked=%v and the response carried %q", flaked, header)
	}

	For(sess).SetFlake(FlakeRule{Challenge: "retries", Probability: 1})
	if flaked, header := call(); !flaked || header != "retries" {
		t.Fatalf("with a rule set, flaked=%v and the response carried %q", flaked, header)
	}
}

// A handler reached without the session middleware -- a unit test, a route
// mounted outside it -- serves its default rather than panicking. Chaos is the
// one thing that must never happen by accident.
func TestFlakedWithoutASessionIsFalse(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/app/retries/data", nil)

	if Flaked(w, r, "retries") {
		t.Error("flaked on a request carrying no session")
	}
	if header := w.Header().Get(HeaderFlaked); header != "" {
		t.Errorf("marked the response %q anyway", header)
	}
}

// Latency is the opposite: a request has one delay, so the first matching
// rule wins outright rather than composing with the ones after it.
func TestLatencyTakesTheFirstMatchOnly(t *testing.T) {
	state := newState(42)
	state.SetLatency(LatencyRule{Route: "/slow/*", Ms: 100})
	state.SetLatency(LatencyRule{Route: "*", Ms: 900})

	if got := state.LatencyFor("/slow/thing"); got != 100 {
		t.Fatalf("latency %d ms, want 100 -- delays must not accumulate", got)
	}
	if got := state.LatencyFor("/other"); got != 900 {
		t.Fatalf("latency %d ms, want 900", got)
	}
}
