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
	beforeCheck := repository.CheckSnapshot(ctx, request.AfterRoot, request.After)
	raw, collectionErr := s.CollectCoverage(ctx, request.AfterRoot, stack, request.Coverage)
	if ctx.Err() != nil {
		return document, ctx.Err()
	}
	if collectionErr == nil {
		if err := json.Unmarshal(raw.Data, &collection); err != nil {
			return document, err
		}
	} else {
		collection.Status, collection.Message = "unavailable", collectionErr.Error()
	}
	afterCheck := repository.CheckSnapshot(ctx, request.AfterRoot, request.After)
	collection.Root = request.Repository
	evidence := contract.CoverageEvidence{}
	if len(collection.Coverage) > 0 {
		if err := json.Unmarshal(collection.Coverage, &evidence); err != nil {
			return document, err
		}
	}
	verified := beforeCheck == nil && afterCheck == nil && request.Coverage.Artifact == "" && len(collection.Coverage) > 0
	if verified {
		evidence.SourceMatch, evidence.SourceSHA = "verified-snapshot", request.After
		evidence.Notes = append(evidence.Notes, "Core verified the after checkout stayed clean at the requested commit across collection.")
		evidence.Notes = filterCoverageNotes(evidence.Notes)
		collection.Coverage, collectionErr = json.Marshal(evidence)
		if collectionErr != nil {
			return document, collectionErr
		}
	}
	executionStatus := "UNAVAILABLE"
	switch collection.Status {
	case "collected":
		executionStatus = "SUCCEEDED"
	case "partial":
		executionStatus = "FAILED"
	}
	if request.Coverage.Artifact != "" {
		executionStatus = "SKIPPED"
	}
	if beforeCheck != nil || afterCheck != nil {
		executionStatus = "SNAPSHOT_CHANGED"
		evidence.SourceMatch = "unverified"
		collection.Message = "After checkout changed during collection; coverage matching remains unknown."
		collection.Status = "partial"
	}
	report.Execution.Status = executionStatus
	report.Execution.Notes = []string{collection.Message}
	report.Evidence.CommandStatus = executionStatus
	report.Evidence.Notes = []string{"Aggregate LLVM source-region evidence; individual test attribution and branch adequacy are unknown."}
	report.Matching.Notes = []string{"Status refers to observed source regions, not proof of a caller path or assertion effectiveness."}
	if !verified {
		report.Matching.Notes = append(report.Matching.Notes, "Coverage is not verified against the after snapshot.")
	}
	report.Notes = []string{report.Notes[0], "Impact coverage is collected on the after checkout. Test or collection failures remain visible alongside static impact."}
	rawCollection, err := json.Marshal(collection)
	if err != nil {
		return document, err
	}
	report.Evidence.Coverage = []json.RawMessage{rawCollection}
	dataset := matchImpactCoverage(report.After.Nodes, facts.After.Symbols, evidence, verified)
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
		Counts:  map[string]int{"EXECUTED": 0, "NOT_EXECUTED": 0, "PARTIALLY_EXECUTED": 0, "UNKNOWN": 0},
	}
	if evidence.Artifact != "" {
		dataset.Artifact = &contract.Artifact{Source: evidence.Artifact, SHA256: evidence.SHA256}
	}
	ranges := map[string]contract.ImpactRange{}
	for _, symbol := range symbols {
		ranges[symbol.ID] = symbol.Location
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
		if location, ok := ranges[node.ID]; verified && ok {
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
		if match.Status != "UNKNOWN" {
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
