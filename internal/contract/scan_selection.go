package contract

import (
	"fmt"
	"slices"
	"strings"
)

type ScanSelection struct {
	Stack       bool
	LOC         bool
	EngineFacts []string
}

func SelectScanFacts(facts string) (ScanSelection, error) {
	var selection ScanSelection
	for _, fact := range strings.Split(facts, ",") {
		fact = strings.TrimSpace(fact)
		switch fact {
		case "stack":
			selection.Stack = true
		case "loc":
			selection.LOC = true
		default:
			if !IsEngineScanFact(fact) {
				return ScanSelection{}, fmt.Errorf("unknown scan fact %q", fact)
			}
			if !slices.Contains(selection.EngineFacts, fact) {
				selection.EngineFacts = append(selection.EngineFacts, fact)
			}
		}
	}
	return selection, nil
}

func IsResolvableScanFact(fact string) bool {
	return fact == "dependencies" || fact == "files" || fact == "call-graph"
}
