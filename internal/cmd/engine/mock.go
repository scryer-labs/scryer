package engine

import (
	"encoding/json"
	"fmt"

	"github.com/edwardnoaland/scryer/internal/application"
	"github.com/spf13/cobra"
)

type mockOptions struct {
	executable string
	json       bool
}

type mockResult struct {
	Operation  string                     `json:"operation"`
	Status     string                     `json:"status"`
	Persisted  bool                       `json:"persisted"`
	Executable string                     `json:"executable"`
	Engine     application.EngineIdentity `json:"engine"`
}

func (options *mockOptions) bind(command *cobra.Command) {
	command.Flags().StringVar(&options.executable, "executable", "", "Explicit mock engine executable; no installation is stored")
	command.Flags().BoolVar(&options.json, "json", false, "Print a JSON mock receipt")
	command.PreRunE = func(cmd *cobra.Command, args []string) error {
		if options.executable == "" {
			return fmt.Errorf("--executable is required for this non-persistent mock connection")
		}
		return nil
	}
}

func (options *mockOptions) run(cmd *cobra.Command, operation string, requested []string) error {
	identity, executable, err := application.DescribeEngine(cmd.Context(), options.executable, cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	for _, id := range requested {
		if id != identity.ID {
			return fmt.Errorf("engine executable identifies as %q, not requested %q", identity.ID, id)
		}
	}
	result := mockResult{Operation: operation, Status: "mock", Persisted: false, Executable: executable, Engine: identity}
	if options.json {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Mock %s: %s (%s) from %s via %s; no installation state persisted\n", operation, identity.ID, identity.Version, identity.Source, executable)
	return err
}
