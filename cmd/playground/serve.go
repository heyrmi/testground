package main

import (
	"fmt"
	"net"
	"time"

	"github.com/spf13/cobra"

	"github.com/heyrmi/testground/internal/build"
	"github.com/heyrmi/testground/internal/playground"
	"github.com/heyrmi/testground/internal/rng"
	"github.com/heyrmi/testground/internal/session"
)

func newServeCommand(logs *logOptions) *cobra.Command {
	var (
		addr      string
		crossAddr string
		seed      uint64
		ttl       time.Duration
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the challenge suite",
		Long: `Serve the challenge suite.

Binds to loopback by default. Pass --addr 0.0.0.0:7373 to expose it to other
machines, which is what a container or a shared workshop host wants.

A second address is bound as well. The browser decides what is same-origin
from scheme, host and port, so the frame challenges need a genuinely different
socket to embed -- it is the same binary sharing the same session store, on
another port. Pass --cross-origin-addr "" to bind only one, in which case the
challenges that need the second origin are not registered at all.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger, err := newLogger(*logs)
			if err != nil {
				return err
			}

			version := build.Current().Version
			ground, err := playground.New(playground.Config{
				Seed:            seed,
				SessionTTL:      ttl,
				Version:         version,
				Logger:          logger,
				CrossOriginAddr: crossAddr,
			})
			if err != nil {
				return err
			}

			// Bind first, announce second: a failed bind should report the
			// failure rather than a URL that was never served.
			listener, err := ground.Main.Listen(addr)
			if err != nil {
				return err
			}
			defer listener.Close()

			var crossListener net.Listener
			if ground.Cross != nil {
				if crossListener, err = ground.Cross.Listen(crossAddr); err != nil {
					return err
				}
				defer crossListener.Close()
			}

			cmd.Printf("testground %s listening on http://%s (seed %d)\n", version, listener.Addr(), seed)
			if crossListener != nil {
				cmd.Printf("second origin on http://%s, for the cross-origin frame challenges\n", crossListener.Addr())
			}
			return ground.Serve(cmd.Context(), listener, crossListener)
		},
	}

	cmd.Flags().StringVarP(&addr, "addr", "a", "127.0.0.1:7373", "address to listen on")
	cmd.Flags().StringVar(&crossAddr, "cross-origin-addr", "127.0.0.1:7374", "second address to bind for the cross-origin challenges; empty to disable")
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
