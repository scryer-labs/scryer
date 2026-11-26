package application

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/edwardnoaland/scryer/internal/contract"
)

// compareImpact uses engine-provided correspondence keys and candidate edges.
func compareImpact(stack string, request contract.AnalyzeRequest, facts contract.ImpactFacts) (contract.Document, error) {
	if facts.BeforeSHA != request.Before || facts.AfterSHA != request.After {
		return contract.Document{}, fmt.Errorf("impact snapshot identity mismatch")
	}
	before, beforeAmbiguous := impactIndex(facts.Before.Symbols)
	after, afterAmbiguous := impactIndex(facts.After.Symbols)
	beforeSeeds, afterSeeds := map[string]bool{}, map[string]bool{}
	changes := []contract.Change{}
	unresolved := changedImpactContext(facts.Before.Contexts, facts.After.Contexts)
	keys := impactKeys(before, after)
	slices.Sort(keys)
	for _, key := range keys {
		left, hasLeft := before[key]
		right, hasRight := after[key]
		if beforeAmbiguous[key] || afterAmbiguous[key] {
			unresolved = append(unresolved, "Ambiguous declaration identity: "+key)
			continue
		}
		change := contract.Change{}
		switch {
		case !hasLeft:
			change.Kind = "ADDED"
		case !hasRight:
			change.Kind = "DELETED"
		case left.Fingerprint != right.Fingerprint:
			change.Kind = "MODIFIED"
		default:
			continue
		}
		if hasLeft {
			change.Before = declaration(left)
			beforeSeeds[left.ID] = true
		}
		if hasRight {
			change.After = declaration(right)
			afterSeeds[right.ID] = true
		}
		changes = append(changes, change)
	}
	notes := []string{
		"Potential impact derived from syntax observations. Candidate call edges do not establish compiler binding or runtime execution.",
		"Coverage is unknown in this static impact slice. No test execution or matched coverage dataset was collected.",
	}
	report := contract.AnalyzeReport{
		Scope: request.Scope, SchemaVersion: 1, GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Repository: request.Repository, BeforeSHA: request.Before, AfterSHA: request.After,
		Files: impactFiles(changes), Changes: changes,
		Before:    impactGraph(facts.Before, beforeSeeds, unresolved),
		After:     impactGraph(facts.After, afterSeeds, unresolved),
		Execution: contract.Execution{AfterSHA: request.After, Status: "SKIPPED", Command: []string{}, Notes: []string{"Static impact only."}},
		Evidence:  contract.Evidence{AfterSHA: request.After, CommandStatus: "SKIPPED", TestRecords: map[string]int{}, Tests: []json.RawMessage{}, Coverage: []json.RawMessage{}, Artifacts: []contract.Artifact{}, Notes: []string{"Coverage remains unknown."}},
		Matching:  contract.Matching{AfterSHA: request.After, Datasets: []contract.DatasetMatch{}, Removed: []string{}, UnresolvedChanges: unresolved, Notes: []string{"No revision-verified coverage."}},
		Notes:     notes,
	}
	report.Extensions = map[string]json.RawMessage{}
	report.Extensions["impactCompleteness"], _ = json.Marshal(map[string]any{
		"scope": "projected-declarations", "before": impactProjectionEvidence(facts.Before),
		"after": impactProjectionEvidence(facts.After), "unassessedChanges": len(unresolved),
		"limitations": []string{"Counts describe supplied projection evidence, not whole-repository coverage.", "Unchanged declaration tokens do not prove unchanged bindings, generated code or configuration effects.", "Absent call edges are not evidence of independence; source candidates and references do not prove runtime invocation."},
	})
	if len(facts.Before.Resolution) > 0 {
		report.Extensions["beforeResolution"] = facts.Before.Resolution
	}
	if len(facts.After.Resolution) > 0 {
		report.Extensions["afterResolution"] = facts.After.Resolution
	}
	if request.IncludeCoverage && !request.Resolve {
		report.CoverageReport = &contract.CoverageReport{SchemaVersion: 1, Root: request.Repository, Engine: stack, Status: "unavailable", Message: "Static impact does not run tests; supply --test-target or --test-binary for dynamic analysis."}
	}
	if request.Resolve {
		report.Notes[1] = "Coverage has not yet been verified; collection completion is reported separately."
		report.Execution.Notes = []string{"Dynamic scan projection collected; native execution evidence is a separate stage."}
		report.Notes[0] = "Impact uses the engine's resolved scan evidence. Binding strength and unresolved boundaries follow that engine's scan contract."
	}
	return contract.NewDocument(stack, "analyze", report)
}

func impactIndex(symbols []contract.ImpactSymbol) (map[string]contract.ImpactSymbol, map[string]bool) {
	index := map[string]contract.ImpactSymbol{}
	ambiguous := map[string]bool{}
	for _, symbol := range symbols {
		if _, exists := index[symbol.Key]; exists {
			ambiguous[symbol.Key] = true
		}
		index[symbol.Key] = symbol
	}
	return index, ambiguous
}

func impactKeys(before, after map[string]contract.ImpactSymbol) []string {
	keys := make([]string, 0, len(before)+len(after))
	for key := range before {
		keys = append(keys, key)
	}
	for key := range after {
		if _, exists := before[key]; !exists {
			keys = append(keys, key)
		}
	}
	return keys
}

func declaration(symbol contract.ImpactSymbol) *contract.DeclarationVersion {
	return &contract.DeclarationVersion{
		Signature: symbol.Signature, Path: &symbol.Location.File,
		Lines:  contract.LineRange{Start: symbol.Location.Start.Line, Count: symbol.Location.End.Line - symbol.Location.Start.Line + 1},
		Source: symbol.Source,
	}
}

func impactGraph(facts contract.ImpactSnapshot, changed map[string]bool, unresolved []string) contract.Snapshot {
	graph := contract.Snapshot{Nodes: []contract.Symbol{}, Edges: []contract.Relation{}, Boundaries: []contract.Boundary{}, Notes: facts.Notes, UnresolvedChanges: unresolved}
	ids := map[string]bool{}
	for _, symbol := range facts.Symbols {
		ids[symbol.ID] = true
	}
	incoming, outgoing := map[string][]string{}, map[string][]string{}
	for _, call := range facts.Calls {
		targets, kind := append([]string{}, call.Candidates...), "NAME_CANDIDATE"
		if call.Candidate != "" && !slices.Contains(targets, call.Candidate) {
			targets = append(targets, call.Candidate)
		}
		switch call.CandidateBasis {
		case "macro-forwarding":
			kind = "MACRO_FORWARD_CANDIDATE"
		case "struct-field-initializer":
			kind = "FIELD_POINTER_CANDIDATE"
		}
		if call.Target != "" {
			targets, kind = []string{call.Target}, "COMPILER_RESOLVED"
			switch call.Kind {
			case "declaration-reference":
				kind = "COMPILER_DECLARATION_REFERENCE"
			case "type-reference":
				kind = "COMPILER_TYPE_REFERENCE"
			case "scan-resolved":
				kind = "SCAN_RESOLVED"
			}
		}
		if call.Kind == "framework-reference" {
			kind = "FRAMEWORK_REFERENCE"
		}
		projected := false
		for _, target := range targets {
			if ids[call.Caller] && ids[target] {
				projected = true
				graph.Edges = append(graph.Edges, contract.Relation{Caller: call.Caller, Callee: target, Kind: kind, Basis: call.CandidateBasis, CallSiteID: call.ID, Path: call.Site.File, Line: call.Site.Start.Line})
				incoming[target] = append(incoming[target], call.Caller)
				outgoing[call.Caller] = append(outgoing[call.Caller], target)
			}
		}
		if !projected {
			var caller *string
			if ids[call.Caller] {
				caller = &call.Caller
			}
			graph.Boundaries = append(graph.Boundaries, contract.Boundary{
				Caller: caller, Path: call.Site.File, Line: call.Site.Start.Line,
				Expression: call.Callee, Reason: call.Resolution + "; no projected target",
			})
		}
	}
	callers := reachableImpact(changed, incoming)
	// Outgoing traversal starts from changed symbols, keeping siblings of callers outside the forward scope.
	indirect := reachableImpact(changed, outgoing)
	keep := map[string]bool{}
	for _, symbol := range facts.Symbols {
		impact := ""
		switch {
		case changed[symbol.ID]:
			impact = "CHANGED"
		case callers[symbol.ID]:
			impact = "CALLER"
		case indirect[symbol.ID]:
			impact = "POTENTIAL_INDIRECT"
		}
		if impact != "" {
			keep[symbol.ID] = true
			role := symbol.Role
			if role == "" {
				role = "UNKNOWN"
			}
			graph.Nodes = append(graph.Nodes, contract.Symbol{
				ExecutionKind: symbol.ExecutionKind, ID: symbol.ID, Signature: symbol.Signature, Path: symbol.Location.File,
				Line: symbol.Location.Start.Line, Role: role, TestContext: symbol.TestContext, Impact: impact,
			})
		}
	}
	graph.Edges = slices.DeleteFunc(graph.Edges, func(edge contract.Relation) bool { return !keep[edge.Caller] || !keep[edge.Callee] })
	graph.TotalBoundaryCount = len(graph.Boundaries)
	return graph
}

func reachableImpact(seeds map[string]bool, adjacency map[string][]string) map[string]bool {
	reachable := map[string]bool{}
	queue := make([]string, 0, len(seeds))
	for seed := range seeds {
		reachable[seed] = true
		queue = append(queue, seed)
	}
	for i := 0; i < len(queue); i++ {
		for _, next := range adjacency[queue[i]] {
			if !reachable[next] {
				reachable[next] = true
				queue = append(queue, next)
			}
		}
	}
	return reachable
}

// impactFiles projects declaration changes without presenting them as Git hunks.
func impactFiles(changes []contract.Change) []contract.FileChange {
	files := []contract.FileChange{}
	byPath := map[string]int{}
	for _, change := range changes {
		version := change.After
		if version == nil {
			version = change.Before
		}
		path := *version.Path
		index, exists := byPath[path]
		if !exists {
			index = len(files)
			byPath[path] = index
			files = append(files, contract.FileChange{Status: change.Kind, Lines: []contract.LineChange{}})
		}
		file := &files[index]
		if change.Before != nil {
			file.BeforePath = change.Before.Path
		}
		if change.After != nil {
			file.AfterPath = change.After.Path
		}
		if file.BeforePath != nil && file.AfterPath != nil {
			file.Status = "MODIFIED"
		}
	}
	return files
}

func changedImpactContext(before, after []contract.ImpactContext) []string {
	left, right := map[string]string{}, map[string]string{}
	for _, item := range before {
		left[item.Path] = item.Fingerprint
	}
	for _, item := range after {
		right[item.Path] = item.Fingerprint
	}
	changed := map[string]bool{}
	for path, hash := range left {
		if right[path] != hash {
			changed[path] = true
		}
	}
	for path, hash := range right {
		if left[path] != hash {
			changed[path] = true
		}
	}
	paths := make([]string, 0, len(changed))
	for path := range changed {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	notes := make([]string, 0, len(paths))
	for _, path := range paths {
		notes = append(notes, "Changed source/configuration context outside projected declarations: "+path+"; impact on function bodies is unknown.")
	}
	return notes
}

// Counts expose projection boundaries without asserting semantic completeness.
func impactProjectionEvidence(snapshot contract.ImpactSnapshot) map[string]any {
	ids := map[string]bool{}
	_, ambiguous := impactIndex(snapshot.Symbols)
	for _, symbol := range snapshot.Symbols {
		ids[symbol.ID] = true
	}
	unknownCaller, unknownTarget := 0, 0
	for _, call := range snapshot.Calls {
		if !ids[call.Caller] {
			unknownCaller++
		}
		targets := append([]string{}, call.Candidates...)
		if call.Candidate != "" {
			targets = append(targets, call.Candidate)
		}
		if call.Target != "" {
			targets = []string{call.Target}
		}
		projected := false
		for _, target := range targets {
			projected = projected || ids[target]
		}
		if !projected {
			unknownTarget++
		}
	}
	return map[string]any{"symbols": len(snapshot.Symbols), "relations": len(snapshot.Calls), "contexts": len(snapshot.Contexts), "ambiguousKeys": len(ambiguous), "unprojectedCallers": unknownCaller, "unprojectedTargets": unknownTarget}
}
