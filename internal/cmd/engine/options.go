package engine

import (
	"encoding/json"
	"fmt"

	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/spf13/cobra"
)

type result struct {
	Operation      string                `json:"operation"`
	Status         string                `json:"status"`
	Persisted      bool                  `json:"persisted"`
	Engines        []engine.Installation `json:"engines"`
	engineStatuses []string
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
	if response.Operation == "list" {
		return presentList(command, response.Engines)
	}
	for index, installed := range response.Engines {
		status := response.Status
		if index < len(response.engineStatuses) {
			status = response.engineStatuses[index]
		}
		entrypoint := installed.Library
		if entrypoint == "" {
			entrypoint = installed.Executable
		}
		if _, err := fmt.Fprintf(command.OutOrStdout(), "%s: %s %s (%s)\n", status, installed.Engine.ID, installed.Engine.Version, entrypoint); err != nil {
			return err
		}

	}
	return nil
}
