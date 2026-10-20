package analyze

import (
	"fmt"
	"github.com/spf13/cobra"
)

func newWorkspaceCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "workspace",
		Short: "Analyze dependencies between workspace projects",
		Args:  cobra.NoArgs,
		RunE:  runWorkspaceAnalysis,
	}
}

func runWorkspaceAnalysis(cmd *cobra.Command, args []string) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "Analyzing workspace dependencies (not implemented yet)")
	return err
}
