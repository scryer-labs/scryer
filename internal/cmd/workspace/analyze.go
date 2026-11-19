package workspace

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/edwardnoaland/scryer/internal/application"
	"github.com/edwardnoaland/scryer/internal/cmd/terminal"
	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/report"
	"github.com/spf13/cobra"
)

func newAnalyzeCommand(service *application.Service) *cobra.Command {
	var operations, output, color string
	var resolve, jsonOutput, verbose bool
	var selected []string
	command := &cobra.Command{
		Use: "analyze [path]", Short: "Analyze configured projects and aggregate their reports",
		Long: "Read projects from .scryer/workspace.yaml. Impact is static by default; --resolve enables engine-resolved facts, native tests and coverage. Each repository selects its own comparison: HEAD^1 to HEAD when clean, or HEAD to an isolated working-tree snapshot when dirty. Workspace --before and --after are not supported. Projects retain independent reports and datasets. Failed or unsupported projects are recorded while other projects continue; incomplete aggregation exits with status 1.",
		Args: cobra.MaximumNArgs(1),
	}
	command.PreRunE = func(cmd *cobra.Command, args []string) error {
		var err error
		selected, err = contract.SelectAnalyzeOperations(operations)
		if err != nil {
			return err
		}
		if resolve && !slices.Contains(selected, contract.ImpactOperation) {
			return fmt.Errorf("--resolve requires impact; coverage already collects runtime evidence")
		}
		if !resolve && len(selected) == 2 {
			return fmt.Errorf("impact,coverage requires --resolve; use coverage alone to collect current-tree coverage")
		}
		if color != "auto" && color != "always" && color != "never" {
			return fmt.Errorf("--color must be auto, always or never")
		}
		if jsonOutput && verbose {
			return fmt.Errorf("--json and --verbose are mutually exclusive")
		}
		if output != "" {
			extension := strings.ToLower(filepath.Ext(output))
			if extension != ".md" && extension != ".html" {
				return fmt.Errorf("-o must use .md or .html; use --json for JSON stdout")
			}
		}
		return nil
	}
	command.RunE = func(cmd *cobra.Command, args []string) error {
		directory := "."
		if len(args) > 0 {
			directory = args[0]
		}
		result, err := service.AnalyzeWorkspace(cmd.Context(), directory, selected, resolve)
		if err != nil {
			return err
		}
		if output != "" {
			var data []byte
			if strings.EqualFold(filepath.Ext(output), ".html") {
				data, err = report.WorkspaceAnalysisHTML(result)
			} else {
				data, err = report.WorkspaceAnalysisMarkdown(result)
			}
			if err != nil {
				return err
			}
			if err = report.WriteFile(output, data); err != nil {
				return err
			}
			fmt.Fprintln(cmd.ErrOrStderr(), "scryer: report written to", output)
		}
		if jsonOutput {
			compact, _ := cmd.Flags().GetBool("compact")
			data, err := report.JSONOutput(result, compact)
			if err != nil {
				return err
			}
			if _, err = cmd.OutOrStdout().Write(data); err != nil {
				return err
			}
		} else if err = report.WorkspaceAnalysisTerminal(result, cmd.OutOrStdout(), verbose, terminal.UseColor(color, cmd.OutOrStdout())); err != nil {
			return err
		}
		if result.Status != "collected" {
			return fmt.Errorf("workspace analysis %s: %d failed, %d unsupported; inspect project results", result.Status, result.Counts["failed"], result.Counts["unsupported"])
		}
		return nil
	}
	command.Flags().StringVar(&operations, "operations", "impact", "Comma-separated operations: impact, coverage")
	command.Flags().BoolVar(&resolve, "resolve", false, "Use resolved project scan facts, native tests and after-snapshot coverage for impact")
	command.Flags().BoolVar(&jsonOutput, "json", false, "Print complete project reports as JSON")
	command.Flags().BoolVar(&verbose, "verbose", false, "Include detailed project graphs and evidence in terminal output")
	command.Flags().StringVar(&color, "color", "auto", "auto, always or never")
	command.Flags().StringVarP(&output, "output", "o", "", "Save a Markdown or offline HTML workspace report")
	return command
}
