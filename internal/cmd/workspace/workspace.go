// Package workspace owns workspace-scoped commands.
package workspace

import (
	"github.com/edwardnoaland/scryer/internal/application"
	initcmd "github.com/edwardnoaland/scryer/internal/cmd/configinit"
	"github.com/edwardnoaland/scryer/internal/configinit"
	"github.com/spf13/cobra"
)

func New(service *application.Service) *cobra.Command {
	command := &cobra.Command{Use: "workspace", Short: "Initialize, scan and analyze a workspace"}
	command.AddCommand(initcmd.NewCommand(configinit.Workspace), newScanCommand(service), newAnalyzeCommand(service))
	return command
}
