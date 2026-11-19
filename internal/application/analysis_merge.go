package application

import (
	"encoding/json"
	"fmt"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func namespaceSnapshot(snapshot *contract.Snapshot, engine string) {
	prefix := func(id string) string {
		if id == "" {
			return ""
		}
		return engine + ":" + id
	}
	for i := range snapshot.Nodes {
		node := &snapshot.Nodes[i]
		node.ID = prefix(node.ID)
		node.Engine = engine
	}
	for i := range snapshot.Edges {
		edge := &snapshot.Edges[i]
		edge.Caller = prefix(edge.Caller)
		edge.Callee = prefix(edge.Callee)
		edge.CallSiteID = prefix(edge.CallSiteID)
	}
	for i := range snapshot.Boundaries {
		boundary := &snapshot.Boundaries[i]
		if boundary.Caller != nil {
			value := prefix(*boundary.Caller)
			boundary.Caller = &value
		}
	}
	for i, note := range snapshot.Notes {
		snapshot.Notes[i] = engine + ": " + note
	}
	for i, note := range snapshot.UnresolvedChanges {
		snapshot.UnresolvedChanges[i] = engine + ": " + note
	}
}
func mergeImpactDocuments(documents []contract.Document, notes []string) (contract.Document, error) {
	if len(documents) == 0 {
		return contract.Document{}, fmt.Errorf("no engine impact reports")
	}
	if len(documents) == 1 {
		document := documents[0]
		if len(notes) == 0 {
			return document, nil
		}
		var report map[string]json.RawMessage
		if err := json.Unmarshal(document.Data, &report); err != nil {
			return document, err
		}
		var existing []string
		if raw, ok := report["notes"]; ok {
			if err := json.Unmarshal(raw, &existing); err != nil {
				return document, err
			}
		}
		report["notes"], _ = json.Marshal(append(existing, notes...))
		data, err := json.Marshal(report)
		document.Data = data
		return document, err
	}

	var combined contract.AnalyzeReport
	engineReports := map[string]json.RawMessage{}
	var coverages []contract.CoverageReport
	for i, document := range documents {
		var report contract.AnalyzeReport
		if err := json.Unmarshal(document.Data, &report); err != nil {
			return document, err
		}
		engineReports[document.Stack] = document.Data
		for j := range report.Changes {
			report.Changes[j].Engine = document.Stack
		}
		namespaceSnapshot(&report.Before, document.Stack)
		namespaceSnapshot(&report.After, document.Stack)
		for j := range report.Matching.Datasets {
			dataset := &report.Matching.Datasets[j]
			dataset.Engine = document.Stack
			for k := range dataset.Methods {
				dataset.Methods[k].Symbol = document.Stack + ":" + dataset.Methods[k].Symbol
			}
		}
		if report.CoverageReport != nil {
			coverages = append(coverages, *report.CoverageReport)
		}
		if i == 0 {
			combined = report
			continue
		}
		if combined.Repository != report.Repository || combined.BeforeSHA != report.BeforeSHA || combined.AfterSHA != report.AfterSHA {
			return document, fmt.Errorf("engine reports disagree on comparison identity")
		}
		combined.Files = append(combined.Files, report.Files...)
		combined.Changes = append(combined.Changes, report.Changes...)
		appendSnapshot(&combined.Before, report.Before)
		appendSnapshot(&combined.After, report.After)

		combined.Evidence.Coverage = append(combined.Evidence.Coverage, report.Evidence.Coverage...)
		combined.Matching.Datasets = append(combined.Matching.Datasets, report.Matching.Datasets...)
		combined.Matching.UnresolvedChanges = append(combined.Matching.UnresolvedChanges, report.Matching.UnresolvedChanges...)
		combined.Matching.Removed = append(combined.Matching.Removed, report.Matching.Removed...)
		combined.Execution.Notes = append(combined.Execution.Notes, report.Execution.Notes...)
		combined.Notes = append(combined.Notes, report.Notes...)
	}
	combined.Notes = append(combined.Notes, notes...)
	combined.Notes = append(combined.Notes, "Engine graphs and coverage datasets remain independent. Node IDs are engine-qualified; no cross-language call binding or combined coverage percentage is inferred.")
	if combined.Extensions == nil {
		combined.Extensions = map[string]json.RawMessage{}
	}
	combined.Extensions["engineReports"], _ = json.Marshal(engineReports)
	if len(coverages) > 0 {
		report := aggregateCoverage(coverages, combined.Repository)
		combined.CoverageReport = &report
	}
	return contract.NewDocument("multiple", "analyze", combined)
}
func aggregateCoverage(children []contract.CoverageReport, root string) contract.CoverageReport {
	if len(children) == 1 {
		return children[0]
	}
	result := contract.CoverageReport{SchemaVersion: 1, Root: root, Status: "collected", EngineReports: children, Runs: []contract.CoverageRun{}, Message: "Independent engine coverage datasets; percentages are not merged."}
	for _, child := range children {
		if child.Status != "collected" {
			result.Status = "partial"
		}
		for _, run := range child.Runs {
			run.Name = child.Engine + ": " + run.Name
			result.Runs = append(result.Runs, run)
		}
	}
	return result
}

func appendSnapshot(target *contract.Snapshot, source contract.Snapshot) {
	target.Nodes = append(target.Nodes, source.Nodes...)
	target.Edges = append(target.Edges, source.Edges...)
	target.Boundaries = append(target.Boundaries, source.Boundaries...)
	target.Notes = append(target.Notes, source.Notes...)
	target.UnresolvedChanges = append(target.UnresolvedChanges, source.UnresolvedChanges...)
	target.TotalBoundaryCount += source.TotalBoundaryCount
}
