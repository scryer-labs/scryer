package application

import (
	"context"
	"fmt"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/edwardnoaland/scryer/internal/projectfacts"
	"github.com/edwardnoaland/scryer/internal/projectscope"
)

type analysisTarget struct {
	analyzer     contract.Analyzer
	capabilities *engine.Capabilities
	projection   bool
}

func (s *Service) analysisTargets(ctx context.Context, stack, before, after, operation string, config *projectscope.Config) ([]analysisTarget, []string, error) {
	store, err := engine.DefaultStore()
	if err != nil {
		return nil, nil, err
	}
	installed, err := store.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	left, err := projectfacts.CollectWithConfig(ctx, before, true, false, config)
	if err != nil {
		return nil, nil, err
	}
	right, err := projectfacts.CollectWithConfig(ctx, after, true, false, config)
	if err != nil {
		return nil, nil, err
	}
	var targets, legacy []analysisTarget
	notes := []string{}
	for _, candidate := range installed {
		if candidate.Capabilities == nil {
			notes = append(notes, candidate.Engine.ID+": capabilities unavailable")
			continue
		}
		if stack != "" && candidate.Engine.ID != stack {
			continue
		}
		if stack == "" && !matchesProject(candidate.Capabilities, left.Stack) && !matchesProject(candidate.Capabilities, right.Stack) {
			continue
		}
		if candidate.Capabilities.Supports(operation) {
			targets = append(targets, analysisTarget{s.NewInstalledAnalyzer(candidate), candidate.Capabilities, true})
		} else if operation == contract.ImpactOperation && candidate.Capabilities.Supports("analyze") {
			legacy = append(legacy, analysisTarget{s.NewInstalledAnalyzer(candidate), candidate.Capabilities, false})
		} else {
			notes = append(notes, candidate.Engine.ID+": "+operation+" is not supported by this installed package")
		}
	}
	if operation == contract.ImpactOperation {
		if len(targets) == 0 && len(legacy) == 1 {
			targets = legacy
		} else {
			for _, target := range legacy {
				notes = append(notes, target.analyzer.Stack()+": legacy analyze is separate; select it explicitly with --stack")
			}
		}
	}
	if len(targets) == 0 {
		return nil, notes, fmt.Errorf("no installed %s-capable engine matches this project or selected stack: %w", operation, ErrAnalysisUnsupported)
	}
	return targets, notes, nil
}
func selectEngineCoverage(target analysisTarget, request contract.CoverageRequest) contract.CoverageRequest {
	if target.capabilities == nil || !target.capabilities.CoverageTestTargets {
		request.TestTargets = nil
		request.TestBinaries = nil
	}
	return request
}
func validateTestSelection(targets []analysisTarget, request contract.CoverageRequest) error {
	if len(request.TestTargets) == 0 && len(request.TestBinaries) == 0 {
		return nil
	}
	for _, target := range targets {
		if target.capabilities != nil && target.capabilities.CoverageTestTargets {
			return nil
		}
	}
	return fmt.Errorf("no selected engine supports CMake target or build-binary selection; use --resolve for native test execution")
}
