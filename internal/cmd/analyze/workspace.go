package analyze

import (
	"fmt"

	"github.com/edwardnoaland/scryer/internal/workspace"
	"github.com/spf13/cobra"
)

func newWorkspaceCommand() *cobra.Command {
	var projects, dependencies string
	command := &cobra.Command{
		Use:   "workspace --projects <projects.csv> --dependencies <dependencies.csv>",
		Short: "Read workspace project and dependency facts from CSV",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkspaceAnalysis(cmd, projects, dependencies)
		},
	}
	command.Flags().StringVar(&projects, "projects", "", "Projects CSV (artifactId, projectId or projectName)")
	command.Flags().StringVar(&dependencies, "dependencies", "", "Dependencies CSV (source,target or Moderne coordinates)")
	command.MarkFlagRequired("projects")
	command.MarkFlagRequired("dependencies")
	return command
}

func runWorkspaceAnalysis(cmd *cobra.Command, projects, dependencies string) error {
	facts, err := workspace.ReadCSV(projects, dependencies)
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
