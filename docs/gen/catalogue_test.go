package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/playground"
)

// liveSite builds the catalogue from the real registry rather than from a
// fixture. A fixture would be a third copy of the challenge declarations, and
// the whole point of generating this page is that there is only ever one.
func liveSite(t *testing.T) (*site, challenge.Manifest) {
	t.Helper()
	registry, err := playground.Registry(playground.Config{CrossOriginAddr: "127.0.0.1:7374"})
	if err != nil {
		t.Fatal(err)
	}
	manifest := registry.Manifest("v0.0.0-test", "test", 42)
	return &site{manifest: manifest}, manifest
}

// TestCatalogueCoversEveryChallenge is the check the generated page exists for.
// A catalogue that quietly omits a challenge looks complete, and the omission
// is only ever found by the reader who went looking for the page that is not
// there.
func TestCatalogueCoversEveryChallenge(t *testing.T) {
	s, manifest := liveSite(t)

	body, contents, err := s.catalogue()
	if err != nil {
		t.Fatal(err)
	}
	index := s.index()

	for _, c := range manifest.Challenges {
		if !strings.Contains(body, `id="ch-`+c.ID+`"`) {
			t.Errorf("the catalogue has no entry for %q", c.ID)
		}
		if !strings.Contains(index, `href="#ch-`+c.ID+`"`) {
			t.Errorf("the index does not link to %q", c.ID)
		}
		if !strings.Contains(body, escape(c.Summary)) {
			t.Errorf("the entry for %q does not carry its summary", c.ID)
		}
		if !strings.Contains(body, escape(c.Hint)) {
			t.Errorf("the entry for %q does not carry its hint", c.ID)
		}
		for _, selector := range c.Selectors {
			if !strings.Contains(body, "<code>"+escape(selector.TestID)+"</code>") {
				t.Errorf("the entry for %q does not declare the selector %q",
					c.ID, selector.TestID)
			}
		}
	}

	// Every zone in the manifest is a section, so a zone cannot be published in
	// the manifest and be missing from the page that claims to list them all.
	if len(contents) != len(manifest.Zones) {
		t.Errorf("got %d zone sections, want %d", len(contents), len(manifest.Zones))
	}
}

func TestZoneTableCountsAddUp(t *testing.T) {
	s, manifest := liveSite(t)

	total := 0
	for _, group := range s.zonesInOrder() {
		total += len(group.Challenges)
		if !strings.Contains(s.zoneTable(), escape(group.Zone.Prefix)) {
			t.Errorf("the zone table omits %q", group.Zone.Prefix)
		}
	}
	// A challenge counted in no zone would be missing from the catalogue while
	// the manifest still advertised it, which is the drift this page exists to
	// make impossible.
	if total != manifest.Count {
		t.Errorf("the zones hold %d challenges between them, but the manifest counts %d",
			total, manifest.Count)
	}
}

// TestCatalogueEscapesManifestProse matters because the prose is written in Go
// source files by contributors, not by whoever wrote this generator. A summary
// mentioning a tag or an ampersand must render as text rather than becoming
// markup on a published page.
func TestCatalogueEscapesManifestProse(t *testing.T) {
	s := &site{manifest: challenge.Manifest{
		Count: 1,
		Zones: []challenge.ZoneInfo{{ID: "app", Title: "Modern SPA", Prefix: "/app"}},
		Challenges: []challenge.Challenge{{
			ID:      "x",
			Title:   "A <script> in the title",
			Zone:    "app",
			Summary: "Renders <b>markup</b> & entities",
			WhyHard: "Nothing <i>here</i> is markup",
			Hint:    "Neither <em>is</em> this",
			Tags:    []string{"<tag>"},
		}},
	}}

	body, _, err := s.catalogue()
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"<script>", "<b>", "<i>", "<em>", "<tag>"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("%q reached the page unescaped:\n%s", forbidden, body)
		}
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Error("the title was not escaped at all")
	}
}

func TestUnknownHoleFailsTheBuild(t *testing.T) {
	s, _ := liveSite(t)
	if _, _, err := s.hole("not-a-generator"); err == nil {
		t.Fatal("want an error for a hole nobody generates")
	}
}

// TestSiteBuilds renders every page exactly as a deploy does. Link resolution
// is strict, so this is what turns a cross-reference to a renamed page into a
// red suite rather than into a dead link on a published site.
func TestSiteBuilds(t *testing.T) {
	root, err := repositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	live, manifest := liveSite(t)

	s := &site{root: root, out: t.TempDir(), manifest: live.manifest}
	if err := s.build(); err != nil {
		t.Fatal(err)
	}

	for _, p := range pages {
		body, err := os.ReadFile(filepath.Join(s.out, p.Slug+".html"))
		if err != nil {
			t.Fatalf("page %q: %v", p.Slug, err)
		}
		// Every page carries the whole sidebar, so a page that renders as an
		// island is a page whose layout stopped being applied.
		for _, other := range pages {
			if !strings.Contains(string(body), `href="`+other.Slug+`.html"`) {
				t.Errorf("page %q does not link to %q", p.Slug, other.Slug)
			}
		}
	}

	catalogue, err := os.ReadFile(filepath.Join(s.out, "catalogue.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range manifest.Challenges {
		if !strings.Contains(string(catalogue), `id="ch-`+c.ID+`"`) {
			t.Errorf("the built catalogue is missing %q", c.ID)
		}
	}

	// The stylesheet is the site's only asset, and a page without it is
	// unreadable rather than merely plain.
	if _, err := os.Stat(filepath.Join(s.out, "assets", "site.css")); err != nil {
		t.Errorf("the stylesheet was not written: %v", err)
	}
}

func TestVersionLabel(t *testing.T) {
	cases := map[string]string{
		"v1.0.0":  "v1.0.0",
		"(devel)": "an untagged build of main",
		"dev":     "an untagged build of main",
		"":        "an untagged build of main",
	}
	for version, want := range cases {
		s := &site{manifest: challenge.Manifest{Version: version}}
		if got := s.versionLabel(); got != want {
			t.Errorf("versionLabel(%q) = %q, want %q", version, got, want)
		}
	}
}
