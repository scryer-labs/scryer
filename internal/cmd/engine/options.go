package engine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/spf13/cobra"
)

type result struct {
	Operation string                `json:"operation"`
	Status    string                `json:"status"`
	Persisted bool                  `json:"persisted"`
	Engines   []engine.Installation `json:"engines"`
}

func bindJSON(command *cobra.Command, jsonOutput *bool) {
	command.Flags().BoolVar(jsonOutput, "json", false, "Print structured engine-management results")
}

func present(command *cobra.Command, jsonOutput bool, response result) error {
	if response.Engines == nil {
		response.Engines = []engine.Installation{}
	}
	if jsonOutput {
		return json.NewEncoder(command.OutOrStdout()).Encode(response)
	}
	if len(response.Engines) == 0 {
		_, err := fmt.Fprintln(command.OutOrStdout(), "No engines installed.")
		return err
	}
	for _, installed := range response.Engines {
		entrypoint := installed.Library
		if entrypoint == "" {
			entrypoint = installed.Executable
		}
		if _, err := fmt.Fprintf(command.OutOrStdout(), "%s: %s %s (%s)\n", response.Status, installed.Engine.ID, installed.Engine.Version, entrypoint); err != nil {
			return err
		}
		if response.Operation == "list" && installed.Capabilities != nil {
			if _, err := fmt.Fprintf(command.OutOrStdout(), "  Operations: %s\n  Scan facts: %s\n", strings.Join(installed.Capabilities.Operations, ", "), strings.Join(installed.Capabilities.ScanFacts, ", ")); err != nil {
				return err
			}
		}
	}
	return nil
}
