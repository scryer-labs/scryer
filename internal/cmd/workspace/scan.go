package workspace

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/edwardnoaland/scryer/internal/application"
	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/report"
	"github.com/spf13/cobra"
)

func newScanCommand(service *application.Service) *cobra.Command {
	var facts, output string
	var jsonOutput, resolve bool
	var selection contract.ScanSelection
	command := &cobra.Command{
		Use:   "scan [path]",
		Short: "Scan projects listed in .scryer/workspace.yaml",
		Args:  cobra.MaximumNArgs(1),
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if output != "" {
				switch strings.ToLower(filepath.Ext(output)) {
				case "", ".md":
				default:
					return fmt.Errorf("-o must use .md")
				}
			}
			var err error
			selection, err = contract.SelectScanFacts(facts)
			if err != nil {
				return err
			}
			if resolve && !slices.ContainsFunc(selection.EngineFacts, contract.IsResolvableScanFact) {
				return fmt.Errorf("--resolve requires dependencies, files or call-graph in --facts")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			result, err := service.ScanWorkspace(cmd.Context(), root, selection, resolve)
			if err != nil {
				return err
			}
			if output != "" {
				data, err := report.WorkspaceScanMarkdown(result)
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
			if result.Communication != nil && result.Communication.IPC != nil {
				fmt.Fprintln(out, "\nIPC (Windows named pipes)")
				for _, link := range result.Communication.IPC.Windows.NamedPipe.Links {
					target := link.To
					if target == "" {
						target = strings.Join(link.Candidates, ", ")
					}
					if target == "" {
						target = "unknown"
					}
					fmt.Fprintf(out, "  %s / %s [%s, %s] %s\n", link.From, target, link.Status, link.Direction, link.Name)
				}
			}
			if result.Communication != nil && result.Communication.IPC != nil {
				if memory := result.Communication.IPC.Windows.SharedMemory; memory != nil {
					fmt.Fprintln(out, "\nIPC (Windows shared memory)")
					for _, link := range memory.Links {
						target := link.To
						if target == "" {
							target = strings.Join(link.Candidates, ", ")
						}
						fmt.Fprintf(out, "  %s / %s [%s] %s\n", link.From, target, link.Status, link.Name)
					}
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
	command.Flags().BoolVar(&resolve, "resolve", false, "Allow build configuration and semantic resolution in workspace projects")
	command.Flags().BoolVar(&jsonOutput, "json", false, "Print report JSON")
	command.Flags().StringVarP(&output, "output", "o", "", "Save a Markdown report")
	return command
}
