// Package application coordinates public use cases without knowing CLI or Java protocol details.
package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/edwardnoaland/scryer/internal/repository"
)

type Service struct {
	Analyzer             contract.Analyzer
	NewInstalledAnalyzer func(engine.Installation) contract.Analyzer
}

func (s *Service) selectStack(stack string) error {
	if stack == "" {
		stack = "java"
	}
	if s.Analyzer == nil || stack != s.Analyzer.Stack() {
		supported := "none"
		if s.Analyzer != nil {
			supported = s.Analyzer.Stack()
		}
		return fmt.Errorf("unsupported stack %q; currently supported: %s", stack, supported)
	}
	return nil
}
func (s *Service) Scan(ctx context.Context, stack string, request contract.ScanRequest) (contract.Document, error) {
	if err := s.selectStack(stack); err != nil {
		return contract.Document{}, err
	}
	absolute, err := filepath.Abs(request.Repository)
	if err != nil {
		return contract.Document{}, err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return contract.Document{}, err
	}
	if !info.IsDir() {
		return contract.Document{}, fmt.Errorf("not a directory: %s", absolute)
	}
	request.Repository = absolute
	document, err := s.Analyzer.Scan(ctx, request)
	if err != nil {
		return document, err
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(document.Data, &data); err != nil {
		return document, err
	}
	data["name"], err = json.Marshal(filepath.Base(absolute))
	if err != nil {
		return document, err
	}
	data["repository"], err = json.Marshal(repository.Identity(ctx, absolute))
	if err != nil {
		return document, err
	}
	document.Data, err = json.Marshal(data)
	return document, err
}
func (s *Service) Analyze(ctx context.Context, stack, directory string, request contract.AnalyzeRequest) (document contract.Document, err error) {
	snapshots, err := repository.Open(ctx, directory, request.Before, request.After)
	if err != nil {
		return contract.Document{}, err
	}
	defer func() { err = errors.Join(err, snapshots.Close()) }()
	request.Repository = snapshots.Repository
	request.Before = snapshots.Before
	request.After = snapshots.After
	request.ComparisonRepository = snapshots.ComparisonRepository
	request.BeforeRoot = snapshots.BeforeRoot
	request.AfterRoot = snapshots.AfterRoot
	var targets []analysisTarget
	notes := []string{}
	if s.NewInstalledAnalyzer == nil {
		if err = s.selectStack(stack); err != nil {
			return document, err
		}
		targets = []analysisTarget{{analyzer: s.Analyzer}}
	} else {
		targets, notes, err = s.analysisTargets(ctx, stack, snapshots.BeforeRoot, snapshots.AfterRoot, contract.ImpactOperation)
		if err != nil {
			return document, err
		}
		if err = validateTestSelection(targets, request.Coverage); err != nil {
			return document, err
		}
	}
	dynamic := request.Resolve || request.RunTests || len(request.Coverage.TestTargets) > 0 || len(request.Coverage.TestBinaries) > 0
	documents := []contract.Document{}
	outcomes := []contract.EngineAnalysisOutcome{}
	for _, target := range targets {
		perEngine := request
		if target.projection {
			perEngine.Resolve = dynamic
			perEngine.Coverage = selectEngineCoverage(target, request.Coverage)
		}
		child, failure := s.analyzeSnapshotEngine(ctx, target.analyzer, target.projection, perEngine)
		if ctx.Err() != nil {
			return document, ctx.Err()
		}
		if failure != nil {
			status := "failed"
			if errors.Is(failure, ErrAnalysisUnsupported) {
				status = "unsupported"
			}
			outcomes = append(outcomes, contract.EngineAnalysisOutcome{Engine: target.analyzer.Stack(), Status: status, Stages: []contract.AnalysisStage{{Name: "impact", Status: status, Message: failure.Error()}}})
			continue
		}
		childOutcomes := analysisOutcomes(child)
		if len(childOutcomes) == 0 {
			childOutcomes = []contract.EngineAnalysisOutcome{{Engine: target.analyzer.Stack(), Status: "collected", Stages: []contract.AnalysisStage{{Name: "impact", Status: "collected"}}}}
		}
		var execution struct{ Execution struct{ Status string } }
		_ = json.Unmarshal(child.Data, &execution)
		if len(analysisOutcomes(child)) == 0 && execution.Execution.Status != "SUCCEEDED" && execution.Execution.Status != "SKIPPED" {
			childOutcomes[0].Status = "partial"
			childOutcomes[0].Stages = append(childOutcomes[0].Stages, contract.AnalysisStage{Name: "execution", Status: "failed", Message: execution.Execution.Status})
		}
		outcomes = append(outcomes, childOutcomes...)
		documents = append(documents, child)
	}
	document, err = mergeProjectAnalysis(documents, outcomes, notes, request)

	if err != nil || snapshots.WorkingBase == "" {
		return document, err
	}
	var data map[string]json.RawMessage
	if err = json.Unmarshal(document.Data, &data); err != nil {
		return document, err
	}
	data["afterSource"], err = json.Marshal(map[string]string{
		"kind": "WORKING_TREE", "ref": ".", "baseHead": snapshots.WorkingBase, "snapshotSha": snapshots.After,
	})
	if err != nil {
		return document, err
	}
	var snapshotNotes []string
	if err = json.Unmarshal(data["notes"], &snapshotNotes); err != nil {
		return document, err
	}
	snapshotNotes = append(snapshotNotes, "After is an isolated working-tree snapshot (staged, unstaged and non-ignored untracked files); its SHA is a private snapshot commit, not a commit in the original repository. Ignored untracked files are excluded.")
	data["notes"], err = json.Marshal(snapshotNotes)
	if err != nil {
		return document, err
	}
	document.Data, err = json.Marshal(data)
	return document, err
}

func (s *Service) analyzeSnapshotEngine(ctx context.Context, analyzer contract.Analyzer, projection bool, request contract.AnalyzeRequest) (document contract.Document, err error) {
	if projection {
		collector, ok := analyzer.(contract.ImpactCollector)
		if !ok {
			return document, fmt.Errorf("engine %q has no impact projection adapter", analyzer.Stack())
		}
		if request.TestCommand != "" || request.BuildTool != "" {
			return document, fmt.Errorf("engine %q uses native tests; --test-command and --build-tool belong to legacy analyze", analyzer.Stack())
		}
		request.Resolve = request.Resolve || request.RunTests || len(request.Coverage.TestTargets) != 0 || len(request.Coverage.TestBinaries) != 0
		var raw contract.Document
		raw, err = collector.CollectImpact(ctx, request)
		if err == nil {
			var facts contract.ImpactFacts
			if err = json.Unmarshal(raw.Data, &facts); err == nil {
				if request.Resolve {
					for _, snapshot := range []contract.ImpactSnapshot{facts.Before, facts.After} {
						var resolution struct {
							Status          string
							UsableForImpact bool
						}
						if json.Unmarshal(snapshot.Resolution, &resolution) != nil || (resolution.Status != "resolved" && !(resolution.Status == "partial" && resolution.UsableForImpact)) {
							return document, fmt.Errorf("engine %q did not return resolved scan evidence for dynamic impact; upgrade the engine", analyzer.Stack())
						}
					}
				}
				document, err = compareImpact(analyzer.Stack(), request, facts)
				if err == nil {
					document, err = recordAnalysisStage(document, analyzer.Stack(), "impact", "collected", "Engine scan facts retained; no static retry was performed.")
				}
				if err == nil && request.Resolve {
					for _, item := range []struct {
						name     string
						snapshot contract.ImpactSnapshot
					}{{"resolution.before", facts.Before}, {"resolution.after", facts.After}} {
						name, snapshot := item.name, item.snapshot
						var resolution struct {
							Status             string
							Message            string
							DiagnosticMessages []string
						}
						_ = json.Unmarshal(snapshot.Resolution, &resolution)
						status := "collected"
						if resolution.Status == "partial" {
							status = "partial"
						}
						document, err = recordAnalysisStage(document, analyzer.Stack(), name, status, resolution.Message+strings.Join(resolution.DiagnosticMessages, "\n"))
						if err != nil {
							return document, err
						}
					}
				}
				if err == nil && request.Resolve {
					document, err = s.attachImpactCoverage(ctx, analyzer.Stack(), request, facts, document)
				}
			}
		}
	} else {
		if request.Resolve {
			return document, fmt.Errorf("legacy analyze does not support --resolve; use its existing test lifecycle options: %w", ErrAnalysisUnsupported)
		}
		if request.IncludeCoverage || request.Coverage.Artifact != "" || len(request.Coverage.TestTargets) != 0 || len(request.Coverage.TestBinaries) != 0 {
			return document, fmt.Errorf("engine %q does not support separate impact coverage collection", analyzer.Stack())
		}
		request.SkipTests = request.SkipTests || (!request.RunTests && request.TestCommand == "")
		document, err = analyzer.Analyze(ctx, request)
	}
	return document, err
}
