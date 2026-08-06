// Package challenge describes what the playground offers.
//
// Every challenge is declared as data, not as a handler that happens to render
// a page. The manifest at /api/challenges is generated from these declarations,
// each page renders its own description and hint panel from them, and the
// registry refuses to start if a challenge is missing anything the project
// promises to ship with every page.
package challenge

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
