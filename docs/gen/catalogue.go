package main

import (
	"fmt"
	"html/template"
	"sort"
	"strings"

	"github.com/heyrmi/testground/internal/challenge"
)

// The catalogue is generated from the manifest and never written by hand.
//
// A hand-written list of forty-four pages is a second declaration of facts that
// already exist in the challenge registry, and the two drift the moment someone
// adds a page and updates one of them. Everything below reads the same document
// the /api/challenges endpoint serves, so a challenge that is missing from this
// page is a challenge that is missing from the product.

// hole dispatches one <!--generated:name--> placeholder. An unknown name stops
// the build rather than rendering nothing, because a hole that silently
// disappears leaves a page that is merely shorter than it should be, which is
// the sort of omission nobody notices in review.
func (s *site) hole(name string) (string, []heading, error) {
	switch name {
	case "catalogue-stats":
		return s.stats(), nil, nil
	case "challenge-index":
		return s.index(), nil, nil
	case "catalogue":
		return s.catalogue()
	case "zone-table":
		return s.zoneTable(), nil, nil
	}
	return "", nil, fmt.Errorf("no generator called %q", name)
}

// zoneGroup is one zone and the challenges the manifest places in it.
type zoneGroup struct {
	Zone       challenge.ZoneInfo
	Challenges []challenge.Challenge
}

// zonesInOrder groups the challenges by zone, keeping the manifest's
// presentation order, which runs oldest technique first. Within a zone the
// order is by title rather than by the registry's, so that a reader scanning
// for a page finds it where its name says it should be.
func (s *site) zonesInOrder() []zoneGroup {
	groups := make([]zoneGroup, 0, len(s.manifest.Zones))
	for _, zone := range s.manifest.Zones {
		g := zoneGroup{Zone: zone}
		for _, c := range s.manifest.Challenges {
			if c.Zone == zone.ID {
				g.Challenges = append(g.Challenges, c)
			}
		}
		sort.SliceStable(g.Challenges, func(i, j int) bool {
			return g.Challenges[i].Title < g.Challenges[j].Title
		})
		groups = append(groups, g)
	}
	return groups
}

func (s *site) stats() string {
	var concepts, tags, selectors []string
	tiers := map[challenge.Tier]int{}
	experimental := 0
	for _, c := range s.manifest.Challenges {
		concepts = append(concepts, c.Concepts...)
		tags = append(tags, c.Tags...)
		for _, selector := range c.Selectors {
			selectors = append(selectors, c.ID+"/"+selector.TestID)
		}
		tiers[c.Tier]++
		if c.Stability != challenge.Stable {
			experimental++
		}
	}

	var out strings.Builder
	out.WriteString(`<dl class="stats">`)
	figure(&out, s.manifest.Count, "challenges")
	figure(&out, len(s.manifest.Zones), "zones")
	figure(&out, len(sortedUnique(concepts)), "distinct concepts")
	figure(&out, len(sortedUnique(tags)), "tags")
	figure(&out, len(selectors), "declared selectors")
	out.WriteString("</dl>\n")

	stability := `Every entry is stable, so all of them are covered by ` +
		`<a href="stability-contract.html">the stability contract</a>.`
	if experimental > 0 {
		stability = fmt.Sprintf(`%d of the %d are still experimental and so `+
			`outside <a href="stability-contract.html">the stability contract</a>.`,
			experimental, s.manifest.Count)
	}

	fmt.Fprintf(&out, `<p class="stats__note">`+
		`Difficulty runs T1 (%d), T2 (%d), T3 (%d), T4 (%d). %s `+
		`Generated from the manifest of %s.</p>`+"\n",
		tiers[challenge.T1], tiers[challenge.T2], tiers[challenge.T3], tiers[challenge.T4],
		stability, escape(s.versionLabel()))
	return out.String()
}

// versionLabel names the build the catalogue was generated from. A build made
// from a checkout rather than from a tag calls itself "(devel)", which tells a
// reader nothing, so it is spelled out instead: a site published from main is
// ahead of every release and had better say so.
func (s *site) versionLabel() string {
	switch s.manifest.Version {
	case "", "dev", "(devel)":
		return "an untagged build of main"
	}
	return s.manifest.Version
}

func figure(out *strings.Builder, value int, label string) {
	fmt.Fprintf(out, `<div><dt>%d</dt><dd>%s</dd></div>`, value, escape(label))
}

func (s *site) zoneTable() string {
	var out strings.Builder
	out.WriteString(`<div class="table-scroll">` + "\n<table>\n<thead>\n" +
		"<tr><th>Zone</th><th>Path</th><th>Technology</th>" +
		"<th style=\"text-align:right\">Challenges</th><th>What it exercises</th></tr>\n" +
		"</thead>\n<tbody>\n")
	for _, group := range s.zonesInOrder() {
		fmt.Fprintf(&out,
			"<tr><td>%s</td><td><code>%s</code></td><td>%s</td>"+
				"<td style=\"text-align:right\">%d</td><td>%s</td></tr>\n",
			escape(group.Zone.Title), escape(group.Zone.Prefix),
			escape(group.Zone.Technology), len(group.Challenges),
			escape(group.Zone.Tests))
	}
	out.WriteString("</tbody>\n</table>\n</div>\n")
	return out.String()
}

func (s *site) index() string {
	var out strings.Builder
	out.WriteString(`<div class="table-scroll">` + "\n<table>\n<thead>\n" +
		"<tr><th>Challenge</th><th>Zone</th><th>Tier</th><th>Category</th></tr>\n" +
		"</thead>\n<tbody>\n")
	for _, group := range s.zonesInOrder() {
		for _, c := range group.Challenges {
			fmt.Fprintf(&out,
				`<tr><td><a href="#%s">%s</a></td><td>%s</td><td>%s</td><td>%s</td></tr>`+"\n",
				anchor(c), escape(c.Title), escape(group.Zone.Title), tierBadge(c.Tier),
				escape(c.Category))
		}
	}
	out.WriteString("</tbody>\n</table>\n</div>\n")
	return out.String()
}

// anchor prefixes the challenge id so that a challenge can never collide with
// a heading slug on the same page, which would leave one of the two
// unreachable by the link the other publishes.
func anchor(c challenge.Challenge) string { return "ch-" + c.ID }

func tierBadge(tier challenge.Tier) string {
	return fmt.Sprintf(`<span class="tier tier--%s">%s</span>`,
		strings.ToLower(string(tier)), escape(string(tier)))
}

func (s *site) catalogue() (string, []heading, error) {
	var (
		out      strings.Builder
		contents []heading
	)
	for _, group := range s.zonesInOrder() {
		id := "zone-" + string(group.Zone.ID)
		contents = append(contents, heading{
			Level: 2,
			Text: template.HTML(fmt.Sprintf("%s <span class=\"count\">%d</span>",
				escape(group.Zone.Title), len(group.Challenges))),
			ID: id,
		})

		fmt.Fprintf(&out, `<section class="zone">`+"\n"+
			`<h2 id=%q>%s<a class="anchor" href="#%s" aria-label="Link to this section">#</a></h2>`+"\n"+
			`<p class="zone__meta"><code>%s</code> — %s</p>`+"\n",
			id, escape(group.Zone.Title), id,
			escape(group.Zone.Prefix), escape(group.Zone.Technology))

		if len(group.Challenges) == 0 {
			out.WriteString("<p>This build ships no challenges in this zone.</p>\n</section>\n")
			continue
		}
		for _, c := range group.Challenges {
			s.challenge(&out, c)
		}
		out.WriteString("</section>\n")
	}
	return out.String(), contents, nil
}

func (s *site) challenge(out *strings.Builder, c challenge.Challenge) {
	fmt.Fprintf(out, `<article class="challenge" id=%q>`+"\n"+
		`<h3>%s %s<a class="anchor" href="#%s" aria-label="Link to this challenge">#</a></h3>`+"\n",
		anchor(c), escape(c.Title), tierBadge(c.Tier), anchor(c))

	fmt.Fprintf(out, `<p class="challenge__meta"><code>%s</code> · %s · %s</p>`+"\n",
		escape(c.URL), escape(c.Category), escape(string(c.Stability)))

	if c.HostileLocators {
		out.WriteString(`<p class="challenge__warning">This page withholds ` +
			`<code>data-testid</code> attributes on purpose: locating the elements ` +
			`is the exercise.</p>` + "\n")
	}

	fmt.Fprintf(out, "<p>%s</p>\n", escape(c.Summary))
	fmt.Fprintf(out, `<p class="challenge__why"><strong>Why it breaks naive automation.</strong> %s</p>`+"\n",
		escape(c.WhyHard))
	fmt.Fprintf(out, "<details><summary>Hint</summary><p>%s</p></details>\n", escape(c.Hint))

	chips(out, "Concepts", c.Concepts)
	chips(out, "Tags", c.Tags)
	selectorTable(out, c.Selectors)
	endpointTable(out, c.Endpoints)
	controlTable(out, c.Controls)

	out.WriteString("</article>\n")
}

func chips(out *strings.Builder, label string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(out, `<p class="chips"><span class="chips__label">%s</span>`, escape(label))
	for _, value := range values {
		fmt.Fprintf(out, `<span class="chip">%s</span>`, escape(value))
	}
	out.WriteString("</p>\n")
}

func selectorTable(out *strings.Builder, selectors []challenge.Selector) {
	if len(selectors) == 0 {
		return
	}
	out.WriteString(`<div class="table-scroll">` + "\n<table class=\"selectors\">\n<thead>\n" +
		"<tr><th>Test id</th><th>Role</th><th>Note</th></tr>\n</thead>\n<tbody>\n")
	for _, selector := range selectors {
		fmt.Fprintf(out, "<tr><td><code>%s</code>%s</td><td>%s</td><td>%s</td></tr>\n",
			escape(selector.TestID), selectorFlags(selector),
			escape(selector.Role), escape(selector.Note))
	}
	out.WriteString("</tbody>\n</table>\n</div>\n")
}

// selectorFlags spells out the three things that change how a selector is
// located: a family it represents, the frames it lives inside, and whether it
// is present at all before the interaction that creates it. A test written
// against the id alone and none of these fails for reasons the manifest
// already knew about.
func selectorFlags(selector challenge.Selector) string {
	var flags []string
	if selector.Family != "" {
		flags = append(flags, `<span class="flag">family <code>`+
			escape(selector.Family)+`</code></span>`)
	}
	if selector.Transient {
		flags = append(flags, `<span class="flag">transient</span>`)
	}
	if len(selector.Frame) > 0 {
		flags = append(flags, `<span class="flag">inside <code>`+
			escape(strings.Join(selector.Frame, " › "))+`</code></span>`)
	}
	if len(flags) == 0 {
		return ""
	}
	return `<span class="flags">` + strings.Join(flags, "") + "</span>"
}

func endpointTable(out *strings.Builder, endpoints []challenge.Endpoint) {
	if len(endpoints) == 0 {
		return
	}
	out.WriteString(`<div class="table-scroll">` + "\n<table>\n<thead>\n" +
		"<tr><th>Endpoint</th><th>Note</th></tr>\n</thead>\n<tbody>\n")
	for _, endpoint := range endpoints {
		fmt.Fprintf(out, "<tr><td><code>%s %s</code></td><td>%s</td></tr>\n",
			escape(endpoint.Method), escape(endpoint.Path), escape(endpoint.Note))
	}
	out.WriteString("</tbody>\n</table>\n</div>\n")
}

func controlTable(out *strings.Builder, controls []challenge.Control) {
	if len(controls) == 0 {
		return
	}
	out.WriteString(`<div class="table-scroll">` + "\n<table>\n<thead>\n" +
		"<tr><th>Control</th><th>Kind</th><th>Default</th><th>Note</th></tr>\n" +
		"</thead>\n<tbody>\n")
	for _, control := range controls {
		fmt.Fprintf(out, "<tr><td><code>%s</code></td><td>%s</td><td><code>%s</code></td><td>%s</td></tr>\n",
			escape(control.Name), escape(control.Kind), escape(control.Default),
			escape(control.Note))
	}
	out.WriteString("</tbody>\n</table>\n</div>\n")
}
