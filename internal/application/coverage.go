package application

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/projectfacts"
)

// CollectCoverage is reusable by operations that need coverage evidence.
func (s *Service) CollectCoverage(ctx context.Context, directory, stack string, request contract.CoverageRequest) (contract.Document, error) {
	project, err := projectfacts.Collect(ctx, directory, true, false)
	if err != nil {
		return contract.Document{}, err
	}
	targets, notes, err := s.analysisTargets(ctx, stack, project.Root, project.Root, contract.CoverageOperation)
	if err != nil {
		return contract.Document{}, err
	}
	if err = validateTestSelection(targets, request); err != nil {
		return contract.Document{}, err
	}
	if len(targets) > 1 && request.Artifact != "" {
		return contract.Document{}, fmt.Errorf("--coverage-file requires --stack when multiple engines match; artifact formats are engine-specific")
	}
	children := []contract.CoverageReport{}
	for _, target := range targets {
		perEngine := selectEngineCoverage(target, request)
		document, err := s.collectEngineCoverage(ctx, project.Root, target.analyzer.Stack(), target, perEngine)
		if ctx.Err() != nil {
			return document, ctx.Err()
		}
		if err != nil {
			children = append(children, contract.CoverageReport{SchemaVersion: 1, Root: project.Root, Engine: target.analyzer.Stack(), Status: "unavailable", Message: err.Error()})
			continue
		}
		var result contract.CoverageReport
		if err = json.Unmarshal(document.Data, &result); err != nil {
			return document, err
		}
		children = append(children, result)
	}
	result := aggregateCoverage(children, project.Root)
	for _, note := range notes {
		result.Message += "; " + note
	}
	return contract.NewDocument(stack, contract.CoverageOperation, result)
}
func (s *Service) collectEngineCoverage(ctx context.Context, root, stack string, target analysisTarget, request contract.CoverageRequest) (contract.Document, error) {
	request.Repository = root
	if request.Artifact != "" {
		absolute, err := filepath.Abs(request.Artifact)
		if err != nil {
			return contract.Document{}, err
		}
		request.Artifact = absolute
	}
	engineID := target.analyzer.Stack()
	if len(request.TestTargets) != 0 && !target.capabilities.CoverageTestTargets {
		return contract.Document{}, fmt.Errorf("engine %q does not support --test-target; install a newer package or use --test-binary", engineID)
	}
	if s.NewInstalledAnalyzer == nil {
		return contract.Document{}, fmt.Errorf("coverage engine execution is unavailable")
	}
	collector, ok := target.analyzer.(contract.CoverageCollector)
	if !ok {
		return contract.Document{}, fmt.Errorf("engine %q has no coverage adapter", engineID)
	}
	document, err := collector.CollectCoverage(ctx, request)
	if err != nil {
		return contract.Document{}, err
	}
	var result contract.CoverageReport
	if err := json.Unmarshal(document.Data, &result); err != nil {
		return contract.Document{}, err
	}
	if result.Root != root {
		return contract.Document{}, fmt.Errorf("coverage response does not identify the requested project")
	}
	result.Engine = engineID
	if len(result.Coverage) > 0 {
		var evidence contract.CoverageEvidence
		if err := json.Unmarshal(result.Coverage, &evidence); err != nil {
			return contract.Document{}, err
		}
		if request.Artifact != "" && evidence.Artifact != request.Artifact {
			return contract.Document{}, fmt.Errorf("coverage response does not identify the requested artifact")
		}
		if len(evidence.Files) == 0 {
			result.Status = "unavailable"
			result.Message = "The artifact contains no coverage paths mapped to this project."
		}
	}
	switch result.Status {
	case "":
		// Coverage import engines from the first ABI iteration omitted status.
		if len(result.Coverage) == 0 {
			return contract.Document{}, fmt.Errorf("engine omitted coverage evidence")
		}
		result.Status = "collected"
	case "collected", "partial", "unavailable":
	default:
		return contract.Document{}, fmt.Errorf("invalid coverage status %q", result.Status)
	}
	return contract.NewDocument(engineID, contract.CoverageOperation, result)
}
