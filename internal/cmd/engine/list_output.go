package engine

import (
	"fmt"
	"strings"

	"github.com/edwardnoaland/scryer/internal/cmd/terminal"
	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/spf13/cobra"
)

func presentList(command *cobra.Command, installations []engine.Installation) error {
	mode, _ := command.Flags().GetString("color")
	colored := terminal.UseColor(mode, command.OutOrStdout())
	paint := func(code, text string) string {
		if !colored {
			return text
		}
		return "\x1b[" + code + "m" + text + "\x1b[0m"
	}
	var output strings.Builder
	for index, installed := range installations {
		if index > 0 {
			output.WriteByte('\n')
		}
		fmt.Fprintf(&output, "%s  %s  %s\n", paint("1;36", installed.Engine.ID), paint("35", installed.Engine.Version), paint("32", "installed"))
		entrypoint, label := installed.Library, "Library"
		if entrypoint == "" {
			entrypoint, label = installed.Executable, "Executable"
		}
		fmt.Fprintf(&output, "  %s %s\n", paint("2", label+":"), paint("2", entrypoint))
		if installed.Capabilities == nil {
			fmt.Fprintf(&output, "  %s\n", paint("33", "Capabilities unavailable"))
			continue
		}
		fmt.Fprintf(&output, "  %s %s\n", paint("2", "Operations:"), paint("36", strings.Join(installed.Capabilities.Operations, ", ")))
		fmt.Fprintf(&output, "  %s %s\n", paint("2", "Scan facts:"), strings.Join(installed.Capabilities.ScanFacts, ", "))
	}
	_, err := fmt.Fprint(command.OutOrStdout(), output.String())
	return err
}
