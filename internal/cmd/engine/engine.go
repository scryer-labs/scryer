// Package engine owns engine-management commands.
package engine

import "github.com/spf13/cobra"

func New() *cobra.Command {
	command := &cobra.Command{
		Use:   "engine",
		Short: "Manage language engine installations",
		Long:  "Install and upgrade engine releases, install local packages, and manage installed engines.",
	}
	command.AddCommand(newListCommand(), newInstallCommand(), newUninstallCommand(), newUpgradeCommand())
	return command
}
