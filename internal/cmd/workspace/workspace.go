// Package workspace owns workspace-scoped commands.
package workspace

import (
	"github.com/edwardnoaland/scryer/internal/application"
	"github.com/spf13/cobra"
)

func New(service *application.Service) *cobra.Command {
	command := &cobra.Command{Use: "workspace", Short: "Work with workspace facts"}
	command.AddCommand(newScanCommand(service))
	return command
}
