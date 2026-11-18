package application

import (
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
