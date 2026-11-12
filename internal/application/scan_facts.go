package application

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
	"github.com/edwardnoaland/scryer/internal/engine"
	"github.com/edwardnoaland/scryer/internal/projectfacts"
)

func (s *Service) ScanFacts(ctx context.Context, directory string, stack, loc bool, requested []string, resolve bool, compileCommands string) (contract.Document, error) {
	var resolveFacts []string
	if resolve {
		resolveFacts = requested
	}
	return s.scanFacts(ctx, directory, stack, loc, requested, resolveFacts, compileCommands)
}

func (s *Service) scanFacts(ctx context.Context, directory string, stack, loc bool, requested, resolveFacts []string, compileCommands string) (contract.Document, error) {
	facts, err := projectfacts.Collect(ctx, directory, stack || len(requested) != 0, loc)
	if err != nil {
		return contract.Document{}, err
	}
	if len(requested) != 0 {
		facts.EngineFacts, err = s.collectEngineFacts(ctx, facts, requested, resolveFacts, compileCommands)
		if err != nil {
			return contract.Document{}, err
		}
	}
	if !stack {
		facts.Stack = nil
	}
	return contract.NewDocument("", contract.ProjectFactsOperation, facts)
}

func (s *Service) collectEngineFacts(ctx context.Context, project contract.ProjectFacts, requested, resolveFacts []string, compileCommands string) ([]contract.FactOutcome, error) {
	store, err := engine.DefaultStore()
	if err != nil {
		return nil, err
	}
	installed, err := store.List(ctx)
	if err != nil {
		return nil, err
	}
	var outcomes []contract.FactOutcome
	supported := map[string]bool{}
	relevant := false
	for _, item := range installed {
		if item.Capabilities == nil || !matchesProject(item.Capabilities, project.Stack) {
			continue
		}
		relevant = true
		var selected []string
		for _, fact := range requested {
			if item.Capabilities.SupportsFact(fact) {
				selected = append(selected, fact)
				supported[fact] = true
			} else {
				outcomes = append(outcomes, contract.FactOutcome{Fact: fact, Engine: item.Engine.ID, Status: "unsupported", Message: "engine does not declare support for this fact"})
			}
		}
		if len(selected) == 0 {
			continue
		}
		results := s.scanInstalled(ctx, item, project.Root, selected, resolveFacts, compileCommands)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		outcomes = append(outcomes, results...)
	}
	for _, fact := range requested {
		if supported[fact] {
			continue
		}
		outcome := contract.FactOutcome{Fact: fact, Status: "unsupported", Message: "no installed engine for the detected stack supports this fact"}
		if !relevant {
			outcome.Status, outcome.Message = "no-matching-engine", "no installed engine matches the detected stack"
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes, nil
}

func matchesProject(capabilities *engine.Capabilities, stack *contract.ProjectStack) bool {
	for _, language := range stack.Languages {
		if slices.ContainsFunc(capabilities.Languages, func(name string) bool { return strings.EqualFold(name, language.Language) }) {
			return true
		}
	}
	return false
}

func (s *Service) scanInstalled(ctx context.Context, installed engine.Installation, root string, selected, resolveFacts []string, compileCommands string) []contract.FactOutcome {
	resolveSelected := slices.ContainsFunc(selected, func(fact string) bool {
		return slices.Contains(resolveFacts, fact) && installed.Capabilities.ResolvesFact(fact)
	})
	if !resolveSelected || !slices.ContainsFunc(selected, func(fact string) bool { return fact != "dependencies" && installed.Capabilities.ResolvesFact(fact) }) {
		compileCommands = ""
	}
	var analyzer contract.Analyzer
	if s.NewInstalledAnalyzer != nil {
		analyzer = s.NewInstalledAnalyzer(installed)
	}
	var data struct {
		Root  string                     `json:"root"`
		Facts map[string]json.RawMessage `json:"facts"`
	}
	var err error
	if analyzer == nil {
		err = fmt.Errorf("engine execution is unavailable")
	} else {
		var document contract.Document
		document, err = analyzer.Scan(ctx, contract.ScanRequest{Repository: root, Static: !resolveSelected, Facts: selected, CompileCommands: compileCommands})
		if err == nil {
			err = json.Unmarshal(document.Data, &data)
		}
		if err == nil && data.Root != root {
			err = fmt.Errorf("engine report root does not match project")
		}
	}
	outcomes := make([]contract.FactOutcome, 0, len(selected))
	for _, fact := range selected {
		outcome := contract.FactOutcome{Fact: fact, Engine: installed.Engine.ID, Status: "collected", Data: data.Facts[fact]}
		if err != nil {
			outcome.Status, outcome.Message, outcome.Data = "failed", err.Error(), nil
		} else if len(outcome.Data) == 0 || string(outcome.Data) == "null" {
			outcome.Status, outcome.Message, outcome.Data = "failed", "engine omitted the requested fact", nil
		}
		if outcome.Status == "collected" && fact == "dependencies" && installed.Engine.ID == "cpp" {
			enriched, err := identifyCMakeRepositories(outcome.Data)
			if err != nil {
				outcome.Status, outcome.Message, outcome.Data = "failed", err.Error(), nil
			} else {
				outcome.Data = enriched
			}
		}
		if outcome.Status == "collected" && contract.IsResolvableScanFact(fact) && slices.Contains(resolveFacts, fact) && !installed.Capabilities.ResolvesFact(fact) {
			outcome.Message = "engine does not declare support for " + fact + " resolution; returned static facts"
		}
		outcomes = append(outcomes, outcome)
	}
	return outcomes
}
