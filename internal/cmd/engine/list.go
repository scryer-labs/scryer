package engine

import (
	"fmt"
	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	var jsonOutput bool
	var color string
	command := &cobra.Command{Use: "list", Short: "List persisted engine installations", Args: cobra.NoArgs}
	bindJSON(command, &jsonOutput)
	command.Flags().StringVar(&color, "color", "auto", "auto, always or never")
	command.PreRunE = func(cmd *cobra.Command, args []string) error {
		if color != "auto" && color != "always" && color != "never" {
			return fmt.Errorf("--color must be auto, always or never")
		}
		return nil
	}
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
