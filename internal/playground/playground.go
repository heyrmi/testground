// Package playground is the composition root: it is the one place that knows
// which zones exist and wires them to the server. Everything below it takes
// its collaborators as arguments and reads no globals.
package playground

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	testground "github.com/heyrmi/testground"
	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/crossorigin"
	"github.com/heyrmi/testground/internal/render"
	"github.com/heyrmi/testground/internal/server"
	"github.com/heyrmi/testground/internal/session"
	"github.com/heyrmi/testground/internal/zones/app"
	"github.com/heyrmi/testground/internal/zones/classic"
	"github.com/heyrmi/testground/internal/zones/wc"
)

// Config is everything the operator can choose at startup.
type Config struct {
	Seed       uint64
	SessionTTL time.Duration
	Version    string
	Logger     *slog.Logger

	// CrossOriginAddr is the second address to bind. The browser decides what
	// is same-origin from scheme, host and port, so a genuinely different
	// origin needs a genuinely different socket. Empty disables it, and the
	// challenges that need it are then not registered at all rather than
	// registered and broken.
	CrossOriginAddr string
}

// CrossOriginEnabled reports whether the second origin will be served.
func (c Config) CrossOriginEnabled() bool { return c.CrossOriginAddr != "" }

// Registry returns every challenge this build serves under cfg. It is also
// what the manifest subcommand prints, so a build can be inspected without
// running it.
func Registry(cfg Config) (*challenge.Registry, error) {
	port, err := portOf(cfg.CrossOriginAddr)
	if err != nil {
		return nil, err
	}
	return challenge.NewRegistry(
		app.Challenges(),
		classic.Challenges(classic.Options{CrossOriginPort: port}),
		wc.Challenges(),
	)
}

// Playground is the pair of servers the binary runs: the playground itself,
// and the second origin its cross-origin challenges embed.
type Playground struct {
	Main  *server.Server
	Cross *server.Server
}

// New assembles the playground.
func New(cfg Config) (*Playground, error) {
	registry, err := Registry(cfg)
	if err != nil {
		return nil, fmt.Errorf("building challenge registry: %w", err)
	}

	renderer, err := render.New(testground.Templates(), cfg.Version, cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}

	// One store, shared by both origins. Two ports are two origins to the
	// browser but one playground to the tester, and a challenge that spans
	// them has to see the same state from either side.
	sessions := session.NewStore(session.Options{Seed: cfg.Seed, TTL: cfg.SessionTTL})

	crossOriginPort, err := portOf(cfg.CrossOriginAddr)
	if err != nil {
		return nil, err
	}

	main, err := server.New(server.Options{
		Registry: registry,
		Sessions: sessions,
		Renderer: renderer,
		Static:   testground.Static(),
		Assets:   testground.AppDist(),
		Zones:    zones(renderer, crossOriginPort),
		Version:  cfg.Version,
		Logger:   cfg.Logger,
	})
	if err != nil {
		return nil, err
	}

	built := &Playground{Main: main}
	if !cfg.CrossOriginEnabled() {
		return built, nil
	}

	built.Cross, err = server.New(server.Options{
		Registry: registry,
		Sessions: sessions,
		Renderer: renderer,
		Static:   testground.Static(),
		Zones:    []server.Zone{{ID: challenge.ZoneClassic, Prefix: "/", Pages: crossorigin.Routes(renderer)}},
		Version:  cfg.Version,
		Logger:   cfg.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("building the second origin: %w", err)
	}
	return built, nil
}

// zones lists every frontend the main server mounts.
func zones(renderer *render.Renderer, crossOriginPort string) []server.Zone {
	dist := testground.AppDist()
	return []server.Zone{
		{
			ID:     challenge.ZoneClassic,
			Prefix: "/classic",
			Pages:  classic.Pages(renderer, classic.Options{CrossOriginPort: crossOriginPort}),
		},
		{ID: challenge.ZoneApp, Prefix: "/app", Pages: app.Pages(dist), API: app.API()},
		{ID: challenge.ZoneComponents, Prefix: "/wc", Pages: wc.Pages(renderer)},
	}
}

// Serve runs both origins until ctx is cancelled. Either one failing to bind
// fails the whole start, because a challenge that silently lost its second
// origin is worse than one that refused to start.
func (p *Playground) Serve(ctx context.Context, mainListener, crossListener net.Listener) error {
	if crossListener == nil {
		return p.Main.Serve(ctx, mainListener)
	}

	ctx, stop := context.WithCancel(ctx)
	defer stop()

	errs := make(chan error, 2)
	go func() { errs <- p.Cross.Serve(ctx, crossListener) }()
	go func() { errs <- p.Main.Serve(ctx, mainListener) }()

	first := <-errs
	stop()
	if second := <-errs; first == nil {
		first = second
	}
	return first
}

// portOf extracts the port from a listen address, so a page can build a URL
// on the second origin from whatever host the browser used to reach the first.
func portOf(addr string) (string, error) {
	if addr == "" {
		return "", nil
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("cross-origin address %q needs a host and port: %w", addr, err)
	}
	return port, nil
}
