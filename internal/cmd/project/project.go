// Package project owns project-scoped commands and their argument/output choices.
package project

import (
	"github.com/edwardnoaland/scryer/internal/application"
	"github.com/spf13/cobra"
)

func New(service *application.Service) *cobra.Command {
	command := &cobra.Command{Use: "project", Short: "Scan and analyze a single repository"}
	command.AddCommand(newScanCommand(service), newAnalyzeCommand(service))
	return command
}
