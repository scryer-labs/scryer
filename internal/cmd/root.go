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
	"github.com/spf13/cobra"
)

func NewRoot(service *application.Service, out, errOut io.Writer) *cobra.Command {
	root := &cobra.Command{Use: "scryer", Short: "Repository facts and refactor test evidence", SilenceUsage: true, SilenceErrors: true}
	root.SetOut(out)
	root.SetErr(errOut)
	root.AddCommand(scan.New(service), analyze.New(service))
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError{err} })
	for _, child := range root.Commands() {
		arguments := child.Args
		child.Args = func(command *cobra.Command, args []string) error {
			if arguments != nil {
				if err := arguments(command, args); err != nil {
					return usageError{err}
				}
			}
			return nil
		}
		validate := child.PreRunE
		child.PreRunE = func(command *cobra.Command, args []string) error {
			if validate != nil {
				if err := validate(command, args); err != nil {
					return usageError{err}
				}
			}
			return nil
		}
	}
	return root
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
