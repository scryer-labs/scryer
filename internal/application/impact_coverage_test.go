package application

import (
	"testing"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func Test_should_keep_coverage_unknown_when_source_provenance_is_unverified(t *testing.T) {
	location := contract.ImpactRange{File: "main.cpp", Start: contract.ImpactPosition{Line: 1, Column: 1}, End: contract.ImpactPosition{Line: 1, Column: 40}}
	symbols := []contract.ImpactSymbol{{ID: "function", Location: location}}
	nodes := []contract.Symbol{{ID: "function", Path: "main.cpp", Impact: "CHANGED"}}
	evidence := contract.CoverageEvidence{Files: []contract.CoverageFile{{
		Path: "main.cpp", Regions: []contract.CoverageRegion{{Start: location.Start, End: location.End, Count: 1}},
	}}}
	unknown := matchImpactCoverage(nodes, symbols, evidence, false)
	if unknown.Counts["UNKNOWN"] != 1 || unknown.MethodExecutionPercent != nil {
		t.Fatal("unverified coverage must not label a symbol executed")
	}
	verified := matchImpactCoverage(nodes, symbols, evidence, true)
	if verified.Counts["EXECUTED"] != 1 || verified.WithHits != 1 {
		t.Fatal("verified overlapping regions must retain observed hits")
	}
	evidence.Files[0].Regions[0].Start = location.End
	evidence.Files[0].Regions[0].End = contract.ImpactPosition{Line: 2, Column: 1}
	outside := matchImpactCoverage(nodes, symbols, evidence, true)
	if outside.Counts["UNKNOWN"] != 1 {
		t.Fatal("exclusive end must exclude a neighboring region")
	}
}

func Test_should_exclude_immediate_functions_and_declarations_from_runtime_coverage(t *testing.T) {
	location := contract.ImpactRange{File: "traits.h", Start: contract.ImpactPosition{Line: 1, Column: 1}, End: contract.ImpactPosition{Line: 1, Column: 80}}
	symbols := []contract.ImpactSymbol{
		{ID: "immediate", ExecutionKind: "compile-time", Location: location},
		{ID: "variable", ExecutionKind: "declaration", Location: location},
		{ID: "constexpr", Location: location},
	}
	nodes := []contract.Symbol{{ID: "immediate", Path: "traits.h"}, {ID: "variable", Path: "traits.h"}, {ID: "constexpr", Path: "traits.h"}}
	evidence := contract.CoverageEvidence{Files: []contract.CoverageFile{{Path: "traits.h", Regions: []contract.CoverageRegion{{Start: location.Start, End: location.End, Count: 1}}}}}
	result := matchImpactCoverage(nodes, symbols, evidence, true)
	if result.Counts["NOT_APPLICABLE"] != 2 || result.Counts["EXECUTED"] != 1 || result.Assessed != 1 || result.WithHits != 1 {
		t.Fatalf("compile-time declarations contaminated runtime coverage: %#v", result)
	}
}

func Test_should_preserve_engine_test_role_and_classification_evidence_in_impact_graph(t *testing.T) {
	context := []byte(`{"scope":"test-only","evidence":[{"kind":"compiler-test-registration","framework":"example"}]}`)
	facts := contract.ImpactSnapshot{Symbols: []contract.ImpactSymbol{{ID: "case", Role: "TEST", TestContext: context}, {ID: "helper"}}}
	graph := impactGraph(facts, map[string]bool{"case": true, "helper": true}, nil)
	if graph.Nodes[0].Role != "TEST" || string(graph.Nodes[0].TestContext) != string(context) || graph.Nodes[1].Role != "UNKNOWN" {
		t.Fatalf("lost engine role or classified an unproven helper: %#v", graph.Nodes)
	}
}
