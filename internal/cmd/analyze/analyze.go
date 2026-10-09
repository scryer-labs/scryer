// Package analyze owns public ref-analysis arguments and output selection.
package analyze

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/edwardnoaland/scryer/internal/application"
	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/report"
	"github.com/spf13/cobra"
)

type ExecutionError struct{ Status string }

func (e ExecutionError) Error() string { return "after test command: " + e.Status }
func New(service *application.Service) *cobra.Command {
	var request contract.AnalyzeRequest
	var stack, output string
	var jsonOutput, verbose bool
	command := &cobra.Command{Use: "analyze [path] --before <ref> --after <ref|.>", Short: "Analyze changes and existing test evidence", Args: cobra.MaximumNArgs(1)}
	command.PreRunE = func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(request.Before) == "" || strings.TrimSpace(request.After) == "" {
			return fmt.Errorf("--before and --after are required")
		}
		if request.SkipTests && request.TestCommand != "" {
			return fmt.Errorf("--skip-tests and --test-command are mutually exclusive")
		}
		if jsonOutput && verbose {
			return fmt.Errorf("--json and --verbose are mutually exclusive")
		}
		if cmd.Flags().Changed("test-command") && strings.TrimSpace(request.TestCommand) == "" {
			return fmt.Errorf("--test-command must not be blank")
		}
		if request.BuildTool != "" && request.BuildTool != "maven" && request.BuildTool != "gradle" {
			return fmt.Errorf("--build-tool must be maven or gradle")
		}
		if request.TestCommand != "" && strings.TrimSpace(request.TestCommand) == "" {
			return fmt.Errorf("--test-command must not be blank")
		}
		if strings.ContainsRune(request.TestCommand, 0) {
			return fmt.Errorf("--test-command must not contain NUL")
		}
		if output != "" {
			ext := strings.ToLower(filepath.Ext(output))
			if ext != ".json" && ext != ".md" && ext != ".html" {
				return fmt.Errorf("-o must use .json, .md or .html")
			}
		}
		return nil
	}
	command.RunE = func(cmd *cobra.Command, args []string) error {
		directory, err := os.Getwd()
		if err != nil {
			return err
		}
		if len(args) == 1 {
			directory = args[0]
		}
		document, err := service.Analyze(cmd.Context(), stack, directory, request)
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
			if err = report.Terminal(document, cmd.OutOrStdout(), false, false, verbose, false); err != nil {
				return err
			}
		}
		if output != "" {
			var data []byte
			switch strings.ToLower(filepath.Ext(output)) {
			case ".json":
				data, err = report.JSON(document)
			case ".md":
				data, err = report.Markdown(document, false)
			case ".html":
				data, err = report.HTML(document)
			}
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
			fmt.Fprintln(cmd.ErrOrStderr(), "scryer: Report saved to "+absolute)
		}
		var result struct {
			Execution struct {
				Status string `json:"status"`
			} `json:"execution"`
		}
		if err = json.Unmarshal(document.Data, &result); err != nil {
			return err
		}
		if result.Execution.Status != "SUCCEEDED" && result.Execution.Status != "SKIPPED" {
			return ExecutionError{result.Execution.Status}
		}
		return nil
	}
	flags := command.Flags()
	flags.StringVar(&stack, "stack", "java", "Technology stack (currently java)")
	flags.StringVar(&request.BuildTool, "build-tool", "", "Select maven or gradle for both models and tests")
	flags.StringVar(&request.Before, "before", "", "Before Git commit/ref")
	flags.StringVar(&request.After, "after", "", "After Git commit/ref, or . for the working tree")
	flags.BoolVar(&request.SkipTests, "skip-tests", false, "Skip build model and test execution")
	flags.StringVar(&request.TestCommand, "test-command", "", "Explicit after test lifecycle")
	flags.BoolVar(&jsonOutput, "json", false, "Print report JSON")
	flags.BoolVar(&verbose, "verbose", false, "Show full graphs and evidence")
	flags.StringVarP(&output, "output", "o", "", "Save JSON, Markdown or HTML report")
	command.MarkFlagsMutuallyExclusive("skip-tests", "test-command")
	command.MarkFlagsMutuallyExclusive("json", "verbose")
	return command
}
