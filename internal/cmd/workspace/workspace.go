// Package workspace owns workspace-scoped commands and external fact input.
package workspace

import "github.com/spf13/cobra"

func New() *cobra.Command {
	command := &cobra.Command{Use: "workspace", Short: "Work with workspace facts"}
	command.AddCommand(newImportCommand())
	return command
}
