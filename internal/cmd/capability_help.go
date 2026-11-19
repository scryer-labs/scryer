package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/edwardnoaland/scryer/internal/cmd/capabilities"
	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/spf13/cobra"
)

// Keep the command grammar stable; append installed capability information only
// when scope help is requested. Inventory reads never load an engine or scan code.
func installCapabilityHelp(root *cobra.Command) {
	standardHelp := root.HelpFunc()
	root.SetHelpFunc(func(command *cobra.Command, args []string) {
		standardHelp(command, args)
		scope := command
		for scope.Parent() != nil && scope.Parent() != root {
			scope = scope.Parent()
		}
		if scope.Name() != "project" && scope.Name() != "workspace" {
			return
		}
		output := command.OutOrStdout()
		fmt.Fprintln(output, "\nInstalled engine capabilities (inventory, not project detection):")
		parent := command.Context()
		if parent == nil {
			parent = context.Background()
		}
		ctx, cancel := context.WithTimeout(parent, 250*time.Millisecond)
		defer cancel()
		store, err := engine.DefaultStore()
		var installed []engine.Installation
		if err == nil {
			installed, err = store.List(ctx)
		}
		if err != nil {
			fmt.Fprintln(output, "  Inventory unavailable; command help above remains valid.")
			return
		}
		scan := command == scope || command.Name() == "scan"
		analyze := command == scope || command.Name() == "analyze"
		if scan {
			fmt.Fprintln(output, "  Core scan facts: stack, loc (no engine required)")
		}
		if len(installed) == 0 {
			fmt.Fprintln(output, "  No engines installed. Use scryer engine install <id>.")
		}
		for _, entry := range installed {
			fmt.Fprintf(output, "  %s %s\n", entry.Engine.ID, entry.Engine.Version)
			if entry.Capabilities == nil {
				fmt.Fprintln(output, "    Capabilities unavailable")
				continue
			}
			if scan {
				facts := strings.Join(entry.Capabilities.ScanFacts, ", ")
				if facts == "" {
					facts = "none"
				}
				fmt.Fprintln(output, "    Scan facts:", facts)
				if len(entry.Capabilities.ResolveFacts) > 0 {
					fmt.Fprintln(output, "    Resolve facts:", strings.Join(entry.Capabilities.ResolveFacts, ", "))
				} else if entry.Capabilities.ResolveDependencies {
					fmt.Fprintln(output, "    Resolve facts: dependencies")
				}
			}
			if analyze {
				operations := strings.Join(capabilities.AnalyzeOperations(entry.Capabilities), ", ")
				if operations == "" {
					operations = "none"
				}
				fmt.Fprintln(output, "    Analyze operations:", operations)
			}
		}
		if scope.Name() == "workspace" {
			fmt.Fprintln(output, "  Workspace analyzes projects independently; reports are aggregated without cross-project impact propagation.")
		}
		fmt.Fprintln(output, "  Capabilities reflect installed versions. Use scryer engine upgrade <id> to update.")
	})
}
