package engine

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install <engine> [engines...]",
		Short: "Install one or more engines (not implemented yet)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "engine install: not implemented yet (requested: %s)\n", strings.Join(args, ", "))
			return err
		},
	}
}
