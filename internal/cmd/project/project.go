// Package project owns project-scoped commands and their argument/output choices.
package project

import (
	"github.com/edwardnoaland/scryer/internal/application"
	initcmd "github.com/edwardnoaland/scryer/internal/cmd/configinit"
	"github.com/edwardnoaland/scryer/internal/configinit"
	"github.com/spf13/cobra"
)

func New(service *application.Service) *cobra.Command {
	command := &cobra.Command{Use: "project", Short: "Initialize, scan and analyze a single repository"}
	command.AddCommand(initcmd.NewCommand(configinit.Project), newScanCommand(service), newAnalyzeCommand(service))
	return command
}
