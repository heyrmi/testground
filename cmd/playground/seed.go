package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/heyrmi/testground/internal/session"
)

func newSeedCommand() *cobra.Command {
	var (
		addr    string
		name    string
		asJSON  bool
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "seed [value]",
		Short: "Read or change the seed of a running playground",
		Long: `Read or change the seed of a running playground.

With no argument this reports the seed a session is currently generating from.
With one it reseeds that session, which also discards every page's state, since
content derived from the old seed would no longer match what the pages claim.

The session is named rather than implicit, so this can be pointed at one
parallel worker's copy of the playground without disturbing the others.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := &http.Client{Timeout: timeout}
			base := "http://" + addr + "/api/control"

			var (
				response *http.Response
				err      error
			)
			if len(args) == 0 {
				response, err = call(cmd, client, http.MethodGet, base+"/state", name, nil)
			} else {
				seed, parseErr := strconv.ParseUint(args[0], 10, 64)
				if parseErr != nil {
					return fmt.Errorf("seed must be a non-negative whole number, got %q", args[0])
				}
				body, _ := json.Marshal(map[string]uint64{"seed": seed})
				response, err = call(cmd, client, http.MethodPost, base+"/seed", name, body)
			}
			if err != nil {
				return fmt.Errorf("is a playground running on %s? %w", addr, err)
			}
			defer response.Body.Close()

			payload, err := io.ReadAll(response.Body)
			if err != nil {
				return err
			}
			if response.StatusCode != http.StatusOK {
				return fmt.Errorf("the playground answered %d: %s", response.StatusCode, bytes.TrimSpace(payload))
			}

			if asJSON {
				_, err := cmd.OutOrStdout().Write(payload)
				return err
			}

			var state struct {
				Session string `json:"session"`
				Seed    uint64 `json:"seed"`
			}
			if err := json.Unmarshal(payload, &state); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "session %s is on seed %d\n", state.Session, state.Seed)
			return err
		},
	}

	cmd.Flags().StringVarP(&addr, "addr", "a", "127.0.0.1:7373", "address of the running playground")
	cmd.Flags().StringVar(&name, "session", "cli", "which session to read or reseed")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the full state dump instead of a summary")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "how long to wait for the playground")

	return cmd
}

func call(cmd *cobra.Command, client *http.Client, method, url, name string, body []byte) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	request, err := http.NewRequestWithContext(cmd.Context(), method, url, reader)
	if err != nil {
		return nil, err
	}
	request.Header.Set(session.Header, name)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	return client.Do(request)
}
