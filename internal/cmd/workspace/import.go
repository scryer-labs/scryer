package workspace

import (
	"fmt"

	"github.com/edwardnoaland/scryer/internal/report"
	workspacefacts "github.com/edwardnoaland/scryer/internal/workspace"
	"github.com/spf13/cobra"
)

func newImportCommand() *cobra.Command {
	var projects, dependencies, output string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "import --projects <projects.csv> --dependencies <dependencies.csv>",
		Short: "Read external project/dependency CSV facts (temporary adapter)",
		Long:  "Read and normalize external project/dependency CSV facts, then print or export a report. This temporary adapter does not scan project source, register projects or modify workspace configuration.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "" || jsonOutput {
				facts, err := workspacefacts.ReadCSV(projects, dependencies)
				if err != nil {
					return err
				}
				document := workspacefacts.NewDocument(facts)
				if jsonOutput {
					data, err := report.Workspace(document, "json")
					if err != nil {
						return err
					}
					if _, err = cmd.OutOrStdout().Write(data); err != nil {
						return err
					}
				}
				if output != "" {
					return report.SaveWorkspace(document, output, cmd.ErrOrStderr())
				}
				return nil
			}
			return runWorkspaceAnalysis(cmd, projects, dependencies)
		},
	}
	command.PreRunE = func(cmd *cobra.Command, args []string) error {
		if projects == "" || dependencies == "" {
			return fmt.Errorf("--projects and --dependencies are required")
		}
		if output != "" {
			_, err := report.WorkspaceFormat(output)
			return err
		}
		return nil
	}
	command.Flags().StringVar(&projects, "projects", "", "Projects CSV (artifactId, projectId or projectName)")
	command.Flags().StringVar(&dependencies, "dependencies", "", "Dependencies CSV (source,target or Moderne coordinates)")
	command.Flags().StringVarP(&output, "output", "o", "", "Save workspace report (.json or .html)")
	command.Flags().BoolVar(&jsonOutput, "json", false, "Print workspace JSON to stdout")
	command.MarkFlagRequired("projects")
	command.MarkFlagRequired("dependencies")
	return command
}

func runWorkspaceAnalysis(cmd *cobra.Command, projects, dependencies string) error {
	facts, err := workspacefacts.ReadCSV(projects, dependencies)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	if _, err = fmt.Fprintf(out, "Projects (%d):\n", len(facts.Projects)); err != nil {
		return err
	}
	for _, project := range facts.Projects {
		if _, err = fmt.Fprintf(out, "  %s\n", project.ID); err != nil {
			return err
		}
	}
	if _, err = fmt.Fprintf(out, "Dependencies (%d; consumer -> dependency):\n", len(facts.Dependencies)); err != nil {
		return err
	}
	for _, edge := range facts.Dependencies {
		suffix := ""
		if !edge.Internal {
			suffix = " [outside projects]"
		}
		if _, err = fmt.Fprintf(out, "  %s -> %s%s\n", edge.Source, edge.Target, suffix); err != nil {
			return err
		}
	}
	return nil
}
