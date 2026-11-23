// Package configinit shares CLI initialization behavior between scope commands.
package configinit

import (
	"fmt"

	"github.com/edwardnoaland/scryer/internal/configinit"
	"github.com/edwardnoaland/scryer/internal/report"
	"github.com/spf13/cobra"
)

func NewCommand(scope configinit.Scope) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "init [path]", Short: fmt.Sprintf("Create a default %s configuration", scope),
		Long: fmt.Sprintf("Create .scryer/%s.yaml with English configuration comments. Path defaults to the current directory and must already exist. Existing paths are preserved unchanged. This command does not discover projects, scan files, install engines or initialize engine settings.", scope),
		Args: cobra.MaximumNArgs(1),
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "Print the initialization result as JSON")
	command.RunE = func(cmd *cobra.Command, args []string) error {
		directory := "."
		if len(args) > 0 {
			directory = args[0]
		}
		result, err := configinit.Initialize(cmd.Context(), directory, scope)
		if err != nil {
			return err
		}
		if jsonOutput {
			compact, _ := cmd.Flags().GetBool("compact")
			data, err := report.JSONOutput(result, compact)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(data)
			return err
		}
		if result.Status == "created" {
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Created "+result.Path)
		} else {
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Already exists; unchanged: "+result.Path)
		}
		return err
	}
	return command
}
