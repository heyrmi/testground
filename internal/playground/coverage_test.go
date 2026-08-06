package playground

import (
	"os"
	"path/filepath"
	"testing"
)

// referenceSuite is the Playwright suite, reached from this package's
// directory. It doubles as the project's integration suite, so a challenge
// without a spec there ships untested.
const referenceSuite = "../../examples/playwright-ts/tests"

// CONTRIBUTING promises every challenge arrives with a reference test. The
// registry cannot check that, so this does: without it the rule is a request
// rather than a requirement.
func TestEveryChallengeHasAReferenceSpec(t *testing.T) {
	registry, err := Registry(Config{CrossOriginAddr: "127.0.0.1:7374"})
	if err != nil {
		t.Fatalf("building registry: %v", err)
	}

	for _, c := range registry.All() {
		spec := filepath.Join(referenceSuite, c.ID+".spec.ts")
		if _, err := os.Stat(spec); err != nil {
			t.Errorf("challenge %q has no reference test at %s", c.ID, spec)
		}
	}
}

// The reverse direction catches a spec left behind after a challenge was
// renamed, which would otherwise sit there testing nothing.
func TestEveryReferenceSpecMatchesAChallenge(t *testing.T) {
	registry, err := Registry(Config{CrossOriginAddr: "127.0.0.1:7374"})
	if err != nil {
		t.Fatalf("building registry: %v", err)
	}

	// Specs that cover the playground itself rather than one challenge.
	crossCutting := map[string]bool{
		"manifest.spec.ts":          true,
		"session-isolation.spec.ts": true,
	}

	specs, err := filepath.Glob(filepath.Join(referenceSuite, "*.spec.ts"))
	if err != nil {
		t.Fatalf("globbing specs: %v", err)
	}
	if len(specs) == 0 {
		t.Fatal("no reference specs found; the suite has moved")
	}

	for _, spec := range specs {
		name := filepath.Base(spec)
		if crossCutting[name] {
			continue
		}
		id := name[:len(name)-len(".spec.ts")]
		if _, ok := registry.Lookup(id); !ok {
			t.Errorf("spec %s covers no registered challenge; rename it or remove it", name)
		}
	}
}
