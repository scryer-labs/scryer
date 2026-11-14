package engine

import (
	"errors"
	"fmt"

	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/spf13/cobra"
)

func newUpgradeCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "upgrade <engine> [engines...]", Short: "Upgrade installed engines to compatible releases", Args: cobra.MinimumNArgs(1)}
	bindJSON(command, &jsonOutput)
	command.RunE = func(cmd *cobra.Command, args []string) error { return fetch(cmd, args, jsonOutput, true) }
	return command
}

func fetch(cmd *cobra.Command, args []string, jsonOutput, upgrade bool) error {
	store, err := engine.DefaultStore()
	if err != nil {
		return err
	}
	response := result{Operation: "install", Status: "installed", Persisted: true}
	if upgrade {
		response.Operation, response.Status = "upgrade", "up-to-date"
	}
	seen := map[string]bool{}
	for _, id := range args {
		if seen[id] {
			return fmt.Errorf("duplicate engine %q", id)
		}
		seen[id] = true
	}
	for _, id := range args {
		installed, changed, err := store.Fetch(cmd.Context(), id, upgrade, cmd.ErrOrStderr())
		if err != nil {
			if len(response.Engines) > 0 {
				response.Status = "partial"
				return errors.Join(err, present(cmd, jsonOutput, response))
			}
			return err
		}
		response.Engines = append(response.Engines, installed)
		if upgrade && changed {
			response.Status = "upgraded"
		}
	}
	return present(cmd, jsonOutput, response)
}
