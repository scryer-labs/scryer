package project

import (
	"fmt"
	"path/filepath"
	"regexp"
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
		return fmt.Errorf("these test execution and build flags are for Java impact; coverage runs tests automatically unless importing an artifact")
	}
	if cmd.Flags().Changed("coverage-file") && (!(selected.coverage || selected.impact) || strings.TrimSpace(artifact) == "") {
		return fmt.Errorf("--coverage-file requires a nonempty path")
	}
	if runTests && request.SkipTests {
		return fmt.Errorf("--run-tests and --skip-tests are mutually exclusive")
	}
	if request.SkipTests && (cmd.Flags().Changed("coverage-file") || cmd.Flags().Changed("test-target") || cmd.Flags().Changed("test-binary") || selected.coverage) {
		return fmt.Errorf("--skip-tests cannot be combined with coverage input or test selection")
	}
	return nil
}

var cmakeTargetName = regexp.MustCompile(`^[a-zA-Z0-9_.+][a-zA-Z0-9_.+-]*$`)

func (selected analyzeOperations) selectCoverageTests(request *contract.CoverageRequest, binaries, targets []string) error {
	if len(binaries) == 0 && len(targets) == 0 {
		return nil
	}
	if !(selected.coverage || selected.impact) || request.Artifact != "" || len(binaries) != 0 && len(targets) != 0 {
		return fmt.Errorf("--test-target and --test-binary are mutually exclusive with each other and --coverage-file")
	}
	var err error
	if len(targets) != 0 {
		request.TestTargets, err = selectTests("--test-target", targets, false)
	} else {
		request.TestBinaries, err = selectTests("--test-binary", binaries, true)
	}
	return err
}

func selectTests(flag string, arguments []string, paths bool) ([]string, error) {
	var results []string
	seen := map[string]bool{}
	for _, argument := range arguments {
		for _, value := range strings.Split(argument, ",") {
			value = strings.TrimSpace(value)
			if paths {
				if !filepath.IsLocal(value) || filepath.Clean(value) == "." || strings.ContainsRune(value, 0) {
					return nil, fmt.Errorf("%s requires an executable path relative to the temporary coverage build directory: %q", flag, value)
				}
				value = filepath.Clean(value)
			} else if !cmakeTargetName.MatchString(value) {
				return nil, fmt.Errorf("%s requires a CMake executable target name: %q", flag, value)
			}
			if seen[value] {
				return nil, fmt.Errorf("duplicate %s value %q", flag, value)
			}
			seen[value] = true
			results = append(results, value)
		}
	}
	return results, nil
}
