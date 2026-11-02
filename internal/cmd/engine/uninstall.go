package engine

import "github.com/spf13/cobra"

func newUninstallCommand() *cobra.Command {
	var options mockOptions
	command := &cobra.Command{
		Use:   "uninstall <engine>",
		Short: "Verify a mock uninstall connection without deleting anything",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return options.run(cmd, "uninstall", args)
		},
	}
	options.bind(command)
	return command
}
