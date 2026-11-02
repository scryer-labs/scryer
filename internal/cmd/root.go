// Package cmd composes Cobra commands and adapters; main only dispatches here.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/edwardnoaland/scryer/internal/analyzer/java"
	"github.com/edwardnoaland/scryer/internal/application"
	"github.com/edwardnoaland/scryer/internal/cmd/analyze"
	"github.com/edwardnoaland/scryer/internal/cmd/scan"
	workspacecmd "github.com/edwardnoaland/scryer/internal/cmd/workspace"
	"github.com/spf13/cobra"
)

func NewRoot(service *application.Service, out, errOut io.Writer) *cobra.Command {
	root := &cobra.Command{Use: "scryer", Short: "Repository facts and change/test evidence", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(out)
	root.SetErr(errOut)
	project := &cobra.Command{Use: "project", Short: "Scan and analyze a single repository"}
	project.AddCommand(scan.New(service), analyze.New(service))
	root.AddCommand(project, workspacecmd.New())

	// Keep existing providers working until their planned CLI migration.
	legacyScan := scan.New(service)
	markLegacyEntry(legacyScan, "scryer project scan")
	legacyAnalyze := analyze.New(service)
	markLegacyEntry(legacyAnalyze, "scryer project analyze")
	legacyWorkspace := workspacecmd.NewImport()
	legacyWorkspace.Use = "workspace --projects <projects.csv> --dependencies <dependencies.csv>"
	markLegacyEntry(legacyWorkspace, "scryer workspace import")
	legacyAnalyze.AddCommand(legacyWorkspace)
	root.AddCommand(legacyScan, legacyAnalyze)

	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError{err} })
	wrapCommandValidation(root)
	return root
}

// Cobra's built-in deprecation notice uses stdout; keep machine JSON clean.
func markLegacyEntry(command *cobra.Command, replacement string) {
	command.Hidden = true
	command.Long += "\nDeprecated compatibility entry; use " + replacement + " instead."
	validate := command.PreRunE
	command.PreRunE = func(cmd *cobra.Command, args []string) error {
		if validate != nil {
			if err := validate(cmd, args); err != nil {
				return err
			}
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "scryer: deprecated command; use "+replacement+" instead")
		return nil
	}
}

// Validation applies to nested scope commands as well as transitional entries.
func wrapCommandValidation(command *cobra.Command) {
	arguments := command.Args
	if arguments != nil {
		command.Args = func(cmd *cobra.Command, args []string) error {
			if err := arguments(cmd, args); err != nil {
				return usageError{err}
			}
			return nil
		}
	}
	validate := command.PreRunE
	if validate != nil {
		command.PreRunE = func(cmd *cobra.Command, args []string) error {
			if err := validate(cmd, args); err != nil {
				return usageError{err}
			}
			return nil
		}
	}
	for _, child := range command.Commands() {
		wrapCommandValidation(child)
	}
}

func Execute() int {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	root := NewRoot(&application.Service{Analyzer: java.New(os.Stderr)}, os.Stdout, os.Stderr)
	if err := root.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "scryer:", err)
		return exitCode(err)
	}
	return 0
}

type usageError struct{ error }

func exitCode(err error) int {
	var usage usageError
	if errors.As(err, &usage) || strings.HasPrefix(err.Error(), "unknown command") {
		return 2
	}
	if errors.Is(err, context.Canceled) {
		return 130
	}
	return 1
}
