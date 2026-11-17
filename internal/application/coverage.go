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
func (s *Service) CollectCoverage(ctx context.Context, directory, stack, artifact string) (contract.Document, error) {
	project, err := projectfacts.Collect(ctx, directory, artifact != "" && stack == "", false)
	if err != nil {
		return contract.Document{}, err
	}
	result := contract.CoverageReport{SchemaVersion: contract.ReportVersion, Root: project.Root}
	if artifact == "" {
		result.Status = "unavailable"
		result.Message = "No coverage artifact supplied. Use --coverage-file with an export from the target build."
		return contract.NewDocument("", contract.CoverageOperation, result)
	}
	artifact, err = filepath.Abs(artifact)
	if err != nil {
		return contract.Document{}, err
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
	document, err := collector.CollectCoverage(ctx, contract.CoverageRequest{Repository: project.Root, Artifact: artifact})
	if err != nil {
		return contract.Document{}, err
	}
	var collected struct {
		Root     string          `json:"root"`
		Coverage json.RawMessage `json:"coverage"`
	}
	if err := json.Unmarshal(document.Data, &collected); err != nil {
		return contract.Document{}, err
	}
	var evidence contract.CoverageEvidence
	if err := json.Unmarshal(collected.Coverage, &evidence); err != nil {
		return contract.Document{}, err
	}
	if collected.Root != project.Root || evidence.Artifact != artifact {
		return contract.Document{}, fmt.Errorf("coverage response does not identify the requested project and artifact")
	}
	result.Engine, result.Status, result.Coverage = selected.Engine.ID, "collected", collected.Coverage
	if len(evidence.Files) == 0 {
		result.Status = "unavailable"
		result.Message = "The artifact contains no coverage paths mapped to this project."
	}
	return contract.NewDocument(selected.Engine.ID, contract.CoverageOperation, result)
}
