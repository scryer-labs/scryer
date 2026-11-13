// Package tauri aggregates app-scoped communication evidence from language engines.
package tauri

import (
	"encoding/json"
	"path"
	"slices"
	"strings"

	"github.com/edwardnoaland/scryer/internal/contract"
)

func Aggregate(outcomes []contract.FactOutcome) *contract.TauriIPC {
	result := &contract.TauriIPC{Apps: []contract.TauriApp{}}
	apps := map[string]int{}
	for _, outcome := range outcomes {
		if outcome.Fact != "ipc" || outcome.Status != "collected" {
			continue
		}
		var input struct {
			Tauri contract.TauriIPC `json:"tauri"`
		}
		if json.Unmarshal(outcome.Data, &input) != nil {
			continue
		}
		for _, observed := range input.Tauri.Apps {
			if observed.Config == "." || !validPath(observed.Config) {
				continue
			}
			index, ok := apps[observed.Config]
			if !ok {
				index = len(result.Apps)
				apps[observed.Config] = index
				result.Apps = append(result.Apps, contract.TauriApp{
					Config: observed.Config, Endpoints: []contract.TauriEndpoint{}, Links: []contract.TauriLink{},
				})
			}
			app := &result.Apps[index]
			for _, endpoint := range observed.Endpoints {
				if !validEndpoint(endpoint) {
					continue
				}
				endpoint.Engine = outcome.Engine
				if !slices.ContainsFunc(app.Endpoints, func(existing contract.TauriEndpoint) bool {
					left, right := existing.Declaration, endpoint.Declaration
					existing.Declaration = nil
					candidate := endpoint
					candidate.Declaration = nil
					return existing == candidate && (left == right || left != nil && right != nil && *left == *right)
				}) {
					app.Endpoints = append(app.Endpoints, endpoint)
				}
			}
		}
	}
	if len(result.Apps) == 0 {
		return nil
	}
	for i := range result.Apps {
		match(&result.Apps[i])
	}
	return result
}

func validPath(file string) bool {
	return file != "" && path.Clean(file) == file && file != ".." && !strings.HasPrefix(file, "../") &&
		!strings.HasPrefix(file, "/") && !strings.Contains(file, "\\")
}

func validEndpoint(endpoint contract.TauriEndpoint) bool {
	if source := endpoint.Declaration; source != nil &&
		(endpoint.Kind != "command" || endpoint.Role != "registration" || source.File == "." ||
			!validPath(source.File) || source.Line == 0 || source.Column == 0) {
		return false
	}
	if endpoint.Name == "" || endpoint.File == "." || !validPath(endpoint.File) || endpoint.Line == 0 || endpoint.Column == 0 {
		return false
	}
	switch endpoint.Kind {
	case "command":
		return endpoint.Role == "declaration" || endpoint.Role == "registration" || endpoint.Role == "invoke"
	case "event":
		return endpoint.Role == "emit" || endpoint.Role == "listen"
	}
	return false
}

type endpointKey struct {
	kind, role, name string
}

func match(app *contract.TauriApp) {
	index := map[endpointKey][]contract.TauriEndpoint{}
	for _, endpoint := range app.Endpoints {
		key := endpointKey{endpoint.Kind, endpoint.Role, endpoint.Name}
		index[key] = append(index[key], endpoint)
	}
	for _, from := range app.Endpoints {
		switch {
		case from.Kind == "command" && from.Role == "invoke":
			declarations := index[endpointKey{"command", "declaration", from.Name}]
			registrations := index[endpointKey{"command", "registration", from.Name}]
			var matches []contract.TauriLink
			for _, registration := range registrations {
				for _, to := range declarations {
					sameDeclaration := registration.ResolutionReason == "" && to.Symbol != "" && to.Symbol == registration.Symbol && to.File == registration.File && to.ModuleID == registration.ModuleID
					if evidence := registration.Declaration; evidence != nil {
						sameDeclaration = to.File == evidence.File && to.Line == evidence.Line && to.Column == evidence.Column
					}
					if sameDeclaration && to.Engine == registration.Engine && from.Engine != to.Engine {
						matches = append(matches, contract.TauriLink{
							Kind: from.Kind, Name: from.Name, From: from, To: to, Registration: &registration,
						})
					}
				}
			}
			if len(registrations) == 1 && len(matches) == 1 && (len(declarations) == 1 || matches[0].Registration.Declaration != nil) {
				app.Links = append(app.Links, matches[0])
			} else {
				app.Unmatched = append(app.Unmatched, contract.TauriUnmatched{Endpoint: from, Reason: "command-target-not-unique-or-unproven"})
			}
		case from.Kind == "event" && from.Role == "emit":
			before := len(app.Links)
			for _, to := range index[endpointKey{"event", "listen", from.Name}] {
				if from.Engine != to.Engine {
					app.Links = append(app.Links, contract.TauriLink{Kind: from.Kind, Name: from.Name, From: from, To: to})
				}
			}
			if len(app.Links) == before {
				app.Unmatched = append(app.Unmatched, contract.TauriUnmatched{Endpoint: from, Reason: "no-cross-engine-listener-evidence"})
			}
		}
	}
}
