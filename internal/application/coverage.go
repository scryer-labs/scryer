package application

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/edwardnoaland/scryer/internal/projectfacts"
)

// CollectCoverage is reusable by operations that need coverage evidence.
func (s *Service) CollectCoverage(ctx context.Context, directory, stack string, request contract.CoverageRequest) (contract.Document, error) {
	project, err := projectfacts.Collect(ctx, directory, stack == "", false)
	if err != nil {
		return contract.Document{}, err
	}
	request.Repository = project.Root
	if request.Artifact != "" {
		request.Artifact, err = filepath.Abs(request.Artifact)
		if err != nil {
			return contract.Document{}, err
		}
	}
	store, err := engine.DefaultStore()
	if err != nil {
		return contract.Document{}, err
	}
	installed, err := store.List(ctx)
	if err != nil {
		return contract.Document{}, err
	}
	var selected *engine.Installation
	for _, candidate := range installed {
		if !candidate.Capabilities.Supports(contract.CoverageOperation) {
			continue
		}
		if stack != "" && stack != candidate.Engine.ID || stack == "" && !matchesProject(candidate.Capabilities, project.Stack) {
			continue
		}
		if selected != nil {
			return contract.Document{}, fmt.Errorf("multiple engines support coverage for this project; select one with --stack")
		}
		selected = &candidate
	}
	if selected == nil {
		return contract.Document{}, fmt.Errorf("no installed coverage-capable engine matches the project or selected stack")
	}
	if s.NewInstalledAnalyzer == nil {
		return contract.Document{}, fmt.Errorf("coverage engine execution is unavailable")
	}
	collector, ok := s.NewInstalledAnalyzer(*selected).(contract.CoverageCollector)
	if !ok {
		return contract.Document{}, fmt.Errorf("engine %q has no coverage adapter", selected.Engine.ID)
	}
	document, err := collector.CollectCoverage(ctx, request)
	if err != nil {
		return contract.Document{}, err
	}
	var result contract.CoverageReport
	if err := json.Unmarshal(document.Data, &result); err != nil {
		return contract.Document{}, err
	}
	if result.Root != project.Root {
		return contract.Document{}, fmt.Errorf("coverage response does not identify the requested project")
	}
	result.Engine = selected.Engine.ID
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
	return contract.NewDocument(selected.Engine.ID, contract.CoverageOperation, result)
}
