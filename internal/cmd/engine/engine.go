// Package engine owns engine-management commands.
package engine

import "github.com/spf13/cobra"

func New() *cobra.Command {
	command := &cobra.Command{
		Use:   "engine",
		Short: "Manage language engines (command placeholders)",
		Long:  "Engine-management entry points. Commands currently print placeholders only; they do not list installed engines or change installations.",
	}
	command.AddCommand(newListCommand(), newInstallCommand(), newUninstallCommand())
	return command
}
