package engine

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall <engine>",
		Short: "Uninstall an engine (not implemented yet)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "engine uninstall: not implemented yet (requested: %s)\n", args[0])
			return err
		},
	}
}
