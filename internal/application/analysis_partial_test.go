package application

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func partialFixture(t *testing.T) (contract.AnalyzeRequest, contract.ImpactFacts, contract.Document) {
	t.Helper()
	request := contract.AnalyzeRequest{Repository: "/repo", Before: "before", After: "after", IncludeCoverage: true}
	symbol := contract.ImpactSymbol{ID: "value", Key: "value", Signature: "value", Fingerprint: strings.Repeat("a", 64), Location: contract.ImpactRange{File: "value.go", Start: contract.ImpactPosition{Line: 1, Column: 1}, End: contract.ImpactPosition{Line: 2, Column: 1}}}
	left := contract.ImpactSnapshot{Symbols: []contract.ImpactSymbol{symbol}, Calls: []contract.ImpactCall{}, Notes: []string{}}
	symbol.Fingerprint = strings.Repeat("b", 64)
	right := contract.ImpactSnapshot{Symbols: []contract.ImpactSymbol{symbol}, Calls: []contract.ImpactCall{}, Notes: []string{}}
	facts := contract.ImpactFacts{BeforeSHA: request.Before, AfterSHA: request.After, Before: left, After: right}
	document, err := compareImpact("go", request, facts)
	if err != nil {
		t.Fatal(err)
	}
	document, err = recordAnalysisStage(document, "go", "impact", "collected", "")
	if err != nil {
		t.Fatal(err)
	}
	return request, facts, document
}
func TestFailedEngineDoesNotDiscardAnotherEngineGraph(t *testing.T) {
	request, _, document := partialFixture(t)
	outcomes := append(analysisOutcomes(document), contract.EngineAnalysisOutcome{Engine: "ts-js", Status: "failed", Stages: []contract.AnalysisStage{{Name: "impact", Status: "failed", Message: "tool unavailable"}}})
	combined, err := mergeProjectAnalysis([]contract.Document{document}, outcomes, nil, request)
	if err != nil {
		t.Fatal(err)
	}
	var report contract.AnalyzeReport
	_ = json.Unmarshal(combined.Data, &report)
	if report.Status != "partial" || len(report.After.Nodes) != 1 || report.After.Nodes[0].Engine != "go" || len(report.EngineOutcomes) != 2 || len(report.Extensions["engineReports"]) == 0 {
		t.Fatal(string(combined.Data))
	}
	failed, err := mergeProjectAnalysis(nil, outcomes[1:], nil, request)
	if err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(failed.Data, &report)
	if report.Status != "failed" || report.Execution.Status != "UNAVAILABLE" {
		t.Fatal(string(failed.Data))
	}
}
func TestIncompleteCoverageRetainsGraphButNeverAttributesFailedRunHits(t *testing.T) {
	request, facts, document := partialFixture(t)
	evidence := contract.CoverageEvidence{Format: "go-coverprofile", Version: "1", Artifact: "profile", SHA256: strings.Repeat("a", 64), SourceMatch: "verified-snapshot", SourceSHA: "after", Granularity: "blocks", Files: []contract.CoverageFile{{Path: "value.go", Regions: []contract.CoverageRegion{{Start: contract.ImpactPosition{Line: 1, Column: 1}, End: contract.ImpactPosition{Line: 2, Column: 1}, Count: 5}}}}, Notes: []string{}}
	raw, _ := json.Marshal(evidence)
	collection := contract.CoverageReport{SchemaVersion: 1, Root: "/repo", Status: "partial", Engine: "go", Coverage: raw, Runs: []contract.CoverageRun{{Name: "tests", Status: "failed"}}}
	document, err := retainIncompleteCoverage(document, "go", request, facts, collection, errors.New("test failed"))
	if err != nil {
		t.Fatal(err)
	}
	var report contract.AnalyzeReport
	_ = json.Unmarshal(document.Data, &report)
	if report.Status != "partial" || len(report.After.Nodes) != 1 || report.Execution.Status != "FAILED" || report.Matching.Datasets[0].Methods[0].Status != "UNKNOWN" {
		t.Fatal(string(document.Data))
	}
	evidence = contract.CoverageEvidence{}
	_ = json.Unmarshal(report.CoverageReport.Coverage, &evidence)
	if evidence.SourceMatch != "unverified" || evidence.SourceSHA != "" {
		t.Fatal(evidence)
	}
}
