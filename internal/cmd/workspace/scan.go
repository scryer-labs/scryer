package workspace

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/edwardnoaland/scryer/internal/application"
	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/report"
	"github.com/spf13/cobra"
)

func newScanCommand(service *application.Service) *cobra.Command {
	var facts, output string
	var jsonOutput bool
	var selection contract.ScanSelection
	command := &cobra.Command{
		Use:   "scan [path]",
		Short: "Scan projects listed in .scryer/workspace.yaml",
		Args:  cobra.MaximumNArgs(1),
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if output != "" {
				switch strings.ToLower(filepath.Ext(output)) {
				case "", ".md", ".html":
				default:
					return fmt.Errorf("-o must use .md or .html")
				}
			}
			var err error
			selection, err = contract.SelectScanFacts(facts)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			result, err := service.ScanWorkspace(cmd.Context(), root, selection)
			if err != nil {
				return err
			}
			if output != "" {
				var data []byte
				if strings.EqualFold(filepath.Ext(output), ".html") {
					data, err = report.ScanHTML(result)
				} else {
					data, err = report.WorkspaceScanMarkdown(result)
				}
				if err != nil {
					return err
				}
				if err := report.WriteFile(output, data); err != nil {
					return err
				}
				fmt.Fprintln(cmd.ErrOrStderr(), "scryer: report written to "+output)
			}
			if jsonOutput {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				return encoder.Encode(result)
			}
			out := cmd.OutOrStdout()
			if result.Dependencies != nil {
				fmt.Fprintln(out, result.Dependencies.Scope)
				fmt.Fprintf(out, "Unsupported dependency types: %s\n", strings.Join(result.Dependencies.Unsupported, ", "))
				for _, ref := range result.Dependencies.References {
					target := ref.Repository.ID
					if ref.To != "" {
						target = ref.To
					}
					fmt.Fprintf(out, "  %s → %s [%s]\n", ref.From, target, ref.Status)
				}
			}
			for _, project := range result.Projects {
				if project.Error != "" {
					fmt.Fprintf(out, "\nProject %s: %s\n", project.Path, project.Error)
					continue
				}
				document, err := contract.NewDocument("", contract.ProjectFactsOperation, project.Facts)
				if err != nil {
					return err
				}
				fmt.Fprintln(out)
				if err := report.Terminal(document, out, false, false); err != nil {
					return err
				}
			}
			return nil
		},
	}
	command.Flags().StringVar(&facts, "facts", "stack,loc", "Collect selected facts")
	command.Flags().BoolVar(&jsonOutput, "json", false, "Print report JSON")
	command.Flags().StringVarP(&output, "output", "o", "", "Save a Markdown or HTML report")
	return command
}
