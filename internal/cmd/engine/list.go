package engine

import "github.com/spf13/cobra"

func newListCommand() *cobra.Command {
	var options mockOptions
	command := &cobra.Command{
		Use:   "list",
		Short: "Describe an explicit mock engine (no installed inventory)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return options.run(cmd, "list", nil)
		},
	}
	options.bind(command)
	return command
}
