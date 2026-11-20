package application

import (
	"encoding/json"
	"fmt"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func analysisOutcomes(document contract.Document) []contract.EngineAnalysisOutcome {
	var report struct {
		EngineOutcomes []contract.EngineAnalysisOutcome
	}
	_ = json.Unmarshal(document.Data, &report)
	return report.EngineOutcomes
}

func analysisStatus(outcomes []contract.EngineAnalysisOutcome) string {
	complete, usable := true, false
	for _, outcome := range outcomes {
		complete = complete && outcome.Status == "collected"
		usable = usable || outcome.Status == "collected" || outcome.Status == "partial"
	}
	if complete {
		return "collected"
	}
	if usable {
		return "partial"
	}
	return "failed"
}

// Metadata is applied to the raw report, retaining unknown engine fields.
func finishAnalysis(document contract.Document, outcomes []contract.EngineAnalysisOutcome) (contract.Document, error) {
	var report map[string]json.RawMessage
	if err := json.Unmarshal(document.Data, &report); err != nil {
		return document, err
	}
	report["status"], _ = json.Marshal(analysisStatus(outcomes))
	report["engineOutcomes"], _ = json.Marshal(outcomes)
	data, err := json.Marshal(report)
	document.Data = data
	return document, err
}

func recordAnalysisStage(document contract.Document, engine, name, status, message string) (contract.Document, error) {
	outcomes := analysisOutcomes(document)
	if len(outcomes) == 0 {
		outcomes = []contract.EngineAnalysisOutcome{{Engine: engine, Status: "collected", Stages: []contract.AnalysisStage{}}}
	}
	outcome := &outcomes[0]
	outcome.Stages = append(outcome.Stages, contract.AnalysisStage{Name: name, Status: status, Message: message})
	outcome.Status = "collected"
	for _, stage := range outcome.Stages {
		if stage.Status != "collected" {
			outcome.Status = "partial"
		}
	}
	return finishAnalysis(document, outcomes)
}

func mergeProjectAnalysis(documents []contract.Document, outcomes []contract.EngineAnalysisOutcome, notes []string, request contract.AnalyzeRequest) (contract.Document, error) {
	if len(documents) == 0 {
		empty := contract.ImpactSnapshot{Symbols: []contract.ImpactSymbol{}, Calls: []contract.ImpactCall{}, Notes: []string{}}
		document, err := compareImpact("multiple", request, contract.ImpactFacts{BeforeSHA: request.Before, AfterSHA: request.After, Before: empty, After: empty})
		if err != nil {
			return document, err
		}
		var report contract.AnalyzeReport
		if err = json.Unmarshal(document.Data, &report); err != nil {
			return document, err
		}
		report.Execution.Status = "UNAVAILABLE"
		report.Execution.Notes = []string{"No engine supplied usable impact facts. Empty graphs do not mean no impact."}
		report.Notes = append(report.Execution.Notes, notes...)
		document, err = contract.NewDocument("multiple", "analyze", report)
		if err != nil {
			return document, err
		}
		return finishAnalysis(document, outcomes)
	}
	document, err := mergeImpactDocuments(documents, notes)
	if err != nil {
		return document, err
	}
	if len(outcomes) > 1 {
		if len(documents) == 1 {
			var projected contract.AnalyzeReport
			if err = json.Unmarshal(document.Data, &projected); err != nil {
				return document, err
			}
			owner := documents[0].Stack
			namespaceSnapshot(&projected.Before, owner)
			namespaceSnapshot(&projected.After, owner)
			for i := range projected.Changes {
				projected.Changes[i].Engine = owner
			}
			for i := range projected.Matching.Datasets {
				dataset := &projected.Matching.Datasets[i]
				dataset.Engine = owner
				for j := range dataset.Methods {
					dataset.Methods[j].Symbol = owner + ":" + dataset.Methods[j].Symbol
				}
			}
			document, err = contract.NewDocument("multiple", "analyze", projected)
			if err != nil {
				return document, err
			}
		}
		var report map[string]json.RawMessage
		if err = json.Unmarshal(document.Data, &report); err != nil {
			return document, err
		}
		extensions := map[string]json.RawMessage{}
		if raw := report["extensions"]; len(raw) > 0 {
			if err = json.Unmarshal(raw, &extensions); err != nil {
				return document, err
			}
		}
		children := map[string]json.RawMessage{}
		for _, child := range documents {
			children[child.Stack] = child.Data
		}
		extensions["engineReports"], _ = json.Marshal(children)
		report["extensions"], _ = json.Marshal(extensions)
		document.Data, err = json.Marshal(report)
		if err != nil {
			return document, fmt.Errorf("aggregate engine reports: %w", err)
		}
	}
	return finishAnalysis(document, outcomes)
}
