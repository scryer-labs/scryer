// Package refactorplan exposes deterministic suggestions, not execution orchestration.
package refactorplan

import (
	"fmt"
	"strings"

	"github.com/edwardnoaland/scryer/internal/report"
	"github.com/edwardnoaland/scryer/internal/workspace"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	command := &cobra.Command{Use: "refactor-plan", Short: "Generate dependency-based phase suggestions and platform plan templates"}
	var projects, dependencies, output string
	var jsonOutput bool
	generate := &cobra.Command{Use: "generate", Short: "Suggest phases from workspace CSV facts; no code or recipes are executed", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if output != "" {
			if _, err := report.WorkspaceFormat(output); err != nil {
				return err
			}
		}
		facts, err := workspace.ReadCSV(projects, dependencies)
		if err != nil {
			return err
		}
		doc := workspace.SuggestPhases(facts)
		if jsonOutput {
			data, err := report.Workspace(doc, "json")
			if err != nil {
				return err
			}
			if _, err = cmd.OutOrStdout().Write(data); err != nil {
				return err
			}
		}
		if output != "" {
			return report.SaveWorkspace(doc, output, cmd.ErrOrStderr())
		}
		if jsonOutput {
			return nil
		}
		for _, phase := range doc.Suggestion.Phases {
			if _, err = fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", phase.ID, strings.Join(phase.Projects, ", ")); err != nil {
				return err
			}
		}
		if _, err = fmt.Fprintf(cmd.OutOrStdout(), "Cycle members: %s\nBlocked (cycles and downstream): %s\nSuggestions only; all internal declared edges constrain order.\n", strings.Join(doc.Suggestion.Cyclic, ", "), strings.Join(doc.Suggestion.Blocked, ", ")); err != nil {
			return err
		}
		return nil
	}}
	generate.Flags().StringVar(&projects, "projects", "", "Projects CSV")
	generate.Flags().StringVar(&dependencies, "dependencies", "", "Dependencies CSV")
	generate.Flags().StringVarP(&output, "output", "o", "", "Save suggestions (.json or .html)")
	generate.Flags().BoolVar(&jsonOutput, "json", false, "Print suggestion JSON to stdout")
	generate.MarkFlagRequired("projects")
	generate.MarkFlagRequired("dependencies")
	command.AddCommand(generate)
	return command
}
