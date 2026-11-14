package engine

import (
	"errors"
	"fmt"
	"io"

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
	var progress io.Writer = io.Discard
	if !jsonOutput {
		progress = cmd.ErrOrStderr()
	}
	for _, id := range args {
		fmt.Fprintf(progress, "Checking releases for %s…\n", id)
		installed, changed, err := store.Fetch(cmd.Context(), id, upgrade, cmd.ErrOrStderr(), progress)
		if err != nil {
			err = fmt.Errorf("%s %s: %w", response.Operation, id, err)
			if len(response.Engines) > 0 {
				response.Status = "partial"
				return errors.Join(err, present(cmd, jsonOutput, response))
			}
			return err
		}
		status := "installed"
		if upgrade {
			status = "up-to-date"
			if changed {
				status = "upgraded"
				response.Status = status
			}
		}
		response.Engines = append(response.Engines, installed)
		response.engineStatuses = append(response.engineStatuses, status)
	}
	return present(cmd, jsonOutput, response)
}
