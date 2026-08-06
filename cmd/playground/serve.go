package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/heyrmi/testground/internal/build"
	"github.com/heyrmi/testground/internal/playground"
	"github.com/heyrmi/testground/internal/rng"
	"github.com/heyrmi/testground/internal/session"
)

func newServeCommand(logs *logOptions) *cobra.Command {
	var (
		addr string
		seed uint64
		ttl  time.Duration
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the challenge suite",
		Long: `Serve the challenge suite.

Binds to loopback by default. Pass --addr 0.0.0.0:7373 to expose it to other
machines, which is what a container or a shared workshop host wants.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger, err := newLogger(*logs)
			if err != nil {
				return err
			}

			version := build.Current().Version
			srv, err := playground.New(playground.Config{
				Seed:       seed,
				SessionTTL: ttl,
				Version:    version,
				Logger:     logger,
			})
			if err != nil {
				return err
			}

			// Bind first, announce second: a failed bind should report the
			// failure rather than a URL that was never served.
			listener, err := srv.Listen(addr)
			if err != nil {
				return err
			}

			cmd.Printf("testground %s listening on http://%s (seed %d)\n", version, listener.Addr(), seed)
			return srv.Serve(cmd.Context(), listener)
		},
	}

	cmd.Flags().StringVarP(&addr, "addr", "a", "127.0.0.1:7373", "address to listen on")
	cmd.Flags().Uint64Var(&seed, "seed", rng.DefaultSeed, "seed every session starts from")
	cmd.Flags().DurationVar(&ttl, "session-ttl", session.DefaultTTL, "how long an idle session survives")

	return cmd
}

func newVersionCommand() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of this binary",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			info := build.Current()
			if asJSON {
				return printJSON(cmd.OutOrStdout(), info)
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "testground %s (commit %s, built %s, %s)\n",
				info.Version, orUnknown(info.Commit), orUnknown(info.Date), info.Go)
			return err
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print machine-readable output")
	return cmd
}

func orUnknown(v string) string {
	if v == "" {
		return "unknown"
	}
	return v
}
