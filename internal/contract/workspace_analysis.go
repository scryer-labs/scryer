package contract

import (
	"encoding/json"
	"fmt"
	"strings"
)

func SelectAnalyzeOperations(value string) ([]string, error) {
	operations := []string{}
	seen := map[string]bool{}
	for _, part := range strings.Split(value, ",") {
		operation := strings.TrimSpace(part)
		if operation != ImpactOperation && operation != CoverageOperation {
			return nil, fmt.Errorf("unknown analysis operation %q", operation)
		}
		if seen[operation] {
			return nil, fmt.Errorf("duplicate analysis operation %q", operation)
		}
		seen[operation] = true
		operations = append(operations, operation)
	}
	return operations, nil
}

type WorkspaceAnalysis struct {
	SchemaVersion int               `json:"schemaVersion"`
	Root          string            `json:"root"`
	Operations    []string          `json:"operations"`
	Resolve       bool              `json:"resolve"`
	Status        string            `json:"status"`
	Counts        map[string]int    `json:"counts"`
	Projects      []ProjectAnalysis `json:"projects"`
	Notes         []string          `json:"notes"`
}

type ProjectAnalysis struct {
	Path            string          `json:"path"`
	Status          string          `json:"status"`
	Engine          string          `json:"engine,omitempty"`
	ReportOperation string          `json:"reportOperation,omitempty"`
	Report          json.RawMessage `json:"report,omitempty"`
	Error           string          `json:"error,omitempty"`
}

func (project ProjectAnalysis) Document() Document {
	return Document{Stack: project.Engine, Operation: project.ReportOperation, Data: project.Report}
}
