package engine

import (
	"errors"
	"fmt"

	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/spf13/cobra"
)

func newInstallCommand() *cobra.Command {
	var sources []string
	var executable string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "install <engine> [engines...]",
		Short: "Install complete local engine packages",
		Args:  cobra.MinimumNArgs(1),
	}
	command.Flags().StringSliceVar(&sources, "from", nil, "Local package directories (repeat for multiple engines)")
	command.Flags().StringVar(&executable, "executable", "", "Local packaged launcher; locates its package directory")
	bindJSON(command, &jsonOutput)
	command.PreRunE = func(cmd *cobra.Command, args []string) error {
		if executable != "" && len(sources) > 0 {
			return fmt.Errorf("--from and --executable are mutually exclusive")
		}
		if executable == "" && len(sources) == 0 {
			return fmt.Errorf("provide --from <local-package>; release/source installation is not supported yet")
		}
		return nil
	}
	command.RunE = func(cmd *cobra.Command, args []string) error {
		if executable != "" {
			directory, err := engine.PackageForExecutable(executable)
			if err != nil {
				return err
			}
			sources = []string{directory}
		}
		packages := map[string]string{}
		for _, source := range sources {
			manifest, directory, err := engine.ReadPackage(source)
			if err != nil {
				return err
			}
			if _, exists := packages[manifest.Engine.ID]; exists {
				return fmt.Errorf("duplicate source for engine %q", manifest.Engine.ID)
			}
			packages[manifest.Engine.ID] = directory
		}
		if len(packages) != len(args) {
			return fmt.Errorf("provide exactly one local package per requested engine")
		}
		requested := map[string]bool{}
		for _, id := range args {
			if requested[id] || packages[id] == "" {
				return fmt.Errorf("missing or duplicate package for engine %q", id)
			}
			requested[id] = true
		}
		store, err := engine.DefaultStore()
		if err != nil {
			return err
		}
		response := result{Operation: "install", Status: "installed", Persisted: true}
		for _, id := range args {
			installed, err := store.Install(cmd.Context(), id, packages[id], cmd.ErrOrStderr())
			if err != nil {
				if len(response.Engines) > 0 {
					response.Status = "partial"
					return errors.Join(err, present(cmd, jsonOutput, response))
				}
				return err
			}
			response.Engines = append(response.Engines, installed)
		}
		return present(cmd, jsonOutput, response)
	}
	return command
}
