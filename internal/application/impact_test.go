package application

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func Test_should_keep_cycles_and_shared_callers_when_impact_is_potential(t *testing.T) {
	symbol := func(id string) contract.ImpactSymbol {
		return contract.ImpactSymbol{ID: id, Signature: id, Location: contract.ImpactRange{File: "main.cpp", Start: contract.ImpactPosition{Line: 1}}}
	}
	input := contract.ImpactSnapshot{
		Symbols: []contract.ImpactSymbol{symbol("changed"), symbol("caller"), symbol("root"), symbol("downstream"), symbol("sibling")},
		Calls: []contract.ImpactCall{
			{ID: "a", Caller: "caller", Candidate: "changed"},
			{ID: "b", Caller: "root", Candidate: "caller"},
			{ID: "c", Caller: "changed", Candidate: "downstream"},
			{ID: "d", Caller: "downstream", Candidate: "changed"},
			{ID: "e", Caller: "root", Candidate: "sibling"},
			{ID: "f", Caller: "changed", Callee: "external", Resolution: "unresolved"},
		},
	}
	graph := impactGraph(input, map[string]bool{"changed": true}, []string{})
	roles := map[string]string{}
	for _, node := range graph.Nodes {
		if roles[node.ID] != "" {
			t.Fatal("cycle produced duplicate nodes")
		}
		roles[node.ID] = node.Impact
	}
	if roles["changed"] != "CHANGED" || roles["root"] != "CALLER" || roles["caller"] != "CALLER" || roles["downstream"] != "CALLER" || roles["sibling"] != "" {
		t.Fatalf("unexpected potential impact roles: %v", roles)
	}
	if len(graph.Boundaries) != 1 || graph.Boundaries[0].Expression != "external" {
		t.Fatal("unresolved calls must remain inspectable")
	}
}

func Test_should_report_source_context_changes_when_no_function_body_changed(t *testing.T) {
	before := []contract.ImpactContext{{Path: "types.h", Fingerprint: "before"}}
	after := []contract.ImpactContext{{Path: "types.h", Fingerprint: "after"}}
	notes := changedImpactContext(before, after)
	if len(notes) != 1 || !strings.Contains(notes[0], "types.h") {
		t.Fatal("source context change was hidden")
	}
}

func Test_should_preserve_multiple_indirect_candidates_without_compiler_binding(t *testing.T) {
	snapshot := contract.ImpactSnapshot{Symbols: []contract.ImpactSymbol{{ID: "caller"}, {ID: "a"}, {ID: "b"}}, Calls: []contract.ImpactCall{{ID: "site", Caller: "caller", Candidates: []string{"a", "b"}, CandidateBasis: "struct-field-initializer"}}}
	graph := impactGraph(snapshot, map[string]bool{"caller": true}, nil)
	if len(graph.Edges) != 2 {
		t.Fatal(graph)
	}
	for _, edge := range graph.Edges {
		if edge.Kind != "FIELD_POINTER_CANDIDATE" || edge.CallSiteID != "site" {
			t.Fatal(edge)
		}
	}
}

func Test_should_expose_projection_limits_when_context_or_identity_is_unassessed(t *testing.T) {
	before := contract.ImpactSnapshot{Symbols: []contract.ImpactSymbol{{ID: "a", Key: "callback"}, {ID: "b", Key: "callback"}}, Contexts: []contract.ImpactContext{{Path: "go.mod", Fingerprint: "left"}}, Calls: []contract.ImpactCall{{ID: "unbound", Caller: "missing", Resolution: "unresolved"}}}
	after := before
	after.Contexts = []contract.ImpactContext{{Path: "go.mod", Fingerprint: "right"}}
	doc, err := compareImpact("go", contract.AnalyzeRequest{Before: "before", After: "after"}, contract.ImpactFacts{BeforeSHA: "before", AfterSHA: "after", Before: before, After: after})
	if err != nil {
		t.Fatal(err)
	}
	var report contract.AnalyzeReport
	if err = json.Unmarshal(doc.Data, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Changes) != 0 || len(report.Matching.UnresolvedChanges) != 2 {
		t.Fatal(report.Matching)
	}
	var evidence struct {
		Scope             string
		UnassessedChanges int
		Before            struct{ Symbols, Relations, Contexts, AmbiguousKeys, UnprojectedCallers, UnprojectedTargets int }
	}
	if err = json.Unmarshal(report.Extensions["impactCompleteness"], &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.Scope != "projected-declarations" || evidence.UnassessedChanges != 2 || evidence.Before.Symbols != 2 || evidence.Before.AmbiguousKeys != 1 || evidence.Before.UnprojectedTargets != 1 || evidence.Before.UnprojectedCallers != 1 {
		t.Fatalf("lost comparison limits: %+v", evidence)
	}
}

func Test_should_keep_framework_references_distinct_from_calls_when_traversing_impact(t *testing.T) {
	snapshot := contract.ImpactSnapshot{Symbols: []contract.ImpactSymbol{{ID: "component"}, {ID: "handler"}}, Calls: []contract.ImpactCall{{ID: "reference", Kind: "framework-reference", Caller: "component", Candidate: "handler", CandidateBasis: "react-handler"}}}
	graph := impactGraph(snapshot, map[string]bool{"handler": true}, nil)
	if len(graph.Nodes) != 2 || len(graph.Edges) != 1 || graph.Edges[0].Kind != "FRAMEWORK_REFERENCE" || graph.Edges[0].Basis != "react-handler" {
		t.Fatal(graph)
	}
}
