package challenge

import (
	"strings"
	"testing"
)

func valid() Challenge {
	return Challenge{
		ID:        "delayed-element",
		Title:     "Delayed element",
		URL:       "/app/delayed-element",
		Zone:      ZoneApp,
		Tier:      T1,
		Category:  "C. Waits & Timing",
		Summary:   "An element appears after a fixed delay.",
		WhyHard:   "A naive locate runs before the element exists.",
		Hint:      "Wait for the element rather than for a duration.",
		Tags:      []string{"waits"},
		Concepts:  []string{"explicit wait"},
		Selectors: []Selector{{TestID: "delayed-message", Note: "Appears after the delay"}},
		Stability: Stable,
	}
}

func TestValidChallengeRegisters(t *testing.T) {
	r, err := NewRegistry([]Challenge{valid()})
	if err != nil {
		t.Fatalf("valid challenge rejected: %v", err)
	}
	if r.Len() != 1 {
		t.Fatalf("registered %d challenges, want 1", r.Len())
	}
	if _, ok := r.Lookup("delayed-element"); !ok {
		t.Fatal("challenge not retrievable by id")
	}
}

func TestRegistryRejectsIncompleteChallenges(t *testing.T) {
	cases := map[string]func(*Challenge){
		"missing hint":         func(c *Challenge) { c.Hint = "" },
		"missing summary":      func(c *Challenge) { c.Summary = "" },
		"missing why hard":     func(c *Challenge) { c.WhyHard = "" },
		"missing tags":         func(c *Challenge) { c.Tags = nil },
		"missing concepts":     func(c *Challenge) { c.Concepts = nil },
		"missing selectors":    func(c *Challenge) { c.Selectors = nil },
		"unannotated selector": func(c *Challenge) { c.Selectors = []Selector{{TestID: "x"}} },
		"unknown tier":         func(c *Challenge) { c.Tier = "T9" },
		"unknown zone":         func(c *Challenge) { c.Zone = "nowhere" },
		"unknown stability":    func(c *Challenge) { c.Stability = "maybe" },
		"url outside zone":     func(c *Challenge) { c.URL = "/classic/delayed-element" },
		"url prefix collides":  func(c *Challenge) { c.URL = "/apple/delayed-element" },
		"id not kebab-case":    func(c *Challenge) { c.ID = "Delayed_Element" },
	}

	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			c := valid()
			breakIt(&c)
			if _, err := NewRegistry([]Challenge{c}); err == nil {
				t.Fatal("registry accepted an invalid challenge")
			}
		})
	}
}

func TestHostileLocatorsMayOmitSelectors(t *testing.T) {
	c := valid()
	c.Selectors = nil
	c.HostileLocators = true

	if _, err := NewRegistry([]Challenge{c}); err != nil {
		t.Fatalf("hostile-locator challenge rejected: %v", err)
	}
}

func TestRegistryRejectsDuplicateIDs(t *testing.T) {
	_, err := NewRegistry([]Challenge{valid()}, []Challenge{valid()})
	if err == nil || !strings.Contains(err.Error(), "duplicate id") {
		t.Fatalf("want a duplicate-id error, got %v", err)
	}
}

func TestChallengesAreOrderedDeterministically(t *testing.T) {
	first, second := valid(), valid()
	second.ID, second.URL = "aaa-first", "/app/aaa-first"

	r, err := NewRegistry([]Challenge{first, second})
	if err != nil {
		t.Fatalf("registry rejected fixtures: %v", err)
	}
	if got := r.All()[0].ID; got != "aaa-first" {
		t.Fatalf("first challenge is %q, want aaa-first", got)
	}
}

func TestByZoneSkipsEmptyZones(t *testing.T) {
	r, err := NewRegistry([]Challenge{valid()})
	if err != nil {
		t.Fatalf("registry rejected fixture: %v", err)
	}

	groups := r.ByZone()
	if len(groups) != 1 {
		t.Fatalf("%d zone groups, want 1", len(groups))
	}
	if groups[0].Zone.ID != ZoneApp {
		t.Fatalf("grouped under %q, want app", groups[0].Zone.ID)
	}
}

func TestManifestReportsTheCallersSeed(t *testing.T) {
	r, err := NewRegistry([]Challenge{valid()})
	if err != nil {
		t.Fatalf("registry rejected fixture: %v", err)
	}

	m := r.Manifest("0.1.0", "worker-1", 42)
	if m.Seed != 42 || m.Session != "worker-1" {
		t.Fatalf("manifest lost session context: %+v", m)
	}
	if m.Count != 1 || len(m.Challenges) != 1 {
		t.Fatalf("manifest count %d, want 1", m.Count)
	}
	if len(m.Zones) != 1 {
		t.Fatalf("manifest advertised %d zones, want only the populated one", len(m.Zones))
	}
}

func TestSelectorFamilyCoversItsMembers(t *testing.T) {
	cases := []struct {
		family, testID string
		covers         bool
	}{
		{"otp-<n>", "otp-3", true},
		{"otp-<n>", "otp-", false},
		{"otp-<n>", "otp-3x", false},
		{"locale-<s>", "locale-ar-EG", true},
		{"locale-<s>", "locale-de-DE", true},
		{"locale-<s>", "localear", false},
		{"status-link-<n>", "status-link-418", true},
		{"status-link-<n>", "status-link-teapot", false},
		{"row-<s>", "row-amount", true},
		{"row-<s>", "rows", false},
	}

	for _, c := range cases {
		// The representative is deliberately not the id under test, so a match
		// can only have come from the family.
		s := Selector{TestID: "unrelated", Family: c.family, Note: "n/a"}
		if got := s.Covers(c.testID); got != c.covers {
			t.Errorf("family %q covers %q = %v, want %v", c.family, c.testID, got, c.covers)
		}
	}
}

func TestSelectorWithoutFamilyCoversOnlyItself(t *testing.T) {
	s := Selector{TestID: "otp-0", Note: "n/a"}
	if !s.Covers("otp-0") {
		t.Error("a selector does not cover its own test id")
	}
	if s.Covers("otp-1") {
		t.Error("a selector with no family covered a neighbour, which would hide an undeclared element")
	}
}

func TestFamilyMustNameAPlaceholderAndIncludeItsOwnSelector(t *testing.T) {
	for _, family := range []string{"otp-box", "field-<n>"} {
		c := valid()
		c.Selectors = []Selector{{TestID: "otp-0", Family: family, Note: "First box"}}
		if _, err := NewRegistry([]Challenge{c}); err == nil {
			t.Errorf("registry accepted family %q on selector otp-0", family)
		}
	}

	c := valid()
	c.Selectors = []Selector{{TestID: "otp-0", Family: "otp-<n>", Note: "First box"}}
	if _, err := NewRegistry([]Challenge{c}); err != nil {
		t.Errorf("registry rejected a well-formed family: %v", err)
	}
}
