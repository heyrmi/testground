package challenge

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Registry is the immutable set of challenges the running binary serves. It is
// assembled once at startup and never mutated, so it needs no locking and
// carries no per-client state.
type Registry struct {
	ordered []Challenge
	byID    map[string]Challenge
}

var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// NewRegistry validates and freezes the given challenge sets. It fails rather
// than serving a challenge that breaks the project's own rules: every page
// ships a description, a hint, a stable URL under its zone, and either test
// ids or an explicit declaration that withholding them is the point.
func NewRegistry(sets ...[]Challenge) (*Registry, error) {
	var all []Challenge
	for _, set := range sets {
		all = append(all, set...)
	}

	byID := make(map[string]Challenge, len(all))
	var problems []error
	for _, c := range all {
		if _, clash := byID[c.ID]; clash {
			problems = append(problems, fmt.Errorf("challenge %q: duplicate id", c.ID))
			continue
		}
		if err := validate(c); err != nil {
			problems = append(problems, err)
			continue
		}
		byID[c.ID] = c
	}
	if len(problems) > 0 {
		return nil, errors.Join(problems...)
	}

	slices.SortFunc(all, func(a, b Challenge) int { return strings.Compare(a.ID, b.ID) })
	return &Registry{ordered: all, byID: byID}, nil
}

// MustRegistry is NewRegistry for callers whose challenge set is compiled in,
// where a validation failure is a build-time mistake rather than a runtime one.
func MustRegistry(sets ...[]Challenge) *Registry {
	r, err := NewRegistry(sets...)
	if err != nil {
		panic("challenge: invalid registry: " + err.Error())
	}
	return r
}

// All returns every challenge, ordered by id. It is never nil, so the manifest
// always encodes an array rather than a null.
func (r *Registry) All() []Challenge {
	out := make([]Challenge, len(r.ordered))
	copy(out, r.ordered)
	return out
}

// Len reports how many challenges are registered.
func (r *Registry) Len() int { return len(r.ordered) }

// Lookup returns the challenge with the given id.
func (r *Registry) Lookup(id string) (Challenge, bool) {
	c, ok := r.byID[id]
	return c, ok
}

// ZoneGroup is one zone with the challenges that live in it.
type ZoneGroup struct {
	Zone       ZoneInfo    `json:"zone"`
	Challenges []Challenge `json:"challenges"`
}

// ByZone groups challenges for display, in zone order, skipping empty zones.
func (r *Registry) ByZone() []ZoneGroup {
	var groups []ZoneGroup
	for _, zone := range zones {
		var members []Challenge
		for _, c := range r.ordered {
			if c.Zone == zone.ID {
				members = append(members, c)
			}
		}
		if len(members) > 0 {
			groups = append(groups, ZoneGroup{Zone: zone, Challenges: members})
		}
	}
	return groups
}

func validate(c Challenge) error {
	var problems []error
	fail := func(format string, args ...any) {
		problems = append(problems, fmt.Errorf("challenge %q: "+format, append([]any{c.ID}, args...)...))
	}

	if !idPattern.MatchString(c.ID) {
		fail("id must be kebab-case")
	}
	required := []struct{ field, value string }{
		{"title", c.Title},
		{"category", c.Category},
		{"summary", c.Summary},
		{"whyHard", c.WhyHard},
		{"hint", c.Hint},
	}
	for _, r := range required {
		if strings.TrimSpace(r.value) == "" {
			fail("%s is required on every page", r.field)
		}
	}

	switch c.Tier {
	case T1, T2, T3, T4:
	default:
		fail("unknown tier %q", c.Tier)
	}
	switch c.Stability {
	case Stable, Experimental:
	default:
		fail("unknown stability %q", c.Stability)
	}

	zone, known := LookupZone(c.Zone)
	switch {
	case !known:
		fail("unknown zone %q", c.Zone)
	case c.URL != zone.Prefix && !strings.HasPrefix(c.URL, zone.Prefix+"/"):
		fail("url %q is outside zone prefix %q", c.URL, zone.Prefix)
	}

	if len(c.Tags) == 0 {
		fail("at least one tag is required")
	}
	if len(c.Concepts) == 0 {
		fail("at least one concept is required")
	}
	if len(c.Selectors) == 0 && !c.HostileLocators {
		fail("no selectors declared; set hostileLocators if that is the exercise")
	}
	for _, s := range c.Selectors {
		if strings.TrimSpace(s.TestID) == "" {
			fail("selector with an empty test id")
		}
		if strings.TrimSpace(s.Note) == "" {
			fail("selector %q needs a note saying what it is", s.TestID)
		}
		if s.Family == "" {
			continue
		}
		// A family that names no placeholder matches one id, so it is a second
		// spelling of TestID rather than a family, and a representative that is
		// not itself a member means the pattern describes something other than
		// what was declared. Either way the contract check would quietly stop
		// covering the members it was written for.
		pattern, ok := FamilyPattern(s.Family)
		switch {
		case !ok:
			fail("selector %q has family %q with no <n> or <s> placeholder", s.TestID, s.Family)
		case !pattern.MatchString(s.TestID):
			fail("selector %q is not itself a member of its family %q", s.TestID, s.Family)
		}
	}

	return errors.Join(problems...)
}
