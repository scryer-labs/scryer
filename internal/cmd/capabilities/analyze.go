// Package capabilities translates installed protocol capabilities into CLI labels.
package capabilities

import "github.com/edwardnoaland/scryer/internal/engine"

func AnalyzeOperations(capabilities *engine.Capabilities) []string {
	operations := []string{}
	for _, operation := range []string{"coverage", "impact"} {
		if capabilities.Supports(operation) {
			operations = append(operations, operation)
		}
	}
	if capabilities.Supports("analyze") {
		operations = append(operations, "legacy analyze")
	}
	return operations
}
