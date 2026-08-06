// Package server wires the zones, the API and the session middleware into one
// http.Handler.
package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/render"
	"github.com/heyrmi/testground/internal/session"
)

// Zone is a set of challenge routes mounted under a path prefix.
type Zone struct {
	Prefix  string
	Handler http.Handler
}

// Options configures a Server. Everything it needs is passed in; the package
// reads no globals.
type Options struct {
	Registry *challenge.Registry
	Sessions *session.Store
	Renderer *render.Renderer
	Static   fs.FS
	Zones    []Zone
	Version  string
	Logger   *slog.Logger
}

// Server serves the playground.
type Server struct {
	opts   Options
	log    *slog.Logger
	router chi.Router
}

// New validates the options and builds the routing table.
func New(opts Options) (*Server, error) {
	switch {
	case opts.Registry == nil:
		return nil, errors.New("server: registry is required")
	case opts.Sessions == nil:
		return nil, errors.New("server: session store is required")
	case opts.Renderer == nil:
		return nil, errors.New("server: renderer is required")
	}
	if opts.Logger == nil {
		opts.Logger = discardLogger()
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}

	s := &Server{opts: opts, log: opts.Logger}
	s.router = s.routes()
	return s, nil
}

// Handler returns the root handler.
func (s *Server) Handler() http.Handler { return s.router }

func (s *Server) routes() chi.Router {
	r := chi.NewRouter()
	// GetHead answers HEAD from the GET handlers, so health checks and link
	// checkers do not see a 405 on every page.
	r.Use(middleware.GetHead, requestID, s.logRequests, s.recoverPanics)

	// Static assets sit outside the session middleware. They carry no state,
	// and minting a session for every stylesheet would churn the store.
	if s.opts.Static != nil {
		r.Handle("/static/*", http.StripPrefix("/static/", cacheableFileServer(s.opts.Static)))
	}
	r.Get("/api/health", s.handleHealth)
	r.Get("/api/version", s.handleVersion)

	r.Group(func(r chi.Router) {
		r.Use(s.opts.Sessions.Middleware)

		r.Get("/", s.handleIndex)
		r.Get("/api/challenges", s.handleManifest)
		r.Get("/api/challenges/{id}", s.handleChallenge)

		for _, zone := range s.opts.Zones {
			r.Mount(zone.Prefix, zone.Handler)
		}

		r.NotFound(s.handleNotFound)
		r.MethodNotAllowed(s.handleMethodNotAllowed)
	})

	return r
}

// Serve runs the playground until ctx is cancelled, then drains in-flight
// requests before returning.
func (s *Server) Serve(ctx context.Context, addr string) error {
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		// No write timeout: challenges deliberately stall, stream and hang,
		// and a server-side deadline would cut those short.
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	go s.opts.Sessions.Run(ctx)

	errs := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- fmt.Errorf("listening on %s: %w", addr, err)
			return
		}
		errs <- nil
	}()

	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpServer.Shutdown(shutdown)
	}
}

// discardLogger is the fallback for callers that do not want request logs.
func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
