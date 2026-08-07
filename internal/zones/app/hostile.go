package app

import (
	"net/http"
	"sync"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/control"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

const (
	hostileStateKey = "hostile-locators"
	// hostileRebuildFlag makes the class names churn on their own. The button
	// on the page ships a build when someone asks for one; this ships one when
	// nobody did, which is the version of the problem that actually reaches
	// people -- a selector that stops matching between a run and its rerun with
	// no code change anyone will connect to it.
	hostileRebuildFlag = "hostile-locators.rebuild"
)

func hostileLocators() challenge.Challenge {
	return challenge.Challenge{
		ID:       "hostile-locators",
		Title:    "A page that gives you nothing to hold on to",
		URL:      "/app/hostile-locators",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T4,
		Category: "S. Hostile Locators",
		Summary: "Generated class names that change on every build, two elements sharing one " +
			"id, twelve levels of div with no semantics at the bottom, text split across " +
			"nodes, invisible characters, CSS truncation, and a pair of buttons identical " +
			"in every respect but position.",
		WhyHard: "Everything here is legal, everything here ships, and none of it is deliberate " +
			"sabotage on anyone's part -- which is why it is worth practising against. The " +
			"class names are content hashes, so a selector written against one is correct " +
			"until the next deploy and then fails with no code change anyone will connect " +
			"to it. The duplicate id is invalid HTML that browsers accept silently, so one " +
			"lookup finds the first and a CSS selector finds both. The split text reads as " +
			"one sentence and is three nodes, so an exact-match assertion fails against a " +
			"string the user can plainly see. The truncated line shows an ellipsis while " +
			"the DOM holds the whole thing, so what a person can read and what a test can " +
			"read have quietly diverged.",
		Hint: "This page is a diagnosis, not a puzzle. Every locator that survives it is one " +
			"anchored to something the application means rather than something it renders: " +
			"a test id, a role, an accessible name. Where none of those exist -- the div " +
			"soup, the twins -- the honest answer is that the markup needs fixing, and a " +
			"positional selector is a note to come back rather than a solution. For the " +
			"split and zero-width text, prefer a contains match over an exact one and " +
			"normalise before comparing.",
		Tags:     []string{"locators", "hostile", "css-in-js", "duplicate-ids", "text"},
		Concepts: []string{"generated class names are not selectors", "duplicate ids are legal and silent", "visible text is not DOM text", "positional selectors are a diagnosis"},
		Selectors: []challenge.Selector{
			{TestID: "rebuild", Role: "button", Note: "Regenerates every class name, as a deploy would"},
			{TestID: "build-number", Note: "Which build the class names belong to"},
			{TestID: "sample-class", Note: "One current class name, so the churn is observable"},
			{TestID: "chosen", Note: "Which element was last activated; the only stable target here"},
			{TestID: "split-text", Note: "Reads as one sentence, is three nodes"},
			{TestID: "zero-width", Note: "Contains zero-width spaces between the words"},
			{TestID: "truncated", Note: "Shows an ellipsis; the DOM holds the whole string"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/app/hostile-locators/build", Note: "The build the class names are derived from; ships a new one on every read while the flag is on"},
		},
		Controls: []challenge.Control{
			{
				Name:    hostileRebuildFlag,
				Kind:    "control-plane",
				Default: "off",
				Note: "POST /api/control/feature {\"flag\":\"hostile-locators.rebuild\",\"enabled\":true} " +
					"ships a new build on every read of the build endpoint, so every generated " +
					"class name changes without anyone pressing anything.",
			},
		},
		HostileLocators: true,
		Stability:       challenge.Stable,
	}
}

// hostileBuild is the build the generated class names are derived from. It sits
// on the session, so one worker shipping a new build never renames the elements
// another worker is halfway through locating.
type hostileBuild struct {
	mu    sync.Mutex
	build int
}

func hostileBuildFor(sess *session.Session) *hostileBuild {
	return session.Value(sess, hostileStateKey, func() *hostileBuild {
		// One, not zero: the page has always called its first build 1, and the
		// class names are derived from the number.
		return &hostileBuild{build: 1}
	})
}

// read reports the current build, shipping a new one first when asked. With the
// flag off the number never moves, so a selector written against a class name
// still matches on the next request -- which is what lets the page demonstrate
// that such a selector works before demonstrating that it breaks.
func (b *hostileBuild) read(ship bool) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ship {
		b.build++
	}
	return b.build
}

type hostileBuildResponse struct {
	Build int    `json:"build"`
	Flag  string `json:"flag"`
	// Churn says whether the next read will move the number, so a test can tell
	// a stable build from one that is about to be replaced underneath it.
	Churn bool `json:"churn"`
}

func handleHostileBuild(w http.ResponseWriter, r *http.Request) {
	sess := session.MustFromContext(r.Context())
	churn := control.For(sess).Feature(hostileRebuildFlag)

	httpx.JSON(w, http.StatusOK, hostileBuildResponse{
		Build: hostileBuildFor(sess).read(churn),
		Flag:  hostileRebuildFlag,
		Churn: churn,
	})
}
