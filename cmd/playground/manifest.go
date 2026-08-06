package main

import (
	"encoding/json"
	"io"

	"github.com/spf13/cobra"

	"github.com/heyrmi/testground/internal/build"
	"github.com/heyrmi/testground/internal/playground"
	"github.com/heyrmi/testground/internal/rng"
)

func newManifestCommand() *cobra.Command {
	var seed uint64

	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Print the challenge manifest without starting a server",
		Long: `Print the challenge manifest without starting a server.

This is the same document served at /api/challenges. Committing its output and
diffing it in CI is how a project detects that a page's contract moved.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			registry, err := playground.Registry()
			if err != nil {
				return err
			}
			return printJSON(cmd.OutOrStdout(),
				registry.Manifest(build.Current().Version, "cli", seed))
		},
	}

	cmd.Flags().Uint64Var(&seed, "seed", rng.DefaultSeed, "seed to report in the manifest")
	return cmd
}

func printJSON(w io.Writer, body any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(body)
}
