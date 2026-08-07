package playground

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The two reference suites, reached from this package's directory. They double
// as the project's integration suite, so a challenge missing from either ships
// half tested.
const (
	playwrightSuite = "../../examples/playwright-ts/tests"
	seleniumSuite   = "../../examples/selenium-java/src/test/java/dev/testground"
)

// suite describes how one reference suite names the file covering a challenge,
// and which of its files cover the playground itself rather than one page.
//
// Both directions are checked for both suites. Reversing the check is what
// catches a file left behind after a challenge was renamed, which would
// otherwise sit there testing nothing while the forward check stayed green.
type suite struct {
	name         string
	dir          string
	fileFor      func(id string) string
	idFor        func(file string) (string, bool)
	crossCutting map[string]bool
}

func suites() []suite {
	return []suite{
		{
			name:    "Playwright",
			dir:     playwrightSuite,
			fileFor: func(id string) string { return id + ".spec.ts" },
			idFor: func(file string) (string, bool) {
				return strings.TrimSuffix(file, ".spec.ts"), strings.HasSuffix(file, ".spec.ts")
			},
			crossCutting: map[string]bool{
				"manifest.spec.ts":          true,
				"session-isolation.spec.ts": true,
				"control-plane.spec.ts":     true,
				"fixtures.ts":               true,
			},
		},
		{
			name:    "Selenium",
			dir:     seleniumSuite,
			fileFor: func(id string) string { return className(id) + ".java" },
			idFor: func(file string) (string, bool) {
				name, ok := strings.CutSuffix(file, "Test.java")
				return kebab(name), ok
			},
			crossCutting: map[string]bool{
				// The base class every test extends, and the proof of the
				// isolation they all rely on. Neither covers a challenge.
				"Playground.java":           true,
				"SessionIsolationTest.java": true,
			},
		},
	}
}

// className turns a challenge id into the JUnit class that covers it:
// otp-input becomes OtpInputTest. The suite is named by this rule rather than
// by a lookup table, so a new challenge needs no edit here.
func className(id string) string {
	var name strings.Builder
	for _, part := range strings.Split(id, "-") {
		if part == "" {
			continue
		}
		name.WriteString(strings.ToUpper(part[:1]))
		name.WriteString(part[1:])
	}
	name.WriteString("Test")
	return name.String()
}

// kebab is className's inverse, used to report a stray class against the id it
// claims to cover.
func kebab(name string) string {
	var id strings.Builder
	for i, r := range name {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				id.WriteByte('-')
			}
			id.WriteRune(r - 'A' + 'a')
			continue
		}
		id.WriteRune(r)
	}
	return id.String()
}

// CONTRIBUTING and PRD section 5 both promise every challenge arrives with a
// reference test in both frameworks. The registry cannot check that, so this
// does: without it the rule is a request rather than a requirement.
func TestEveryChallengeHasAReferenceTest(t *testing.T) {
	registry, err := Registry(Config{CrossOriginAddr: "127.0.0.1:7374"})
	if err != nil {
		t.Fatalf("building registry: %v", err)
	}

	for _, s := range suites() {
		for _, c := range registry.All() {
			path := filepath.Join(s.dir, s.fileFor(c.ID))
			if _, err := os.Stat(path); err != nil {
				t.Errorf("challenge %q has no %s reference test at %s", c.ID, s.name, path)
			}
		}
	}
}

// The reverse direction catches a test left behind after a challenge was
// renamed, which would otherwise sit there testing nothing.
func TestEveryReferenceTestMatchesAChallenge(t *testing.T) {
	registry, err := Registry(Config{CrossOriginAddr: "127.0.0.1:7374"})
	if err != nil {
		t.Fatalf("building registry: %v", err)
	}

	for _, s := range suites() {
		entries, err := os.ReadDir(s.dir)
		if err != nil {
			t.Fatalf("reading the %s suite: %v", s.name, err)
		}

		found := 0
		for _, entry := range entries {
			file := entry.Name()
			if entry.IsDir() || s.crossCutting[file] {
				continue
			}
			id, ok := s.idFor(file)
			if !ok {
				continue
			}
			found++
			if _, known := registry.Lookup(id); !known {
				t.Errorf("%s test %s covers no registered challenge %q; rename it or remove it", s.name, file, id)
			}
		}
		if found == 0 {
			t.Errorf("no %s reference tests found; the suite has moved", s.name)
		}
	}
}
