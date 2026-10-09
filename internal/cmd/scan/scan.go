// Package scan owns public scan arguments and output choices; analyzers do not see presentation flags.
package scan

import (
	"fmt"
	"path/filepath"

	"github.com/edwardnoaland/scryer/internal/application"
	"github.com/edwardnoaland/scryer/internal/cmd/terminal"
	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/report"
	"github.com/spf13/cobra"
)

func New(service *application.Service) *cobra.Command {
	var request contract.ScanRequest
	var stack, output, color string
	var jsonOutput, dependencies, tree bool
	command := &cobra.Command{Use: "scan <path>", Short: "Collect repository facts", Args: cobra.ExactArgs(1)}
	command.PreRunE = func(cmd *cobra.Command, args []string) error {
		if request.BuildTool != "" && request.BuildTool != "maven" && request.BuildTool != "gradle" {
			return fmt.Errorf("--build-tool must be maven or gradle")
		}
		if color != "auto" && color != "always" && color != "never" {
			return fmt.Errorf("--color must be auto, always or never")
		}
		return nil
	}
	command.RunE = func(cmd *cobra.Command, args []string) error {
		request.Repository = args[0]
		document, err := service.Scan(cmd.Context(), stack, request)
		if err != nil {
			return err
		}
		if jsonOutput {
			data, err := report.JSON(document)
			if err != nil {
				return err
			}
			if _, err = cmd.OutOrStdout().Write(data); err != nil {
				return err
			}
		} else {
			if err = report.Terminal(document, cmd.OutOrStdout(), dependencies, tree, false, terminal.UseColor(color, cmd.OutOrStdout())); err != nil {
				return err
			}
		}
		if output != "" {
			data, err := report.Markdown(document, tree)
			if err != nil {
				return err
			}
			absolute, err := filepath.Abs(output)
			if err != nil {
				return err
			}
			if err = report.WriteFile(absolute, data); err != nil {
				return err
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "scryer: Markdown report written to "+absolute)
		}
		return nil
	}
	flags := command.Flags()
	flags.StringVar(&stack, "stack", "java", "Technology stack (currently java)")
	flags.StringVar(&request.BuildTool, "build-tool", "", "Select maven or gradle")
	flags.BoolVar(&request.Static, "static", false, "Skip build-model execution")
	flags.BoolVar(&request.RemoteList, "remote-list", false, "Compare dependency releases with Maven Central")
	flags.BoolVar(&dependencies, "dependencies", false, "List direct dependencies by module/configuration")
	flags.BoolVar(&tree, "dependency-tree", false, "Show resolved dependency graph")
	flags.BoolVar(&jsonOutput, "json", false, "Print report JSON")
	flags.StringVar(&color, "color", "auto", "auto, always or never")
	flags.StringVarP(&output, "output", "o", "", "Save Markdown report")
	return command
}
