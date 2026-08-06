package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/heyrmi/testground/internal/build"
)

type logOptions struct {
	level  string
	format string
}

func newRootCommand() *cobra.Command {
	var logs logOptions

	root := &cobra.Command{
		Use:   "playground",
		Short: "A deterministic, offline testing playground for QA engineers",
		Long: strings.TrimSpace(`
A deterministic, offline testing playground for QA engineers.

Every challenge runs from this binary. Nothing is fetched from a network, page
contracts are frozen once released, and the same seed produces the same content
on every run and every machine.`),
		// Usage is silenced so a runtime failure reports the failure rather
		// than a wall of flags. Errors are not silenced: cobra prints them
		// with the prefix set below, which is the only thing standing between
		// a bad --addr and an exit code with no explanation.
		SilenceUsage: true,
		Version:      build.Current().Version,
	}

	root.PersistentFlags().StringVar(&logs.level, "log-level", "info", "log level: debug, info, warn or error")
	root.PersistentFlags().StringVar(&logs.format, "log-format", "text", "log format: text or json")

	root.AddCommand(
		newServeCommand(&logs),
		newSeedCommand(),
		newManifestCommand(),
		newVersionCommand(),
	)

	cobra.EnableCommandSorting = false
	root.SetErrPrefix("playground:")
	return root
}

func newLogger(opts logOptions) (*slog.Logger, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(opts.level)); err != nil {
		return nil, fmt.Errorf("unknown log level %q", opts.level)
	}

	handlerOpts := &slog.HandlerOptions{Level: level}
	switch opts.format {
	case "text":
		return slog.New(slog.NewTextHandler(os.Stderr, handlerOpts)), nil
	case "json":
		return slog.New(slog.NewJSONHandler(os.Stderr, handlerOpts)), nil
	default:
		return nil, fmt.Errorf("unknown log format %q: use text or json", opts.format)
	}
}
