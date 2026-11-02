package engine

import (
	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/spf13/cobra"
)

func newUninstallCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{Use: "uninstall <engine>", Short: "Remove an owned engine installation", Args: cobra.ExactArgs(1)}
	bindJSON(command, &jsonOutput)
	command.RunE = func(cmd *cobra.Command, args []string) error {
		store, err := engine.DefaultStore()
		if err != nil {
			return err
		}
		installed, err := store.Uninstall(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		return present(cmd, jsonOutput, result{Operation: "uninstall", Status: "uninstalled", Persisted: true, Engines: []engine.Installation{installed}})
	}
	return command
}
