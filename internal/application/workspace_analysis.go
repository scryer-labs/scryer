package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
)

var ErrAnalysisUnsupported = errors.New("analysis is not supported")

// AnalyzeWorkspace orchestrates the existing project use cases. Every repository
// chooses its own baseline; reports and coverage datasets are never merged.
func (s *Service) AnalyzeWorkspace(ctx context.Context, directory string, operations []string, resolve bool) (contract.WorkspaceAnalysis, error) {
	if len(operations) == 0 {
		operations = []string{contract.ImpactOperation}
	}
	selected, err := contract.SelectAnalyzeOperations(strings.Join(operations, ","))
	if err != nil {
		return contract.WorkspaceAnalysis{}, err
	}
	impact := slices.Contains(selected, contract.ImpactOperation)
	coverage := slices.Contains(selected, contract.CoverageOperation)
	if resolve && !impact {
		return contract.WorkspaceAnalysis{}, fmt.Errorf("--resolve requires impact; coverage already collects runtime evidence")
	}
	if impact && coverage && !resolve {
		return contract.WorkspaceAnalysis{}, fmt.Errorf("impact,coverage requires --resolve; use coverage alone to collect current-tree coverage")
	}
	root, err := filepath.Abs(directory)
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if err != nil {
		return contract.WorkspaceAnalysis{}, err
	}
	members, err := workspaceMembers(root)
	if err != nil {
		return contract.WorkspaceAnalysis{}, err
	}
	result := contract.WorkspaceAnalysis{
		SchemaVersion: contract.ReportVersion, Root: root, Operations: selected, Resolve: resolve, Status: "collected",
		Counts: map[string]int{"collected": 0, "failed": 0, "unsupported": 0}, Projects: []contract.ProjectAnalysis{},
		Notes: []string{"Projects use independent comparison baselines and retain complete project reports. No cross-project impact propagation or combined coverage percentage is inferred."},
	}
	for _, member := range members {
		if err := ctx.Err(); err != nil {
			return contract.WorkspaceAnalysis{}, err
		}
		project := contract.ProjectAnalysis{Path: filepath.ToSlash(member), Status: "collected"}
		var document contract.Document
		directory := filepath.Join(root, member)
		if impact {
			document, err = s.Analyze(ctx, "", directory, contract.AnalyzeRequest{Resolve: resolve, IncludeCoverage: coverage})
		} else {
			document, err = s.CollectCoverage(ctx, directory, "", contract.CoverageRequest{})
		}
		if err := ctx.Err(); err != nil {
			return contract.WorkspaceAnalysis{}, err
		}
		if err == nil {
			err = document.Validate()
		}
		if err != nil {
			project.Status, project.Error = "failed", err.Error()
			if errors.Is(err, ErrAnalysisUnsupported) {
				project.Status = "unsupported"
			}
		} else {
			project.Engine, project.ReportOperation, project.Report = document.Stack, document.Operation, document.Data
			var state struct {
				Status    string
				Message   string
				Execution struct{ Status string }
			}
			if err = json.Unmarshal(document.Data, &state); err != nil {
				project.Status, project.Error = "failed", err.Error()
			} else if impact && state.Execution.Status != "SUCCEEDED" && state.Execution.Status != "SKIPPED" {
				project.Status, project.Error = "failed", "project execution: "+state.Execution.Status
			} else if !impact && state.Status != "collected" {
				project.Status, project.Error = "failed", "project coverage: "+state.Status+"; "+state.Message
			}
		}
		result.Counts[project.Status]++
		result.Projects = append(result.Projects, project)
	}
	if result.Counts["collected"] != len(result.Projects) {
		result.Status = "partial"
		if result.Counts["collected"] == 0 {
			result.Status = "failed"
		}
	}
	return result, nil
}
