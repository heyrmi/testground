package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	testground "github.com/heyrmi/testground"
	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/render"
	"github.com/heyrmi/testground/internal/session"
)

func fixture() challenge.Challenge {
	return challenge.Challenge{
		ID:        "delayed-element",
		Title:     "Delayed element",
		URL:       "/app/delayed-element",
		Zone:      challenge.ZoneApp,
		Tier:      challenge.T1,
		Category:  "C. Waits & Timing",
		Summary:   "An element appears after a fixed delay.",
		WhyHard:   "A naive locate runs before the element exists.",
		Hint:      "Wait for the element, not for a duration.",
		Tags:      []string{"waits"},
		Concepts:  []string{"explicit wait"},
		Selectors: []challenge.Selector{{TestID: "delayed-message", Note: "Appears after the delay"}},
		Stability: challenge.Stable,
	}
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()

	renderer, err := render.New(testground.Templates(), "0.0.0-test", discardLogger())
	if err != nil {
		t.Fatalf("parsing templates: %v", err)
	}
	srv, err := New(Options{
		Registry: challenge.MustRegistry([]challenge.Challenge{fixture()}),
		Sessions: session.NewStore(session.Options{Seed: 42}),
		Renderer: renderer,
		Static:   testground.Static(),
		Version:  "0.0.0-test",
	})
	if err != nil {
		t.Fatalf("building server: %v", err)
	}
	return srv.Handler()
}

func get(t *testing.T, h http.Handler, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestIndexListsEveryChallenge(t *testing.T) {
	rec := get(t, newTestServer(t), "/", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`data-testid="challenge-card-delayed-element"`,
		`data-testid="zone-app"`,
		"Delayed element",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("index page is missing %s", want)
		}
	}
}

func TestManifestIsSessionScoped(t *testing.T) {
	rec := get(t, newTestServer(t), "/api/challenges", map[string]string{
		session.Header: "worker-7",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}

	var manifest challenge.Manifest
	if err := json.Unmarshal(rec.Body.Bytes(), &manifest); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}
	if manifest.Session != "worker-7" {
		t.Errorf("manifest session %q, want worker-7", manifest.Session)
	}
	if manifest.Seed != 42 {
		t.Errorf("manifest seed %d, want 42", manifest.Seed)
	}
	if manifest.Count != 1 || len(manifest.Challenges) != 1 {
		t.Fatalf("manifest lists %d challenges, want 1", manifest.Count)
	}
	if got := manifest.Challenges[0].Hint; got == "" {
		t.Error("manifest omitted the hint tests read to self-describe")
	}
}

func TestSingleChallengeLookup(t *testing.T) {
	h := newTestServer(t)

	if rec := get(t, h, "/api/challenges/delayed-element", nil); rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	rec := get(t, h, "/api/challenges/nope", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"error"`) {
		t.Errorf("unstructured error body: %s", rec.Body.String())
	}
}

func TestHealthAndVersionNeedNoSession(t *testing.T) {
	h := newTestServer(t)

	for _, path := range []string{"/api/health", "/api/version"} {
		rec := get(t, h, path, nil)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status %d, want 200", path, rec.Code)
		}
		if got := rec.Header().Get(session.Header); got != "" {
			t.Errorf("%s minted session %q; monitoring should not churn the store", path, got)
		}
	}
}

func TestStaticAssetsAreServedWithoutASession(t *testing.T) {
	rec := get(t, newTestServer(t), "/static/shell.css", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	if got := rec.Header().Get(session.Header); got != "" {
		t.Errorf("stylesheet minted session %q", got)
	}
}

func TestUnknownRouteRendersAPageButUnknownAPIReturnsJSON(t *testing.T) {
	h := newTestServer(t)

	page := get(t, h, "/no/such/page", nil)
	if page.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", page.Code)
	}
	if !strings.Contains(page.Body.String(), `data-testid="not-found-heading"`) {
		t.Error("404 did not render the page template")
	}

	api := get(t, h, "/api/no-such-route", nil)
	if api.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", api.Code)
	}
	if ct := api.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("API 404 content type %q, want JSON", ct)
	}
}

func TestEveryResponseAdvertisesItsRequestID(t *testing.T) {
	rec := get(t, newTestServer(t), "/", nil)

	if rec.Header().Get(RequestIDHeader) == "" {
		t.Error("no request id to correlate a failing test with the log")
	}
}
