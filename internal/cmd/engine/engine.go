// Package engine owns engine-management commands.
package engine

import "github.com/spf13/cobra"

func New() *cobra.Command {
	command := &cobra.Command{
		Use:   "engine",
		Short: "Manage language engines (non-persistent mock)",
		Long:  "Probe an explicit mock engine process for list/install/uninstall. Each command requires --executable; no installed inventory, global configuration or package changes are persisted.",
	}
	command.AddCommand(newListCommand(), newInstallCommand(), newUninstallCommand())
	return command
}
