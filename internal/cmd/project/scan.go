package project

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

func newScanCommand(service *application.Service) *cobra.Command {
	var request contract.ScanRequest
	var output, color string
	var jsonOutput, resolve bool
	var facts string
	var useCore, collectStack, collectLOC bool
	var engineFacts []string
	command := &cobra.Command{Use: "scan [path]", Short: "Collect project facts",
		Long: "Collect stack and line counts by default. Select facts with --facts. Explicit Java scan options select the Java engine. Path defaults to the current directory.", Args: cobra.MaximumNArgs(1)}
	command.PreRunE = func(cmd *cobra.Command, args []string) error {
		javaFlag := ""
		for _, flag := range []string{"static", "remote-list", "build-tool"} {
			if cmd.Flags().Changed(flag) {
				javaFlag = flag
				break
			}
		}
		useCore = javaFlag == ""
		collectStack, collectLOC = true, true
		engineFacts = nil
		if cmd.Flags().Changed("facts") {
			if javaFlag != "" {
				return fmt.Errorf("--%s cannot be combined with --facts", javaFlag)
			}
			selection, err := contract.SelectScanFacts(facts)
			if err != nil {
				return err
			}
			collectStack, collectLOC, engineFacts = selection.Stack, selection.LOC, selection.EngineFacts
		}
		if resolve && !slices.ContainsFunc(engineFacts, contract.IsResolvableScanFact) {
			return fmt.Errorf("--resolve requires dependencies, files or call-graph in --facts")
		}
		if request.CompileCommands != "" && (!resolve || !slices.ContainsFunc(engineFacts, func(fact string) bool { return fact == "files" || fact == "call-graph" })) {
			return fmt.Errorf("--compile-commands requires --resolve and files or call-graph in --facts")
		}
		if request.BuildTool != "" && request.BuildTool != "maven" && request.BuildTool != "gradle" {
			return fmt.Errorf("--build-tool must be maven or gradle")
		}
		if output != "" {
			switch strings.ToLower(filepath.Ext(output)) {
			case "", ".md":
			default:
				return fmt.Errorf("-o must use .md")
			}
		}
		if color != "auto" && color != "always" && color != "never" {
			return fmt.Errorf("--color must be auto, always or never")
		}
		return nil
	}
	command.RunE = func(cmd *cobra.Command, args []string) error {
		request.Repository = "."
		if len(args) == 1 {
			request.Repository = args[0]
		}
		var document contract.Document
		var err error
		if useCore {
			document, err = service.ScanFacts(cmd.Context(), request.Repository, collectStack, collectLOC, engineFacts, resolve, request.CompileCommands)
		} else {
			document, err = service.Scan(cmd.Context(), "java", request)
		}
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
			if err = report.Terminal(document, cmd.OutOrStdout(), false, terminal.UseColor(color, cmd.OutOrStdout())); err != nil {
				return err
			}
		}
		if output != "" {
			data, err := report.Markdown(document)
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
	flags.StringVar(&facts, "facts", "", "Collect selected facts (for example stack,loc,dependencies)")
	flags.BoolVar(&resolve, "resolve", false, "Allow build configuration and semantic resolution")
	flags.StringVar(&request.CompileCommands, "compile-commands", "", "Select a C++ compilation database for --resolve")
	flags.StringVar(&request.BuildTool, "build-tool", "", "Select maven or gradle")
	flags.BoolVar(&request.Static, "static", false, "Skip build-model execution")
	flags.BoolVar(&request.RemoteList, "remote-list", false, "Compare dependency and Java/build-tool releases with remote metadata")
	flags.BoolVar(&jsonOutput, "json", false, "Print report JSON")
	flags.StringVar(&color, "color", "auto", "auto, always or never")
	flags.StringVarP(&output, "output", "o", "", "Save a Markdown report")
	return command
}
