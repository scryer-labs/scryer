package project

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/edwardnoaland/scryer/internal/application"
	"github.com/edwardnoaland/scryer/internal/cmd/terminal"
	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/report"
	"github.com/spf13/cobra"
)

type ExecutionError struct{ Status string }

func (e ExecutionError) Error() string { return "after test command: " + e.Status }
func newAnalyzeCommand(service *application.Service) *cobra.Command {
	var request contract.AnalyzeRequest
	var stack, output, color, operations string
	var coverage contract.CoverageRequest
	var testBinaryArgs, testTargetArgs []string
	var jsonOutput, verbose, runTests bool
	var selected analyzeOperations
	command := &cobra.Command{
		Use: "analyze [path]", Short: "Analyze change impact or collect coverage evidence",
		Long: "Analyze one project. The default operation is impact. A clean worktree compares HEAD^1 to HEAD; a changed worktree compares HEAD to an isolated snapshot. Pass both --before and --after to override the comparison. Impact uses static scan facts by default. --resolve enables engine-resolved scan facts and native tests; --test-target or --test-binary implies it. Multi-stack projects dispatch to all matching capable engines. CMake selections are sent only to supporting engines; other engines use their native defaults. Failures stop dynamic impact. Coverage alone builds and tests the current project. Use --test-target for CMake targets, --test-binary for paths relative to the temporary build directory, or --coverage-file to import existing evidence.",
		Args: cobra.MaximumNArgs(1),
	}
	command.PreRunE = func(cmd *cobra.Command, args []string) error {
		if color != "auto" && color != "always" && color != "never" {
			return fmt.Errorf("--color must be auto, always or never")
		}
		var err error
		selected, err = selectAnalyzeOperations(operations)
		if err != nil {
			return err
		}
		if err := selected.validate(cmd, request, coverage.Artifact, runTests); err != nil {
			return err
		}
		err = selected.selectCoverageTests(&coverage, testBinaryArgs, testTargetArgs)
		if err != nil {
			return err
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
		if strings.ContainsRune(request.TestCommand, 0) {
			return fmt.Errorf("--test-command must not contain NUL")
		}
		if output != "" {
			ext := strings.ToLower(filepath.Ext(output))
			if ext != ".md" && ext != ".html" {
				return fmt.Errorf("-o must use .md or .html; use --json to write JSON to stdout")
			}
		}
		return nil
	}
	command.RunE = func(cmd *cobra.Command, args []string) error {
		directory := "."
		if len(args) == 1 {
			directory = args[0]
		}
		var document contract.Document
		var err error
		if selected.impact {
			request.Coverage, request.IncludeCoverage, request.RunTests = coverage, selected.coverage, runTests
			document, err = service.Analyze(cmd.Context(), stack, directory, request)
		} else {
			document, err = service.CollectCoverage(cmd.Context(), directory, stack, coverage)
		}
		if err != nil {
			return err
		}
		return presentAnalysis(cmd, document, output, color, jsonOutput, verbose)
	}

	flags := command.Flags()
	flags.StringVar(&color, "color", "auto", "auto, always or never")
	flags.StringVar(&stack, "stack", "", "Restrict analysis to one engine (default: all matching capable engines)")
	flags.StringVar(&operations, "operations", "impact", "Comma-separated analysis operations: impact, coverage")
	flags.StringVar(&coverage.Artifact, "coverage-file", "", "Import engine coverage instead of testing (LLVM JSON, Go profile or Istanbul JSON; mixed projects require --stack)")
	flags.StringArrayVar(&testBinaryArgs, "test-binary", nil, "Executable paths relative to the temporary coverage build directory (repeat or comma-separate)")
	flags.StringArrayVar(&testTargetArgs, "test-target", nil, "CMake executable target names for coverage (repeat or comma-separate)")
	flags.BoolVar(&request.Resolve, "resolve", false, "Use engine-resolved scan facts and native test coverage for impact")
	flags.BoolVar(&runTests, "run-tests", false, "Run native tests with resolved impact (also supports the legacy Java lifecycle)")
	flags.StringVar(&request.BuildTool, "build-tool", "", "Select maven or gradle for both models and tests")
	flags.StringVar(&request.Before, "before", "", "Before Git commit/ref")
	flags.StringVar(&request.After, "after", "", "After Git commit/ref, or . for the working tree")
	flags.BoolVar(&request.SkipTests, "skip-tests", false, "Skip build model and test execution")
	flags.StringVar(&request.TestCommand, "test-command", "", "Explicit after test lifecycle")
	flags.BoolVar(&jsonOutput, "json", false, "Print report JSON")
	flags.BoolVar(&verbose, "verbose", false, "Show full graphs and evidence")
	flags.StringVarP(&output, "output", "o", "", "Save a Markdown or HTML report")
	command.MarkFlagsMutuallyExclusive("skip-tests", "test-command")
	command.MarkFlagsMutuallyExclusive("json", "verbose")
	return command
}

func presentAnalysis(cmd *cobra.Command, document contract.Document, output, color string, jsonOutput, verbose bool) error {
	var err error
	compact, _ := cmd.Flags().GetBool("compact")
	if jsonOutput {
		data, err := report.JSONOutput(document.Data, compact)
		if err != nil {
			return err
		}
		if _, err = cmd.OutOrStdout().Write(data); err != nil {
			return err
		}
	} else {
		if err = report.Terminal(document, cmd.OutOrStdout(), verbose, terminal.UseColor(color, cmd.OutOrStdout())); err != nil {
			return err
		}
	}
	if output != "" {
		var data []byte
		switch strings.ToLower(filepath.Ext(output)) {
		case ".md":
			data, err = report.Markdown(document)
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
	if document.Operation == contract.CoverageOperation {
		return nil
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
