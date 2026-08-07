// Package challenge describes what the playground offers.
//
// Every challenge is declared as data, not as a handler that happens to render
// a page. The manifest at /api/challenges is generated from these declarations,
// each page renders its own description and hint panel from them, and the
// registry refuses to start if a challenge is missing anything the project
// promises to ship with every page.
package challenge

import (
	"regexp"
	"strings"
)

// Tier grades how hard a challenge is to automate.
type Tier string

const (
	T1 Tier = "T1" // intro
	T2 Tier = "T2" // intermediate
	T3 Tier = "T3" // hard
	T4 Tier = "T4" // hostile: deliberately near-unautomatable, a teaching case
)

// Stability states whether the DOM contract is frozen. Stable pages never
// change behaviour again; only bugs are fixed.
type Stability string

const (
	Stable       Stability = "stable"
	Experimental Stability = "experimental"
)

// Zone identifies one of the coexisting frontends.
type Zone string

const (
	ZoneClassic    Zone = "classic"
	ZoneLegacy     Zone = "legacy"
	ZoneHypermedia Zone = "hx"
	ZoneApp        Zone = "app"
	ZoneComponents Zone = "wc"
	ZoneRealtime   Zone = "live"
)

// ZoneInfo describes a zone for the manifest and the index page.
type ZoneInfo struct {
	ID         Zone   `json:"id"`
	Title      string `json:"title"`
	Prefix     string `json:"prefix"`
	Technology string `json:"technology"`
	Tests      string `json:"tests"`
}

// zones is declared in presentation order, oldest technique first.
var zones = []ZoneInfo{
	{ZoneClassic, "Classic", "/classic", "Go html/template, no JavaScript",
		"Full page loads, form posts, redirect chains, no-JS fallbacks"},
	{ZoneLegacy, "Legacy", "/legacy", "jQuery 3 and Bootstrap 3",
		"jQuery widgets, $.ajax partial updates, old datepickers and table sorters"},
	{ZoneHypermedia, "Hypermedia", "/hx", "htmx 2 and Alpine.js over Go templates",
		"Partial DOM swaps, out-of-band updates, element replacement mid-interaction"},
	{ZoneApp, "Modern SPA", "/app", "React 19, TypeScript, Tailwind",
		"Client routing, optimistic updates, portals, virtualisation, controlled inputs"},
	{ZoneComponents, "Components", "/wc", "Lit web components",
		"Open and closed shadow DOM, nesting, slots, events crossing the boundary"},
	{ZoneRealtime, "Realtime", "/live", "Vanilla TypeScript over WebSocket and SSE",
		"Live regions, streaming updates, reconnect behaviour, race conditions"},
}

// Zones returns every zone in presentation order.
func Zones() []ZoneInfo { return append([]ZoneInfo(nil), zones...) }

// LookupZone returns the description of a zone.
func LookupZone(id Zone) (ZoneInfo, bool) {
	for _, z := range zones {
		if z.ID == id {
			return z, true
		}
	}
	return ZoneInfo{}, false
}

// Selector points at an element worth locating, so a manifest consumer can
// generate page objects without scraping the page.
type Selector struct {
	TestID string `json:"testId"`
	Role   string `json:"role,omitempty"`
	Note   string `json:"note"`
	// Transient marks an element that exists only during an interaction, so a
	// contract check knows not to expect it in the page as first served. Every
	// other declared selector must be present on load, and the reference suite
	// asserts exactly that against the live DOM.
	Transient bool `json:"transient,omitempty"`
	// Frame is the path of iframe test ids to descend before looking, for an
	// element that lives in a nested browsing context rather than in the top
	// document. A locator that does not enter these frames will never find it.
	Frame []string `json:"frame,omitempty"`
	// Family names a repeated set this selector is one representative of, as a
	// pattern: <n> stands for a run of digits and <s> for a run of word
	// characters, so "otp-<n>" covers otp-0 through otp-5.
	//
	// A page that renders six identical boxes is worth declaring once, not six
	// times, and listing every index would turn the manifest from the elements
	// worth locating into an inventory of the DOM. Naming the shape instead
	// keeps the declaration short and still says exactly what exists, which is
	// what lets a generated page object offer an indexed accessor and a
	// contract check tell a member of a known family from an element nobody
	// declared.
	Family string `json:"family,omitempty"`
}

// familyPlaceholders expands a Family pattern. Everything outside a placeholder
// is matched literally, so a pattern cannot accidentally become a wildcard
// through a character that happens to mean something to a regexp.
var familyPlaceholders = map[string]string{
	"<n>": `[0-9]+`,
	"<s>": `[A-Za-z0-9_-]+`,
}

// FamilyPattern compiles a Family into an anchored expression. It reports
// whether the pattern named any placeholder at all: one that names none matches
// only itself, which means the author wrote a second spelling of TestID rather
// than a family.
func FamilyPattern(family string) (*regexp.Regexp, bool) {
	var (
		expanded strings.Builder
		found    bool
		rest     = family
	)
	for rest != "" {
		next, name := nextPlaceholder(rest)
		expanded.WriteString(regexp.QuoteMeta(rest[:next]))
		if name == "" {
			break
		}
		expanded.WriteString(familyPlaceholders[name])
		found = true
		rest = rest[next+len(name):]
	}

	pattern, err := regexp.Compile(`\A` + expanded.String() + `\z`)
	if err != nil {
		return nil, false
	}
	return pattern, found
}

// nextPlaceholder finds the first placeholder in s, returning where it starts
// and which one it is. An empty name means there are none left.
func nextPlaceholder(s string) (at int, name string) {
	at, name = len(s), ""
	for placeholder := range familyPlaceholders {
		if i := strings.Index(s, placeholder); i >= 0 && i < at {
			at, name = i, placeholder
		}
	}
	return at, name
}

// Covers reports whether a rendered test id is this selector, or a member of
// the family it represents.
func (s Selector) Covers(testID string) bool {
	if s.TestID == testID {
		return true
	}
	if s.Family == "" {
		return false
	}
	pattern, ok := FamilyPattern(s.Family)
	return ok && pattern.MatchString(testID)
}

// Endpoint is an HTTP route the challenge's page talks to. Listing them lets a
// test drive the challenge without a browser.
type Endpoint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Note   string `json:"note"`
}

// Control is a knob that makes the challenge behave predictably, or
// deliberately badly, on request.
type Control struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"` // query, control-plane
	Default string `json:"default"`
	Note    string `json:"note"`
}

// Challenge is one practice page and everything a tester needs to know
// about it without opening it.
type Challenge struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Zone     Zone   `json:"zone"`
	Tier     Tier   `json:"tier"`
	Category string `json:"category"`

	// Summary says what the page does, WhyHard says what breaks naive
	// automation, and Hint gives the intended approach as a concept rather
	// than as framework-specific code.
	Summary string `json:"summary"`
	WhyHard string `json:"whyHard"`
	Hint    string `json:"hint"`

	Tags      []string   `json:"tags"`
	Concepts  []string   `json:"concepts"`
	Selectors []Selector `json:"selectors"`
	Endpoints []Endpoint `json:"endpoints,omitempty"`
	Controls  []Control  `json:"controls,omitempty"`

	Stability Stability `json:"stability"`

	// HostileLocators marks a page that withholds test ids on purpose,
	// because locating the elements is the exercise.
	HostileLocators bool `json:"hostileLocators,omitempty"`
}
