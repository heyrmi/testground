// Package playground is the composition root: it is the one place that knows
// which zones exist and wires them to the server. Everything below it takes
// its collaborators as arguments and reads no globals.
package playground

import (
	"fmt"
	"log/slog"
	"time"

	testground "github.com/heyrmi/testground"
	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/render"
	"github.com/heyrmi/testground/internal/server"
	"github.com/heyrmi/testground/internal/session"
	"github.com/heyrmi/testground/internal/zones/app"
	"github.com/heyrmi/testground/internal/zones/wc"
)

// Config is everything the operator can choose at startup.
type Config struct {
	Seed       uint64
	SessionTTL time.Duration
	Version    string
	Logger     *slog.Logger
}

// Registry returns every challenge this build serves. It is also what the
// manifest subcommand prints, so a build can be inspected without running it.
func Registry() (*challenge.Registry, error) {
	return challenge.NewRegistry(
		app.Challenges(),
		wc.Challenges(),
	)
}

// zones lists every frontend the server mounts.
func zones(renderer *render.Renderer) []server.Zone {
	dist := testground.AppDist()
	return []server.Zone{
		{ID: challenge.ZoneApp, Prefix: "/app", Pages: app.Pages(dist), API: app.API()},
		{ID: challenge.ZoneComponents, Prefix: "/wc", Pages: wc.Pages(renderer)},
	}
}

// New assembles the playground.
func New(cfg Config) (*server.Server, error) {
	registry, err := Registry()
	if err != nil {
		return nil, fmt.Errorf("building challenge registry: %w", err)
	}

	renderer, err := render.New(testground.Templates(), cfg.Version, cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}

	return server.New(server.Options{
		Registry: registry,
		Sessions: session.NewStore(session.Options{Seed: cfg.Seed, TTL: cfg.SessionTTL}),
		Renderer: renderer,
		Static:   testground.Static(),
		Assets:   testground.AppDist(),
		Zones:    zones(renderer),
		Version:  cfg.Version,
		Logger:   cfg.Logger,
	})
}
