// Package engine owns engine-management commands.
package engine

import "github.com/spf13/cobra"

func New() *cobra.Command {
	command := &cobra.Command{
		Use:   "engine",
		Short: "Manage local language engine installations",
		Long:  "Install complete local engine packages, list installed engines and uninstall owned packages.",
	}
	command.AddCommand(newListCommand(), newInstallCommand(), newUninstallCommand())
	return command
}
