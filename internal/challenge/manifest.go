package challenge

// Manifest is the machine-readable description of the running playground,
// served at /api/challenges. Documentation, coverage tooling and page-object
// generators read this instead of scraping the site.
type Manifest struct {
	Version string `json:"version"`
	// Session and Seed describe the caller's own isolated playground, so a
	// manifest fetched by one worker states the seed that worker's pages are
	// generated from.
	Session    string      `json:"session"`
	Seed       uint64      `json:"seed"`
	Count      int         `json:"count"`
	Zones      []ZoneInfo  `json:"zones"`
	Challenges []Challenge `json:"challenges"`
}

// Manifest renders the registry for the given session.
func (r *Registry) Manifest(version, session string, seed uint64) Manifest {
	return Manifest{
		Version:    version,
		Session:    session,
		Seed:       seed,
		Count:      len(r.ordered),
		Zones:      populatedZones(r),
		Challenges: r.All(),
	}
}

// populatedZones lists only the zones this build actually serves, so the
// manifest never advertises an empty zone.
func populatedZones(r *Registry) []ZoneInfo {
	groups := r.ByZone()
	out := make([]ZoneInfo, 0, len(groups))
	for _, g := range groups {
		out = append(out, g.Zone)
	}
	return out
}
