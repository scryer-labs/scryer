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

	"github.com/edwardnoaland/scryer/internal/analyzer/external"
	"github.com/edwardnoaland/scryer/internal/application"
	enginecmd "github.com/edwardnoaland/scryer/internal/cmd/engine"
	projectcmd "github.com/edwardnoaland/scryer/internal/cmd/project"
	workspacecmd "github.com/edwardnoaland/scryer/internal/cmd/workspace"
	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/spf13/cobra"
)

func NewRoot(service *application.Service, out, errOut io.Writer) *cobra.Command {
	root := &cobra.Command{Use: "scryer", Short: "Repository facts and change/test evidence", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(out)
	root.SetErr(errOut)
	root.AddCommand(projectcmd.New(service), workspacecmd.New(), enginecmd.New())

	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError{err} })
	wrapCommandValidation(root)
	return root
}

// Validation applies to all nested scope commands.
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
	root := NewRoot(&application.Service{
		Analyzer: external.New("java", os.Stderr),
		NewInstalledAnalyzer: func(installed engine.Installation) contract.Analyzer {
			client := external.New(installed.Engine.ID, os.Stderr)
			client.Executable = installed.Library
			if client.Executable == "" {
				client.Executable = installed.Executable
			}
			return client
		},
	}, os.Stdout, os.Stderr)
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
