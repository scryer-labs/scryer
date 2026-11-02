package engine

import (
	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "list", Short: "List persisted engine installations", Args: cobra.NoArgs}
	bindJSON(command, &jsonOutput)
	command.RunE = func(cmd *cobra.Command, args []string) error {
		store, err := engine.DefaultStore()
		if err != nil {
			return err
		}
		installed, err := store.List(cmd.Context())
		if err != nil {
			return err
		}
		return present(cmd, jsonOutput, result{Operation: "list", Status: "installed", Persisted: true, Engines: installed})
	}
	return command
}
