package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/repository"
)

// attachImpactCoverage reuses coverage collection against the immutable after checkout.
func (s *Service) attachImpactCoverage(ctx context.Context, stack string, request contract.AnalyzeRequest, facts contract.ImpactFacts, document contract.Document) (contract.Document, error) {
	var report contract.AnalyzeReport
	if err := json.Unmarshal(document.Data, &report); err != nil {
		return document, err
	}
	collection := contract.CoverageReport{SchemaVersion: 1, Root: request.Repository, Engine: stack}
	if err := repository.CheckSnapshot(ctx, request.AfterRoot, request.After); err != nil {
		return document, err
	}
	raw, err := s.CollectCoverage(ctx, request.AfterRoot, stack, request.Coverage)
	if err != nil {
		return document, fmt.Errorf("impact coverage failed; analysis stopped: %w", err)
	}
	if err := json.Unmarshal(raw.Data, &collection); err != nil {
		return document, err
	}
	if collection.Status != "collected" {
		details, _ := json.Marshal(collection.Runs)
		return document, fmt.Errorf("impact tests or coverage did not complete (%s): %s; runs: %s", collection.Status, collection.Message, details)
	}
	if err := repository.CheckSnapshot(ctx, request.AfterRoot, request.After); err != nil {
		return document, fmt.Errorf("impact stopped: %w", err)
	}
	collection.Root = request.Repository
	evidence := contract.CoverageEvidence{}
	if len(collection.Coverage) > 0 {
		if err := json.Unmarshal(collection.Coverage, &evidence); err != nil {
			return document, err
		}
	}
	if request.Coverage.Artifact != "" || len(collection.Coverage) == 0 {
		return document, fmt.Errorf("impact requires coverage collected from the selected after snapshot")
	}
	verified := true
	if verified {
		evidence.SourceMatch, evidence.SourceSHA = "verified-snapshot", request.After
		evidence.Notes = append(evidence.Notes, "Core verified the after checkout stayed clean at the requested commit across collection.")
		evidence.Notes = filterCoverageNotes(evidence.Notes)
		collection.Coverage, err = json.Marshal(evidence)
		if err != nil {
			return document, err
		}
	}
	executionStatus := "SUCCEEDED"
	report.Execution.Status = executionStatus
	report.Execution.Notes = []string{collection.Message}
	report.Evidence.CommandStatus = executionStatus
	report.Evidence.Notes = []string{"Aggregate engine-normalized source coverage; individual test attribution and branch adequacy are unknown."}
	report.Matching.Notes = []string{"Status refers to observed source regions, not proof of a caller path or assertion effectiveness."}
	if !verified {
		report.Matching.Notes = append(report.Matching.Notes, "Coverage is not verified against the after snapshot.")
	}
	report.Notes = []string{report.Notes[0], "Impact reuses engine-resolved scan evidence and coverage from the after checkout. Dynamic failures stop analysis."}
	rawCollection, err := json.Marshal(collection)
	if err != nil {
		return document, err
	}
	report.Evidence.Coverage = []json.RawMessage{rawCollection}
	dataset := matchImpactCoverage(report.After.Nodes, facts.After.Symbols, evidence, verified)
	dataset.Engine = stack
	report.Matching.Datasets = []contract.DatasetMatch{dataset}
	for _, change := range report.Changes {
		if change.Kind == "DELETED" && change.Before != nil {
			report.Matching.Removed = append(report.Matching.Removed, change.Before.Signature)
		}
	}
	if request.IncludeCoverage {
		report.CoverageReport = &collection
	}
	return contract.NewDocument(stack, "analyze", report)
}

func filterCoverageNotes(notes []string) []string {
	result := []string{}
	for _, note := range notes {
		if !strings.Contains(note, "working tree was not snapshotted") && !strings.Contains(note, "does not identify the source revision") {
			result = append(result, note)
		}
	}
	return result
}

func matchImpactCoverage(nodes []contract.Symbol, symbols []contract.ImpactSymbol, evidence contract.CoverageEvidence, verified bool) contract.DatasetMatch {
	dataset := contract.DatasetMatch{
		Methods: []contract.SymbolEvidence{},
		Counts:  map[string]int{"EXECUTED": 0, "NOT_EXECUTED": 0, "PARTIALLY_EXECUTED": 0, "UNKNOWN": 0, "NOT_APPLICABLE": 0},
	}
	if evidence.Artifact != "" {
		dataset.Artifact = &contract.Artifact{Source: evidence.Artifact, SHA256: evidence.SHA256}
	}
	ranges := map[string]contract.ImpactRange{}
	executionKinds := map[string]string{}
	for _, symbol := range symbols {
		ranges[symbol.ID] = symbol.Location
		executionKinds[symbol.ID] = symbol.ExecutionKind
	}
	files := map[string][]contract.CoverageRegion{}
	for _, file := range evidence.Files {
		files[file.Path] = file.Regions
	}
	for _, node := range nodes {
		match := contract.SymbolEvidence{
			Symbol: node.ID, Signature: node.Signature, Path: node.Path, Line: node.Line,
			Role: node.Role, Impact: node.Impact, Status: "UNKNOWN",
			Reason: "No verified source-region coverage for this symbol.", Routes: []contract.TestRoute{},
		}
		kind := executionKinds[node.ID]
		if kind == "compile-time" || kind == "declaration" || kind == "test-harness" {
			match.Status = "NOT_APPLICABLE"
			match.Reason = "Runtime function coverage does not apply to a declaration node; compiler references describe potential dependencies, not runtime execution."
			if kind == "test-harness" {
				match.Reason = "This collector does not instrument test-harness source; runtime function coverage is not applicable. Test execution and per-test coverage attribution remain separate evidence."
			}
			if kind == "compile-time" {
				match.Reason = "Immediate (consteval) function; runtime coverage does not apply. A compiler reference does not prove this function was evaluated."
			}
		} else if location, ok := ranges[node.ID]; verified && ok {
			hits, misses := 0, 0
			for _, region := range files[node.Path] {
				if positionBefore(region.Start, location.End) && positionBefore(location.Start, region.End) {
					if region.Count > 0 {
						hits++
					} else {
						misses++
					}
				}
			}
			switch {
			case hits > 0 && misses > 0:
				match.Status = "PARTIALLY_EXECUTED"
			case hits > 0:
				match.Status = "EXECUTED"
			case misses > 0:
				match.Status = "NOT_EXECUTED"
			}
			if hits+misses > 0 {
				match.Reason = fmt.Sprintf("%d source regions with hits and %d without hits within the symbol range; aggregate profile only.", hits, misses)
			}
		}
		dataset.Counts[match.Status]++
		if match.Status != "UNKNOWN" && match.Status != "NOT_APPLICABLE" {
			dataset.Assessed++
		}
		if match.Status == "EXECUTED" || match.Status == "PARTIALLY_EXECUTED" {
			dataset.WithHits++
		}
		dataset.Methods = append(dataset.Methods, match)
	}
	if dataset.Assessed > 0 {
		value := 100 * float64(dataset.WithHits) / float64(dataset.Assessed)
		dataset.MethodExecutionPercent = &value
	}
	return dataset
}

func positionBefore(left, right contract.ImpactPosition) bool {
	return left.Line < right.Line || left.Line == right.Line && left.Column < right.Column
}
