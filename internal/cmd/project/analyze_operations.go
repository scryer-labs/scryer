package project

import (
	"fmt"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/spf13/cobra"
)

type analyzeOperations struct {
	impact   bool
	coverage bool
}

func selectAnalyzeOperations(value string) (analyzeOperations, error) {
	var selected analyzeOperations
	for _, operation := range strings.Split(value, ",") {
		var enabled *bool
		switch strings.TrimSpace(operation) {
		case "impact":
			enabled = &selected.impact
		case contract.CoverageOperation:
			enabled = &selected.coverage
		default:
			return analyzeOperations{}, fmt.Errorf("unknown analysis operation %q", strings.TrimSpace(operation))
		}
		if *enabled {
			return analyzeOperations{}, fmt.Errorf("duplicate analysis operation %q", strings.TrimSpace(operation))
		}
		*enabled = true
	}
	return selected, nil
}

func (selected analyzeOperations) validate(cmd *cobra.Command, request contract.AnalyzeRequest, artifact string, runTests bool) error {
	before, after := cmd.Flags().Changed("before"), cmd.Flags().Changed("after")
	if before != after || before && (strings.TrimSpace(request.Before) == "" || strings.TrimSpace(request.After) == "") {
		return fmt.Errorf("--before and --after must be supplied together with nonempty values")
	}
	if before && !selected.impact {
		return fmt.Errorf("--before and --after require an operation that compares versions")
	}
	if selected.coverage && (cmd.Flags().Changed("run-tests") || cmd.Flags().Changed("skip-tests") ||
		cmd.Flags().Changed("test-command") || cmd.Flags().Changed("build-tool")) {
		return fmt.Errorf("coverage reads existing artifacts; test execution and build flags are currently available for Java impact only")
	}
	if cmd.Flags().Changed("coverage-file") && (!selected.coverage || strings.TrimSpace(artifact) == "") {
		return fmt.Errorf("--coverage-file requires the coverage operation and a nonempty path")
	}
	if runTests && request.SkipTests {
		return fmt.Errorf("--run-tests and --skip-tests are mutually exclusive")
	}
	if selected.impact && selected.coverage {
		return fmt.Errorf("combining impact and coverage is not yet supported")
	}
	return nil
}
