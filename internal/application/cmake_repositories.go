package application

import (
	"encoding/json"

	"github.com/edwardnoaland/scryer/internal/repository"
)

// identifyCMakeRepositories adds hosted identities while preserving engine-supplied fields.
func identifyCMakeRepositories(payload json.RawMessage) (json.RawMessage, error) {
	var fact map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fact); err != nil || fact["cmake"] == nil {
		return payload, nil
	}
	var cmake map[string]json.RawMessage
	if err := json.Unmarshal(fact["cmake"], &cmake); err != nil || cmake["externalDependencies"] == nil {
		return payload, nil
	}
	var declarations []map[string]json.RawMessage
	if err := json.Unmarshal(cmake["externalDependencies"], &declarations); err != nil {
		return payload, nil
	}

	changed := false
	for _, declaration := range declarations {
		var command string
		if err := json.Unmarshal(declaration["command"], &command); err != nil ||
			(command != "fetchcontent_declare" && command != "externalproject_add") {
			continue
		}
		var args []string
		if err := json.Unmarshal(declaration["arguments"], &args); err != nil {
			continue
		}
		for i, arg := range args {
			if arg != "GIT_REPOSITORY" || i+1 >= len(args) {
				continue
			}
			identity, ok := repository.ParseRemote(args[i+1])
			if ok {
				encoded, err := json.Marshal(identity)
				if err != nil {
					return nil, err
				}
				declaration["repository"] = encoded
				changed = true
			}
			break
		}
	}
	if !changed {
		return payload, nil
	}
	encoded, err := json.Marshal(declarations)
	if err != nil {
		return nil, err
	}
	cmake["externalDependencies"] = encoded
	encoded, err = json.Marshal(cmake)
	if err != nil {
		return nil, err
	}
	fact["cmake"] = encoded
	return json.Marshal(fact)
}
