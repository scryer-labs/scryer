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
