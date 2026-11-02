package engine

import "github.com/spf13/cobra"

func newInstallCommand() *cobra.Command {
	var options mockOptions
	command := &cobra.Command{
		Use:   "install <engine> [engines...]",
		Short: "Verify a mock install connection without persisting state",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return options.run(cmd, "install", args)
		},
	}
	options.bind(command)
	return command
}
