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
				if !slices.Contains(app.Endpoints, endpoint) {
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
			if len(declarations) != 1 || len(registrations) != 1 {
				continue
			}
			to, registration := declarations[0], registrations[0]
			// The initial declaration collector records a bare symbol. A qualified
			// registration cannot be tied to it without module identity.
			if to.Symbol != "" && to.Symbol == registration.Symbol && to.Engine == registration.Engine && from.Engine != to.Engine {
				app.Links = append(app.Links, contract.TauriLink{
					Kind: from.Kind, Name: from.Name, From: from, To: to, Registration: &registration,
				})
			}
		case from.Kind == "event" && from.Role == "emit":
			for _, to := range index[endpointKey{"event", "listen", from.Name}] {
				if from.Engine != to.Engine {
					app.Links = append(app.Links, contract.TauriLink{Kind: from.Kind, Name: from.Name, From: from, To: to})
				}
			}
		}
	}
}
