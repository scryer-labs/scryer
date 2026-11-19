package application

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/engine"
)

func TestMultiEngineImpactQualifiesIdentitiesAndKeepsSeparateEvidence(t *testing.T) {
	makeDocument := func(stack string) contract.Document {
		request := contract.AnalyzeRequest{Repository: "/repo", Before: "before", After: "after"}
		symbol := contract.ImpactSymbol{ID: "same", Key: "function", Signature: "value", Fingerprint: strings.Repeat("a", 64), Location: contract.ImpactRange{File: "value", Start: contract.ImpactPosition{Line: 1, Column: 1}, End: contract.ImpactPosition{Line: 2, Column: 1}}}
		left := contract.ImpactSnapshot{Symbols: []contract.ImpactSymbol{symbol}, Calls: []contract.ImpactCall{}, Notes: []string{}}
		symbol.Fingerprint = strings.Repeat("b", 64)
		right := contract.ImpactSnapshot{Symbols: []contract.ImpactSymbol{symbol}, Calls: []contract.ImpactCall{{ID: "site", Caller: "same", Candidate: "same", Callee: "value", Resolution: "candidate"}}, Notes: []string{}}
		doc, err := compareImpact(stack, request, contract.ImpactFacts{BeforeSHA: "before", AfterSHA: "after", Before: left, After: right})
		if err != nil {
			t.Fatal(err)
		}
		var report contract.AnalyzeReport
		_ = json.Unmarshal(doc.Data, &report)
		report.Matching.Datasets = []contract.DatasetMatch{{Methods: []contract.SymbolEvidence{{Symbol: "same", Status: "UNKNOWN"}}, Counts: map[string]int{"UNKNOWN": 1}}}
		doc, err = contract.NewDocument(stack, "analyze", report)
		if err != nil {
			t.Fatal(err)
		}
		return doc
	}
	documents := []contract.Document{makeDocument("go"), makeDocument("rust")}
	raw := string(documents[0].Data)
	merged, err := mergeImpactDocuments(documents, nil)
	if err != nil {
		t.Fatal(err)
	}
	var result contract.AnalyzeReport
	if err = json.Unmarshal(merged.Data, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.After.Nodes) != 2 || result.After.Nodes[0].ID != "go:same" || result.After.Nodes[1].ID != "rust:same" {
		t.Fatal(result.After.Nodes)
	}
	if result.After.Edges[0].Caller != "go:same" || result.After.Edges[1].Callee != "rust:same" {
		t.Fatal(result.After.Edges)
	}
	if result.Matching.Datasets[0].Engine != "go" || result.Matching.Datasets[1].Methods[0].Symbol != "rust:same" {
		t.Fatal(result.Matching)
	}
	if string(documents[0].Data) != raw || len(result.Extensions["engineReports"]) == 0 {
		t.Fatal("original evidence was lost or mutated")
	}
}
func TestNativeTestSelectionOnlyReachesSupportingEngines(t *testing.T) {
	request := contract.CoverageRequest{TestTargets: []string{"cpp_tests"}}
	cpp := analysisTarget{capabilities: &engine.Capabilities{CoverageTestTargets: true}}
	goTarget := analysisTarget{capabilities: &engine.Capabilities{Languages: []string{"Go"}}}
	if err := validateTestSelection([]analysisTarget{goTarget, cpp}, request); err != nil {
		t.Fatal(err)
	}
	if len(selectEngineCoverage(goTarget, request).TestTargets) != 0 || len(selectEngineCoverage(cpp, request).TestTargets) != 1 {
		t.Fatal("CMake selection leaked into a native engine")
	}
	if validateTestSelection([]analysisTarget{goTarget}, request) == nil {
		t.Fatal("unconsumed target selection must fail")
	}
}
